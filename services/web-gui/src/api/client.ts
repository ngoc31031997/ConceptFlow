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

export async function uploadThumbnail(
  projectId: string,
  file: File,
  onProgress?: (loaded: number, total: number) => void,
): Promise<ThumbnailUploadResult> {
  const formData = new FormData();
  formData.append("thumbnail", file);
  if (onProgress && typeof XMLHttpRequest !== "undefined") {
    return uploadWithProgress<ThumbnailUploadResult>(`/v1/projects/${projectId}/thumbnail`, formData, onProgress);
  }
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
  // 202 (a job accepted) carries no body either.
  if (response.status === 204 || response.status === 202) {
    return undefined as T;
  }
  return response.json() as Promise<T>;
}

/** POST multipart bằng XHR — fetch chưa báo được số byte đã gửi (FR116.2, tải ảnh lên). */
function uploadWithProgress<T>(
  path: string,
  body: FormData,
  onProgress: (loaded: number, total: number) => void,
): Promise<T> {
  return new Promise<T>((resolve, reject) => {
    const xhr = new XMLHttpRequest();
    xhr.open("POST", `${GATEWAY_URL}${path}`);
    xhr.upload.onprogress = (e) => {
      if (e.lengthComputable) onProgress(e.loaded, e.total);
    };
    xhr.onerror = () => reject(new ApiError(GENERIC_CONNECTION_ERROR));
    xhr.onload = () => {
      let parsed: unknown;
      try {
        parsed = xhr.responseText ? JSON.parse(xhr.responseText) : undefined;
      } catch {
        parsed = undefined;
      }
      if (xhr.status >= 200 && xhr.status < 300) {
        resolve(parsed as T);
        return;
      }
      const err = (parsed as { message?: string; error?: string } | undefined) ?? {};
      reject(new ApiError(err.message ?? err.error ?? GENERIC_CONNECTION_ERROR, err.error));
    };
    xhr.send(body);
  });
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

/** One row of the project's project_errors trace. */
export interface ProjectErrorEntry {
  at: string;
  source: string;
  step?: string;
  kind?: string;
  provider?: string;
  message: string;
  detail?: string;
  partial_chars?: number;
  elapsed_seconds?: number;
  usage?: { model?: string; prompt_tokens: number; completion_tokens: number; reasoning_tokens: number };
}

export function listProjectErrors(id: string): Promise<ProjectErrorEntry[]> {
  return apiFetch<ProjectErrorEntry[]>(`/v1/projects/${id}/errors`);
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

/** Id do GUI tự cấp cho một lượt gọi chạy lâu để poll tiến độ trong lúc request còn mở (FR116.3). */
export function newOperationId(): string {
  return globalThis.crypto?.randomUUID?.() ?? `op-${Date.now()}-${Math.random().toString(36).slice(2)}`;
}

const OPERATION_ID_HEADER = "X-Operation-Id";

export function suggestPublishMetadata(id: string, operationId?: string): Promise<SuggestedMetadata> {
  return apiFetch<SuggestedMetadata>(`/v1/projects/${id}/suggest-metadata`, {
    method: "POST",
    ...(operationId ? { headers: { [OPERATION_ID_HEADER]: operationId } } : {}),
  });
}

export interface SuggestShortScriptInput {
  topic: string;
  language: "vi" | "en";
  /** Ngữ cảnh tuỳ chọn — script dài đã có, dùng để rút chủ đề (CR-026 FR71.1). */
  source_script_content?: string;
}

/** CR-026 FR71 — soạn nháp script Shorts/TikTok bằng AI nội bộ (Ollama). */
export function suggestShortScript(
  input: SuggestShortScriptInput,
  operationId?: string,
): Promise<{ script_content: string }> {
  return apiFetch<{ script_content: string }>("/v1/short-script-suggestions", {
    method: "POST",
    headers: { "Content-Type": "application/json", ...(operationId ? { [OPERATION_ID_HEADER]: operationId } : {}) },
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

export type PromptRole =
  | "story_architect"
  | "visual_director"
  | "manim_engineer"
  | "remotion_engineer"
  // CR-039 — luồng "Chạy bằng AI" có prompt riêng (storyboard xuất JSON, code
  // chỉ viết các hàm shot); luồng Copy giữ bốn vai trò trên nguyên vẹn.
  | "visual_director_ai"
  | "manim_engineer_ai"
  | "remotion_engineer_ai"
  // CR-040 FR113 — prompts that used to be assembled in the browser.
  | "manim_adjust"
  | "remotion_adjust"
  | "short_script"
  | "thumbnail_design";

/**
 * CR-031 — một dòng trong thư viện prompt. Mỗi vai trò có một danh sách; tại
 * một thời điểm chỉ MỘT dòng `is_active` và đó là dòng pipeline chạy.
 *
 * `is_system` là bản mặc định đi kèm hệ thống: chỉ xem và copy được, không
 * sửa/xóa. Mọi dòng khác do người dùng tạo. Không có version, không có ngôn
 * ngữ — ngôn ngữ lời thoại do `{{narration_language_rule}}` quyết định.
 */
export interface Prompt {
  id: string;
  role: PromptRole;
  name: string;
  template_text: string;
  is_system: boolean;
  is_active: boolean;
  created_at?: string;
  updated_at?: string;
}

/** Prompt đang chạy của một vai trò (wizard đọc lúc runtime, không hardcode). */
export function getPromptTemplate(role: PromptRole): Promise<Prompt> {
  return apiFetch<Prompt>(`/v1/prompts/${role}`);
}

/**
 * CR-040 FR113 — what the caller has in hand but has not saved. Every field is
 * optional; the server fills a blank one with the placeholder the browser used
 * to show.
 */
export interface PromptRenderInput {
  role: PromptRole;
  language: "vi" | "en";
  topic?: string;
  /** The Creator's existing code, for `manim_adjust` / `remotion_adjust`. */
  script?: string;
  previous_output?: string;
  subtitle_mode?: string;
  subtitle_font_size?: string;
  subtitle_position?: string;
  format_id?: string;
  format_version?: number;
  voice_id?: string;
}

export interface RenderedPromptResult {
  prompt: string;
  prompt_id: string;
  prompt_name: string;
  is_system: boolean;
}

/** The prompt of one library role with every {{variable}} substituted by the server. */
export function renderPrompt(input: PromptRenderInput): Promise<RenderedPromptResult> {
  return apiFetch<RenderedPromptResult>("/v1/prompt-renders", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(input),
  });
}

/** CR-040 FR113 — starter scripts and insertable snippets, served by authoring-service. */
export interface ScriptTemplates {
  starter_script: Record<"vi" | "en", string>;
  hook_snippet: Record<"vi" | "en", string>;
  end_screen_snippet: Record<"vi" | "en", string>;
}

export function getScriptTemplates(): Promise<ScriptTemplates> {
  return apiFetch<ScriptTemplates>("/v1/script-templates");
}

/** Toàn bộ thư viện prompt (mọi vai trò) cho màn cài đặt. */
export async function listPrompts(): Promise<Prompt[]> {
  const result = await apiFetch<{ prompts: Prompt[] }>("/v1/admin/prompts");
  return result.prompts;
}

const JSON_HEADERS = { "Content-Type": "application/json" };

/** Tạo prompt mới của người dùng (không tự bật). */
export function createPrompt(role: PromptRole, name: string, templateText: string): Promise<Prompt> {
  return apiFetch<Prompt>("/v1/admin/prompts", {
    method: "POST",
    headers: JSON_HEADERS,
    body: JSON.stringify({ role, name, template_text: templateText }),
  });
}

/** Copy một dòng (kể cả dòng hệ thống) thành dòng mới của người dùng. */
export function copyPrompt(id: string): Promise<Prompt> {
  return apiFetch<Prompt>(`/v1/admin/prompts/${id}/copy`, { method: "POST" });
}

/** Sửa prompt của người dùng. Dòng hệ thống trả 403. */
export function updatePrompt(id: string, name: string, templateText: string): Promise<Prompt> {
  return apiFetch<Prompt>(`/v1/admin/prompts/${id}`, {
    method: "PUT",
    headers: JSON_HEADERS,
    body: JSON.stringify({ name, template_text: templateText }),
  });
}

/** Bật một dòng làm prompt chạy của vai trò — dòng đang bật trước đó tự tắt. */
export function activatePrompt(id: string): Promise<Prompt> {
  return apiFetch<Prompt>(`/v1/admin/prompts/${id}/activate`, { method: "POST" });
}

/** Xóa prompt của người dùng. Xóa dòng đang bật thì vai trò quay về bản hệ thống. */
export async function deletePrompt(id: string): Promise<void> {
  await apiFetch<undefined>(`/v1/admin/prompts/${id}`, { method: "DELETE" });
}

/** CR-041 — một kiểu video mà prompt Biên kịch có thể được bảo dựng. */
export interface VideoArchetype {
  id: string;
  /** Mã ngắn (A, B, ...) — Creator gõ "kiểu: B" vào chủ đề để ép kiểu. */
  code: string;
  name: string;
  /** Chủ đề nào hợp — model chọn kiểu dựa vào đây. */
  when_to_use: string;
  /** Cách gán kiểu này vào các beat của format đang chọn. */
  playbook: string;
  is_system: boolean;
}

export interface VideoArchetypeInput {
  /** Để trống khi tạo mới thì server lấy chữ cái còn trống kế tiếp. */
  code: string;
  name: string;
  when_to_use: string;
  playbook: string;
}

export async function listVideoArchetypes(): Promise<VideoArchetype[]> {
  const result = await apiFetch<{ video_archetypes: VideoArchetype[] }>("/v1/video-archetypes");
  return result.video_archetypes;
}

export function createVideoArchetype(input: VideoArchetypeInput): Promise<VideoArchetype> {
  return apiFetch<VideoArchetype>("/v1/admin/video-archetypes", {
    method: "POST",
    headers: JSON_HEADERS,
    body: JSON.stringify(input),
  });
}

/** Copy một dòng (kể cả dòng hệ thống) thành dòng của người dùng. */
export function copyVideoArchetype(id: string): Promise<VideoArchetype> {
  return apiFetch<VideoArchetype>(`/v1/admin/video-archetypes/${id}/copy`, { method: "POST" });
}

/** Dòng hệ thống trả 403. */
export function updateVideoArchetype(id: string, input: VideoArchetypeInput): Promise<VideoArchetype> {
  return apiFetch<VideoArchetype>(`/v1/admin/video-archetypes/${id}`, {
    method: "PUT",
    headers: JSON_HEADERS,
    body: JSON.stringify(input),
  });
}

export async function deleteVideoArchetype(id: string): Promise<void> {
  await apiFetch<undefined>(`/v1/admin/video-archetypes/${id}`, { method: "DELETE" });
}

/**
 * CR-027 FR79.4 — nút "Chạy bằng AI" có nơi nào để gọi không. Hỏi trước khi
 * vẽ nút: một nút bấm vào là lỗi tệ hơn một nút không có kèm lời giải thích.
 */
/**
 * Một model trong danh mục Hive máy chủ cho phép chọn (model-per-step, tiếp
 * theo CR-027) — `id` là chuỗi gửi thẳng cho Hive, `label` là tên hiển thị.
 * "" luôn là một lựa chọn hợp lệ, nghĩa là "dùng mặc định máy chủ".
 */
export type AuthoringModelOption = {
  id: string;
  label: string;
};

export type LlmStatus = {
  enabled: boolean;
  provider: string;
  reason?: string;
  /** Danh mục model cho picker ở bước 1 — rỗng khi chế độ AI chưa khả dụng. */
  models?: AuthoringModelOption[];
  /** Model cụ thể mà lựa chọn rỗng ("") được máy chủ quy về. */
  default_model?: string;
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

/**
 * Ba bước của pipeline soạn kịch bản, theo đúng tên server dùng (FR78.5).
 *
 * CR-030 — bước "review" (Script Reviewer) đã bị bỏ hẳn khỏi sản phẩm; server
 * cũng không còn nhận nó nữa.
 */
export type AuthoringStep = "story" | "storyboard" | "code";

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
  /**
   * CR-039 — chỉ bước 1c. `check_failed`: code đã được lưu nhưng vẫn không qua
   * kiểm tra biên dịch sau `repair_rounds` vòng sửa; `diagnostics` là danh sách
   * lỗi để Creator tự sửa. `model_calls` là số lượt gọi model của cả lượt chạy.
   */
  check_failed?: boolean;
  diagnostics?: string[];
  repair_rounds?: number;
  warnings?: string[];
  model_calls?: number;
};

/**
 * CR-027 FR78.1 — chạy một bước bằng API: server tự render prompt (cùng một
 * hàm với nút Copy), gọi provider, lưu kết quả, trả nội dung về.
 *
 * Đây là lựa chọn thứ hai, không phải bản thay thế: nút Sao chÃ©p prompt vẫn là
 * đường đi khi chưa có key, hết số dư, hoặc Creator muốn dùng AI khác.
 */
export function generateAuthoringStep(projectId: string, step: AuthoringStep): Promise<GeneratedStep> {
  return apiFetch<GeneratedStep>(`/v1/projects/${projectId}/authoring/${step}/generate`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
  });
}

/**
 * Chuỗi các bước 1a/1b/1c chạy ở server, không phụ thuộc trình duyệt: POST
 * bắt đầu (trả về ngay), GET cho biết đang chạy hay kết cục lần chạy gần nhất.
 */
export interface AuthoringChainState {
  running: boolean;
  steps: AuthoringStep[];
  current_index: number;
  finished: boolean;
  /** Lý do dừng (đã dịch sẵn cho Creator) và bước dừng. */
  error?: string;
  error_step?: AuthoringStep;
  /** Dừng nhưng không phải lỗi: code còn lỗi biên dịch, hoặc không lưu được. */
  note?: string;
  /** Creator bấm Dừng (hoặc dự án bị xoá): không phải lỗi, `error` để trống. */
  cancelled?: boolean;
  started_at?: string;
  finished_at?: string;
}

export function startAuthoringChain(projectId: string, steps: AuthoringStep[]): Promise<void> {
  return apiFetch<void>(`/v1/projects/${projectId}/authoring/chain`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ steps }),
  });
}

