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

/** CR-025 — one role/language row of the DB-backed prompt-template store. */
export interface PromptTemplate {
  role:
    | "story_architect"
    | "visual_director"
    | "manim_engineer"
    | "script_reviewer"
    | "remotion_engineer"
    | "remotion_visual_director";
  language: "vi" | "en";
  template_text: string;
  version: number;
  updated_at?: string;
  /** CR-027 FR84.4 — true khi nội dung trả về là bản tuỳ chỉnh đang bật. */
  from_override?: boolean;
}

/**
 * CR-027 FR84 — bản prompt do Creator tự viết, sống ở bảng riêng
 * `prompt_overrides`, tách hẳn khỏi bản gốc ship trong binary.
 *
 * `is_active` tắt thì giữ nguyên nội dung nhưng chạy bản gốc — đây là bản
 * thay thế không phá huỷ cho nút "Khôi phục mặc định" cũ, vốn xoá hẳn bản đã
 * sửa và không lấy lại được.
 */
export interface PromptOverride {
  role: PromptTemplate["role"];
  language: "vi" | "en";
  template_text: string;
  is_active: boolean;
  /** Version của bản gốc mà bản tuỳ chỉnh này được viết dựa trên (0 = không rõ). */
  based_on_version: number;
  updated_at?: string;
}

/** Đọc wording hiện tại của một vai trò (chạy lúc runtime, không hardcode nữa). */
export function getPromptTemplate(role: PromptTemplate["role"], language: "vi" | "en"): Promise<PromptTemplate> {
  return apiFetch<PromptTemplate>(`/v1/prompts/${role}?language=${language}`);
}

/** Toàn bộ template (mọi vai trò/ngôn ngữ) — cho màn admin sửa prompt. */
export async function listPromptTemplates(): Promise<PromptTemplate[]> {
  const result = await apiFetch<{ templates: PromptTemplate[] }>("/v1/admin/prompts");
  return result.templates;
}

/** Lưu nội dung một template mới, tăng version (CR-025). */
export function updatePromptTemplate(
  role: PromptTemplate["role"],
  language: "vi" | "en",
  templateText: string,
): Promise<PromptTemplate> {
  return apiFetch<PromptTemplate>(`/v1/admin/prompts/${role}`, {
    method: "PUT",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ language, template_text: templateText }),
  });
}

/** CR-027 — mọi bản tuỳ chỉnh của Creator, cho màn admin hai tầng. */
export async function listPromptOverrides(): Promise<PromptOverride[]> {
  const result = await apiFetch<{ overrides: PromptOverride[] }>("/v1/admin/prompt-overrides");
  return result.overrides;
}

/** Lưu bản tuỳ chỉnh. Sửa nội dung KHÔNG tự bật một bản đang tắt. */
export function savePromptOverride(
  role: PromptTemplate["role"],
  language: "vi" | "en",
  templateText: string,
): Promise<PromptOverride> {
  return apiFetch<PromptOverride>(`/v1/admin/prompt-overrides/${role}`, {
    method: "PUT",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ language, template_text: templateText }),
  });
}

/**
 * Bật/tắt bản tuỳ chỉnh (FR84.5).
 *
 * Tắt là quay về bản gốc mà KHÔNG mất nội dung đã viết — bật lại là có
 * nguyên. Khác hẳn `resetPromptTemplate` bên dưới, vốn xoá vĩnh viễn.
 */
export function setPromptOverrideActive(
  role: PromptTemplate["role"],
  language: "vi" | "en",
  active: boolean,
): Promise<PromptOverride> {
  return apiFetch<PromptOverride>(`/v1/admin/prompt-overrides/${role}/active`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ language, active }),
  });
}

/** Xoá hẳn bản tuỳ chỉnh, trả quyền cho bản gốc. */
export async function deletePromptOverride(
  role: PromptTemplate["role"],
  language: "vi" | "en",
): Promise<void> {
  await apiFetch<undefined>(`/v1/admin/prompt-overrides/${role}?language=${language}`, {
    method: "DELETE",
  });
}

