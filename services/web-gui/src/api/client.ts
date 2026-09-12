import type {
  Project,
  ProgressMessage,
  ProjectSummary,
  PublishMetadata,
  YoutubeAccount,
  YoutubeApp,
  RenderInput,
  SagaStartedResponse,
  Voice,
  VideoFormat,
} from "../types";

const GATEWAY_URL = import.meta.env.VITE_API_BASE_URL;

export const GENERIC_CONNECTION_ERROR = "Không thể kết nối máy chủ, thử lại sau";

export class ApiError extends Error {
  /**
   * Mã lỗi đọc được bằng máy, khi máy chủ gửi kèm. Publish trả 409 cho cả
   * "project sai trạng thái" lẫn "QC có lỗi chặn" (CR-021 FR61.3), và chỉ cái
   * thứ hai mới có đường đi tiếp — dò theo câu chữ sẽ vỡ ngay khi đổi từ ngữ.
   */
  readonly code?: string;

  constructor(message: string, code?: string) {
    super(message);
    this.code = code;
  }
}

/** CR-021 FR61.3 — mã 409 mà `acknowledge_qc: true` đi qua được. */
export const ERROR_CODE_QC_BLOCKED = "qc_blocked";

/** Duyệt dàn ý, cho Saga chạy tiếp (CR-024 FR69.2). */
export async function approveOutline(projectId: string): Promise<void> {
  await postDecision(`${GATEWAY_URL}/v1/projects/${projectId}/approve`);
}

/** Từ chối dàn ý — Saga kết thúc để Creator quay lại sửa script (FR69.3). */
export async function rejectOutline(projectId: string): Promise<void> {
  await postDecision(`${GATEWAY_URL}/v1/projects/${projectId}/reject`);
}

/**
 * Sửa một câu lời thoại ngay tại màn duyệt (FR70).
 *
 * Server có thể từ chối với 422 khi câu này không truy ngược được về đúng một
 * chỗ trong script — lời thoại sinh trong vòng lặp hoặc bằng f-string. Thông
 * báo kèm theo nói rõ lý do, nên hiển thị nguyên văn thay vì nuốt đi.
 */
export async function editNarration(
  projectId: string,
  sceneIndex: number,
  narrationText: string,
): Promise<void> {
  const response = await fetch(`${GATEWAY_URL}/v1/projects/${projectId}/narration`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ scene_index: sceneIndex, narration_text: narrationText }),
  });
  if (!response.ok) {
    const body = await response.json().catch(() => null);
    throw new ApiError(body?.error ?? "Không sửa được lời thoại");
  }
}

async function postDecision(url: string): Promise<void> {
  const response = await fetch(url, { method: "POST" });
  if (!response.ok) {
    const body = await response.json().catch(() => null);
    throw new ApiError(body?.error ?? GENERIC_CONNECTION_ERROR);
  }
}

/** Các hình dạng video Creator chọn được (CR-019 FR51.3). */
export async function fetchVideoFormats(): Promise<VideoFormat[]> {
  const response = await fetch(`${GATEWAY_URL}/v1/formats`);
  if (!response.ok) return [];
  const body = await response.json();
  return Array.isArray(body?.formats) ? (body.formats as VideoFormat[]) : [];
}

/**
 * Tốc độ đọc **đo được** của từng giọng (CR-016 FR43.2).
 *
 * Giọng chưa đủ mẫu không có mặt trong map — bên gọi rơi về hằng số theo ngôn
 * ngữ, và đó là câu trả lời trung thực hơn một con số độ tin cậy thấp.
 */
export async function fetchVoiceCalibration(): Promise<Record<string, number>> {
  const response = await fetch(`${GATEWAY_URL}/v1/voice-calibration`);
  if (!response.ok) return {};
  const body = await response.json();
  return typeof body?.words_per_minute === "object" && body.words_per_minute !== null
    ? (body.words_per_minute as Record<string, number>)
    : {};
}

export function getProjectVideoUrl(projectId: string): string {
  return `${GATEWAY_URL}/v1/projects/${projectId}/video`;
}

export function getProjectThumbnailUrl(projectId: string): string {
  return `${GATEWAY_URL}/v1/projects/${projectId}/thumbnail`;
}