/**
 * Dừng chuỗi đang chạy: server huỷ lời gọi model đang dở nên nhà cung cấp
 * ngừng sinh (và tính) thêm token. 404 nếu không có chuỗi nào đang chạy.
 */
export async function cancelAuthoringChain(projectId: string): Promise<void> {
  await apiFetch<undefined>(`/v1/projects/${projectId}/authoring/chain`, { method: "DELETE" });
}

export function getAuthoringChain(projectId: string): Promise<AuthoringChainState> {
  return apiFetch<AuthoringChainState>(`/v1/projects/${projectId}/authoring/chain`);
}

/** Tiến độ sống của một lượt chạy AI (phản hồi streaming từ Hive). */
export interface AuthoringProgress {
  running: boolean;
  phase: "idle" | "waiting" | "reasoning" | "writing" | "layout" | "cast" | "chunks" | "merge" | "check" | "repair";
  reasoning_chars: number;
  content_chars: number;
  elapsed_seconds: number;
  // CR-039 — tiến độ riêng của bước 1c: số lô đã xong / tổng, vòng sửa hiện tại / tối đa.
  chunks_done?: number;
  chunks_total?: number;
  repair_round?: number;
  repair_max?: number;
}

export function getAuthoringProgress(projectId: string, step: AuthoringStep): Promise<AuthoringProgress> {
  return apiFetch<AuthoringProgress>(`/v1/projects/${projectId}/authoring/${step}/progress`);
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

/**
 * Cả ba kết quả đã lưu của pipeline soạn kịch bản (CR-025) — dùng để nạp
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
  /** Model-per-step picker's đã lưu cho project này — "" nghĩa là mặc định. */
  story_model?: string;
  storyboard_model?: string;
  code_model?: string;
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

/** Model-per-step picker's cho từng tab — "" nghĩa là dùng mặc định máy chủ. */
export interface AuthoringStepModels {
  story: string;
  storyboard: string;
  code: string;
}

/**
 * Lưu model Hive cho cả ba tab 1a/1b/1c trong một lượt — một picker ở bước 1
 * quyết định cho cả ba, giống hệt cách saveAuthoringMode lưu cách làm.
 */
export async function saveAuthoringModels(projectId: string, models: AuthoringStepModels): Promise<void> {
  await apiFetch<undefined>(`/v1/projects/${projectId}/authoring/models`, {
    method: "PUT",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(models),
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
  // CR-030 — optional: "" (mặc định) nghĩa là "không khai báo ở lượt gọi
  // này", server giữ nguyên engine đã lưu chứ không reset về Manim. Truyền
  // vào khi Creator vừa chọn engine ở "/" hoặc ngay trước khi chạy chuỗi AI
  // 1a→1b→1c ở tab 1a — server đọc project.RenderEngine để chọn đúng vai trò
  // Manim/Remotion cho 1b/1c, nên nó phải có mặt trước khi bước 1a chạy xong.
  renderEngine?: "manim" | "remotion",
): Promise<{ similarProjects: SimilarProject[] }> {
  const res = await apiFetch<{ project_id: string; similar_projects: similarProjectsWire[] }>(
    "/v1/projects",
    {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({
        project_id: projectId,
        topic,
        content_language: contentLanguage,
        ...(renderEngine ? { render_engine: renderEngine } : {}),
      }),
    },
  );
  return { similarProjects: fromWireSimilarProjects(res.similar_projects ?? []) };
}

/** Bước 2 (Cấu hình) — mọi thứ Creator chọn trước khi vào Script. */
export interface WizardSettingsInput {
  voiceLanguage: "vi" | "en";
  renderEngine: "manim" | "remotion";
  videoFont: string;
  ttsEnabled: boolean;
  voiceId: string | null;
  subtitleMode: string;
  subtitleStyle?: { font_family?: string; font_size: string; text_color: string; background_opacity: number; position: string };
  renderQuality: string;
  videoFormatId: string;
  videoOutputMode: string;
  /** "" xoá nhạc nền. */
  backgroundMusicPath: string;
  backgroundMusicVolume: number;
}

/** Chỉ các field vừa đổi; `confirm` = Creator bấm "Tiếp tục" (sang bước 3). */
export type WizardSettingsPatch = Partial<WizardSettingsInput> & { confirm?: boolean };

const WIZARD_PATCH_WIRE_KEYS: Record<keyof WizardSettingsPatch, string> = {
  voiceLanguage: "voice_language",
  renderEngine: "render_engine",
  videoFont: "video_font",
  ttsEnabled: "tts_enabled",
  voiceId: "voice_id",
  subtitleMode: "subtitle_mode",
  subtitleStyle: "subtitle_style",
  renderQuality: "render_quality",
  videoFormatId: "video_format_id",
  videoOutputMode: "video_output_mode",
  backgroundMusicPath: "background_music_path",
  backgroundMusicVolume: "background_music_volume",
  confirm: "confirm",
};

/**
 * Lưu bước 2 lên server từng field ngay khi Creator đổi — giọng đọc, phụ đề,
 * định dạng, chất lượng, nhạc nền nằm trong hàng project chứ không chỉ trong
 * localStorage, nên mở lại ở máy khác vẫn còn. 409 nếu render đã bắt đầu.
 */
export async function patchWizardSettings(projectId: string, patch: WizardSettingsPatch): Promise<void> {
  const body: Record<string, unknown> = {};
  for (const [key, value] of Object.entries(patch)) {
    if (value !== undefined) body[WIZARD_PATCH_WIRE_KEYS[key as keyof WizardSettingsPatch]] = value;
  }
  await apiFetch<undefined>(`/v1/projects/${projectId}/settings`, {
    method: "PATCH",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(body),
  });
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

/** One line of a project's journey through the 13-step flow (project_events). */
export interface ProjectEvent {
  id: number;
  project_id: string;
  at: string;
  flow_step: number;
  step_label: string;
  run_state: "idle" | "running" | "done" | "failed" | "cancelled";
  source: "authoring" | "saga";
  from_status?: string;
  to_status?: string;
  /** Bước dự án vừa rời; với dòng saga, duration_ms là thời gian ở bước này. */
  from_flow_step?: number;
  /** authoring: how long the run took; saga: time spent in from_status. */
  duration_ms?: number;
  detail?: string;
  content_chars?: number;
  prompt_tokens?: number;
  completion_tokens?: number;
}

export function listProjectEvents(id: string): Promise<ProjectEvent[]> {
  return apiFetch<ProjectEvent[]>(`/v1/projects/${id}/events`);
}

export function listRecentEvents(limit = 500): Promise<ProjectEvent[]> {
  return apiFetch<ProjectEvent[]>(`/v1/events?limit=${limit}`);
}

/** Dừng bước đang chạy; dự án ở lại bước đó (đã hủy) để thử lại. */
export function cancelProject(id: string): Promise<{ step: string; status: string }> {
  return apiFetch<{ step: string; status: string }>(`/v1/projects/${id}/cancel`, { method: "POST" });
}

export interface ForkResult {
  project_id: string;
  from_step: number;
  /** Nhạc nền không được mang sang bản mới — chọn lại ở bước Cấu hình. */
  needs_music_reselect: boolean;
}

/** Tạo project mới từ project này, làm lại từ bước `fromStep` (2-5). Bản gốc không đổi. */
export function forkProject(id: string, fromStep: number): Promise<ForkResult> {
  return apiFetch<ForkResult>(`/v1/projects/${id}/fork`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ from_step: fromStep }),
  });
}

/** FR116.3 — tiến độ chung của một lượt gọi chạy lâu. */
export interface OperationProgress {
  kind: string;
  phase: string;
  reasoning_chars: number;
  content_chars: number;
  done: number | null;
  total: number | null;
  elapsed_ms: number;
  status: "pending" | "running" | "succeeded" | "failed";
  error: string | null;
}

export function getOperation(operationId: string): Promise<OperationProgress> {
  return apiFetch<OperationProgress>(`/v1/operations/${operationId}`);
}