/**
 * CR-025 legacy — khôi phục prompt mặc định, xoá bản người vận hành đã sửa.
 *
 * Seeding ở Orchestrator là insert-if-absent — nó cố ý KHÔNG đè lên bản sửa
 * tay khi service khởi động lại. Nên khi prompt trong source được cải tiến,
 * đây là đường duy nhất để bản mới vào được một DB đã bootstrap, và nó xảy ra
 * vì người vận hành bấm nút, không phải vì một tiến trình vừa restart.
 */
export function resetPromptTemplate(
  role: PromptTemplate["role"],
  language: "vi" | "en",
): Promise<PromptTemplate> {
  return apiFetch<PromptTemplate>(`/v1/admin/prompts/${role}/reset?language=${language}`, {
    method: "POST",
  });
}

/**
 * CR-027 FR79.4 — nút "Chạy bằng AI" có nơi nào để gọi không. Hỏi trước khi
 * vẽ nút: một nút bấm vào là lỗi tệ hơn một nút không có kèm lời giải thích.
 */
export type LlmStatus = {
  enabled: boolean;
  provider: string;
  reason?: string;
};

export function getLlmStatus(): Promise<LlmStatus> {
  return apiFetch<LlmStatus>("/v1/llm/status");
}

/**
 * CR-027 FR79 — cách làm bước 1, theo đúng hai giá trị server nhận. Kiểu nằm ở
 * đây vì đây là hợp đồng trên đường truyền; ProjectDraftContext export lại nó
 * kèm ý nghĩa nghiệp vụ.
 */
export type AuthoringMode = "manual" | "ai";

/** Bốn bước của pipeline soạn kịch bản, theo đúng tên server dùng (FR78.5). */
export type AuthoringStep = "story" | "storyboard" | "code" | "review";

/**
 * CR-027 FR78 — kết quả một lượt chạy bằng AI. `save_error` có nghĩa là đã
 * sinh được nội dung nhưng chưa lưu được (ví dụ project đã khoá vì đang
 * render): nội dung vẫn trả về, vì token đã bị tính tiền rồi.
 */
export type GeneratedStep = {
  step: string;
  role: string;
  content: string;
  provider: string;
  usage: {
    model: string;
    PromptTokens?: number;
    CompletionTokens?: number;
  };
  save_error?: string;
};

/**
 * CR-027 FR78.1 — chạy một bước bằng API: server tự render prompt (cùng một
 * hàm với nút Copy), gọi provider, lưu kết quả, trả nội dung về.
 *
 * Đây là lựa chọn thứ hai, không phải bản thay thế: nút Copy prompt vẫn là
 * đường đi khi chưa có key, hết số dư, hoặc Creator muốn dùng AI khác.
 */
export function generateAuthoringStep(
  projectId: string,
  step: AuthoringStep,
  lintResults?: string,
): Promise<GeneratedStep> {
  const query = lintResults ? `?lint_results=${encodeURIComponent(lintResults)}` : "";
  return apiFetch<GeneratedStep>(`/v1/projects/${projectId}/authoring/${step}/generate${query}`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
  });
}

/** CR-025 bước 1 — lưu dàn ý câu chuyện (Story Architect) Creator dán vào. */
export async function saveAuthoringStory(
  projectId: string,
  content: string,
  topic: string,
): Promise<void> {
  await apiFetch<undefined>(`/v1/projects/${projectId}/authoring/story`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ content, topic }),
  });
}

/** CR-025 bước 2 — lưu storyboard (Visual Director) Creator dán vào. */
export async function saveAuthoringStoryboard(projectId: string, content: string): Promise<void> {
  await apiFetch<undefined>(`/v1/projects/${projectId}/authoring/storyboard`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ content }),
  });
}

/** CR-025 bước 3 — lưu code Manim (Manim Engineer) Creator dán vào. */
export async function saveAuthoringCode(projectId: string, content: string): Promise<void> {
  await apiFetch<undefined>(`/v1/projects/${projectId}/authoring/code`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ content }),
  });
}