/** CR-007 D7 — streams one generated vertical clip from the shared volume. */
export function getProjectClipUrl(projectId: string, name: string, preset: string): string {
  return `${GATEWAY_URL}/v1/projects/${projectId}/clips/${encodeURIComponent(name)}/${preset}`;
}

export interface ThumbnailInfo {
  exists: boolean;
  thumbnail_path: string | null;
  /** True when the image is the frame Video Assembly extracted, not the
   * Creator's own upload (CR-006 FR16). */
  auto_generated?: boolean;
}

export function getThumbnailInfo(projectId: string): Promise<ThumbnailInfo> {
  return apiFetch<ThumbnailInfo>(`/v1/projects/${projectId}/thumbnail/info`);
}

export interface ThumbnailUploadResult {
  thumbnail_path: string;
}

export async function uploadThumbnail(projectId: string, file: File): Promise<ThumbnailUploadResult> {
  const formData = new FormData();
  formData.append("thumbnail", file);
  return apiFetch<ThumbnailUploadResult>(`/v1/projects/${projectId}/thumbnail`, {
    method: "POST",
    body: formData,
  });
}

export function getProjectMusicUrl(projectId: string): string {
  return `${GATEWAY_URL}/v1/projects/${projectId}/music`;
}

export interface MusicInfo {
  exists: boolean;
  background_music_path: string | null;
}

export function getMusicInfo(projectId: string): Promise<MusicInfo> {
  return apiFetch<MusicInfo>(`/v1/projects/${projectId}/music/info`);
}

export interface MusicUploadResult {
  background_music_path: string;
}

export async function uploadMusic(projectId: string, file: File): Promise<MusicUploadResult> {
  const formData = new FormData();
  formData.append("music", file);
  return apiFetch<MusicUploadResult>(`/v1/projects/${projectId}/music`, {
    method: "POST",
    body: formData,
  });
}

async function parseError(response: Response): Promise<{ message: string; code?: string }> {
  try {
    const body = await response.json();
    const code = typeof body?.code === "string" ? body.code : undefined;
    if (typeof body?.error === "string") return { message: body.error, code };
    return {
      message: body?.error?.message ?? body?.error_message ?? GENERIC_CONNECTION_ERROR,
      code,
    };
  } catch {
    return { message: GENERIC_CONNECTION_ERROR };
  }
}

async function apiFetch<T>(path: string, init?: RequestInit): Promise<T> {
  let response: Response;
  try {
    response = await fetch(`${GATEWAY_URL}${path}`, init);
  } catch {
    throw new ApiError(GENERIC_CONNECTION_ERROR);
  }
  if (!response.ok) {
    const { message, code } = await parseError(response);
    throw new ApiError(message, code);
  }
  if (response.status === 204) {
    return undefined as T;
  }
  return response.json() as Promise<T>;
}

export function startRenderSaga(input: RenderInput): Promise<SagaStartedResponse> {
  return apiFetch<SagaStartedResponse>("/v1/sagas/render", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(input),
  });
}

export function listVoices(): Promise<Voice[]> {
  return apiFetch<Voice[]>("/v1/voices");
}

export function getProject(id: string): Promise<Project> {
  return apiFetch<Project>(`/v1/projects/${id}`);
}

export function retryProject(id: string): Promise<SagaStartedResponse> {
  return apiFetch<SagaStartedResponse>(`/v1/projects/${id}/retry`, { method: "POST" });
}

export async function listProjects(): Promise<ProjectSummary[]> {
  const result = await apiFetch<{ projects: ProjectSummary[] }>("/v1/projects");
  return result.projects;
}

export async function deleteProject(id: string): Promise<void> {
  await apiFetch<undefined>(`/v1/projects/${id}`, { method: "DELETE" });
}

export function startPublishSaga(
  id: string,
  metadata: PublishMetadata,
  /**
   * CR-021 FR61.3 — chỉ gửi `true` sau khi Creator đã đọc báo cáo QC và chủ
   * động chọn đăng bất chấp lỗi chặn. Không bao giờ gửi ở lần bấm đầu: bỏ qua
   * phải là một hành động có ý thức, và máy chủ ghi lại lần bỏ qua đó.
   */
  acknowledgeQC = false,
): Promise<SagaStartedResponse> {
  return apiFetch<SagaStartedResponse>("/v1/sagas/publish", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({
      project_id: id,
      ...metadata,
      ...(acknowledgeQC ? { acknowledge_qc: true } : {}),
    }),
  });
}