/** CR-025 bước 4 — lưu verdict PASS/REVISE (Script Reviewer) Creator dán vào. */
export async function saveAuthoringReview(projectId: string, content: string): Promise<void> {
  await apiFetch<undefined>(`/v1/projects/${projectId}/authoring/review`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ content }),
  });
}

/**
 * Cả bốn kết quả đã lưu của pipeline soạn kịch bản (CR-025) — dùng để nạp
 * lại trạng thái khi Creator tải lại trang hoặc quay lại một bước trước đó,
 * thay vì chỉ dựa vào draft ở client (localStorage có thể đã mất khi mở lại
 * bằng một trình duyệt/máy khác dùng chung project_id).
 */
export interface AuthoringState {
  /**
   * CR-027 FR79 — cách làm bước 1 đã lưu cho project này: "manual" hoặc "ai".
   * Server luôn trả một trong hai (project cũ đọc ra "manual"), nhưng để
   * optional để một orchestrator chưa nâng cấp không làm vỡ phần rehydrate.
   */
  mode?: AuthoringMode;
  /** CR-027 D0 — "" cho mọi project tạo trước CR-027. */
  topic: string;
  story: string;
  storyboard: string;
  code: string;
  review: string;
}

export function getAuthoringState(projectId: string): Promise<AuthoringState> {
  return apiFetch<AuthoringState>(`/v1/projects/${projectId}/authoring`);
}

/**
 * CR-027 FR79 — lưu cách làm bước 1 cho project này.
 *
 * Nằm ở server, không chỉ trong localStorage: lựa chọn này áp cho cả 4 tab và
 * một project có thể được mở lại ở bất cứ tab nào, từ trình duyệt khác hoặc
 * sau khi stack restart. PUT nên bấm qua lại nhiều lần cũng chỉ là một giá trị.
 */
export async function saveAuthoringMode(projectId: string, mode: AuthoringMode): Promise<void> {
  await apiFetch<undefined>(`/v1/projects/${projectId}/authoring/mode`, {
    method: "PUT",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ mode }),
  });
}

/** CR-028 FR85 — một project khác (cùng ngôn ngữ) có chủ đề trùng sau khi chuẩn hoá. */
export interface SimilarProject {
  projectId: string;
  topic: string;
  status: string;
  createdAt: string;
}

interface similarProjectsWire {
  project_id: string;
  topic: string;
  status: string;
  created_at: string;
}

function fromWireSimilarProjects(wire: similarProjectsWire[]): SimilarProject[] {
  return wire.map((p) => ({
    projectId: p.project_id,
    topic: p.topic,
    status: p.status,
    createdAt: p.created_at,
  }));
}

/**
 * CR-028 FR83.1 — tạo hàng project ngay khi Creator gõ xong chủ đề (bước 1),
 * thay vì đợi tới POST /v1/sagas/render. `projectId` là id đã sinh sẵn ở
 * client (ProjectDraftContext) — gửi lên để mọi endpoint authoring đã và sẽ
 * gọi với id đó vẫn trỏ đúng một project, không đổi kiến trúc id ở client.
 */
export async function createProjectDraft(
  projectId: string,
  topic: string,
  contentLanguage: "vi" | "en",
): Promise<{ similarProjects: SimilarProject[] }> {
  const res = await apiFetch<{ project_id: string; similar_projects: similarProjectsWire[] }>(
    "/v1/projects",
    {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ project_id: projectId, topic, content_language: contentLanguage }),
    },
  );
  return { similarProjects: fromWireSimilarProjects(res.similar_projects ?? []) };
}

/**
 * CR-028 FR83.2 — Creator quay lại bước 1 và sửa chủ đề của draft đã tạo.
 * 409 nếu render đã bắt đầu (FR84.2 — cùng khoá với authoring saves).
 */
export async function updateProjectTopic(
  projectId: string,
  topic: string,
): Promise<{ similarProjects: SimilarProject[] }> {
  const res = await apiFetch<{ similar_projects: similarProjectsWire[] }>(
    `/v1/projects/${projectId}/topic`,
    {
      method: "PATCH",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ topic }),
    },
  );
  return { similarProjects: fromWireSimilarProjects(res.similar_projects ?? []) };
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