/** CR-021 — một phát hiện QC, kèm mốc thời gian để tua tới chỗ đó (FR59.6). */
export interface QCFinding {
  rule: string;
  severity: "blocking" | "warning";
  message: string;
  timestamp_seconds: number;
}

export interface QCReport {
  project_id: string;
  status: "passed" | "has_findings" | "not_scored";
  reason: string | null;
  findings: QCFinding[];
  created_at: string;
  overridden_at?: string;
}

/**
 * Báo cáo QC của project (CR-021 FR61.1/FR61.2).
 *
 * Máy chủ luôn trả 200: chưa chấm thì `status: "not_scored"` với danh sách
 * rỗng, không phải 404 — một cổng chưa chạy không phải một lỗi.
 */
export function getQCReport(id: string): Promise<QCReport> {
  return apiFetch<QCReport>(`/v1/projects/${id}/qc-report`);
}

export interface SuggestedMetadata {
  title: string;
  description: string;
  tags: string[];
}

export function suggestPublishMetadata(id: string): Promise<SuggestedMetadata> {
  return apiFetch<SuggestedMetadata>(`/v1/projects/${id}/suggest-metadata`, { method: "POST" });
}

export interface SuggestShortScriptInput {
  topic: string;
  language: "vi" | "en";
  /** Ngữ cảnh tuỳ chọn — script dài đã có, dùng để rút chủ đề (CR-026 FR71.1). */
  source_script_content?: string;
}

/** CR-026 FR71 — soạn nháp script Shorts/TikTok bằng AI nội bộ (Ollama). */
export function suggestShortScript(input: SuggestShortScriptInput): Promise<{ script_content: string }> {
  return apiFetch<{ script_content: string }>("/v1/short-script-suggestions", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(input),
  });
}

export function getYoutubeAuthStartUrl(projectId: string, clientId?: string): string {
  const params = new URLSearchParams({ state: projectId });
  // Omitted when there is only one configured client — the Publisher picks
  // it, so the Creator never sees a one-option chooser (CR-012 FR34.2).
  if (clientId) params.set("app", clientId);
  return `${GATEWAY_URL}/v1/auth/youtube/start?${params.toString()}`;
}

export async function getYoutubeConnectionStatus(): Promise<boolean> {
  const result = await apiFetch<{ connected: boolean }>("/v1/auth/youtube/status");
  return result.connected;
}

export function listYoutubeApps(): Promise<YoutubeApp[]> {
  return apiFetch<YoutubeApp[]>("/v1/auth/youtube/apps");
}

export function listYoutubeAccounts(): Promise<YoutubeAccount[]> {
  return apiFetch<YoutubeAccount[]>("/v1/auth/youtube/accounts");
}

export async function disconnectYoutubeAccount(channelId: string): Promise<void> {
  await apiFetch<undefined>(`/v1/auth/youtube/accounts/${encodeURIComponent(channelId)}`, {
    method: "DELETE",
  });
}

export function makeYoutubeAccountDefault(channelId: string): Promise<YoutubeAccount> {
  return apiFetch<YoutubeAccount>(
    `/v1/auth/youtube/accounts/${encodeURIComponent(channelId)}/default`,
    { method: "POST" },
  );
}

export interface YoutubeAuthCallbackResult {
  connected: boolean;
  error: string | null;
  state: string | null;
  channel_id: string | null;
  channel_title: string | null;
}

export async function completeYoutubeAuthCallback(
  code: string,
  state: string | null,
): Promise<YoutubeAuthCallbackResult> {
  const params = new URLSearchParams({ code });
  if (state) params.set("state", state);
  let response: Response;
  try {
    response = await fetch(`${GATEWAY_URL}/v1/auth/youtube/callback?${params.toString()}`);
  } catch {
    throw new ApiError(GENERIC_CONNECTION_ERROR);
  }
  return response.json() as Promise<YoutubeAuthCallbackResult>;
}

export function subscribeProgress(
  projectId: string,
  onMessage: (msg: ProgressMessage) => void,
): () => void {
  const source = new EventSource(`${GATEWAY_URL}/v1/progress/${projectId}`);
  source.onmessage = (event) => {
    onMessage(JSON.parse(event.data) as ProgressMessage);
  };
  return () => source.close();
}
