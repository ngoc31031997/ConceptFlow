import type {
  Project,
  ProgressMessage,
  ProjectSummary,
  ProjectListFilter,
  ProjectPage,
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
   * "project sai trạng thái" lẫn "QC có lỗi chặn", và chỉ cái
   * thứ hai mới có đường đi tiếp — dò theo câu chữ sẽ vỡ ngay khi đổi từ ngữ.
   */
  readonly code?: string;
  /** Lỗi code của một hình minh hoạ, kèm số dòng để trình sửa chỉ ra. */
  readonly diagnostics?: CodeDiagnostic[];
  /** HTTP status of the refusal, when there was a response at all. */
  readonly status?: number;

  constructor(message: string, code?: string, diagnostics?: CodeDiagnostic[], status?: number) {
    super(message);
    this.code = code;
    this.diagnostics = diagnostics;
    this.status = status;
  }
}

export interface CodeDiagnostic {
  message: string;
  line: number | null;
}

/** Mã 409 mà `acknowledge_qc: true` đi qua được. */
export const ERROR_CODE_QC_BLOCKED = "qc_blocked";

/** Duyệt dàn ý, cho Saga chạy tiếp. */
export async function approveOutline(projectId: string): Promise<void> {
  await postDecision(`${GATEWAY_URL}/v1/projects/${projectId}/approve`);
}

/** Từ chối dàn ý — Saga kết thúc để Creator quay lại sửa script. */
export async function rejectOutline(projectId: string): Promise<void> {
  await postDecision(`${GATEWAY_URL}/v1/projects/${projectId}/reject`);
}

/**
 * Sửa một câu lời thoại ngay tại màn duyệt.
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

/** Các hình dạng video Creator chọn được. */
export async function fetchVideoFormats(): Promise<VideoFormat[]> {
  const response = await fetch(`${GATEWAY_URL}/v1/formats`);
  if (!response.ok) return [];
  const body = await response.json();
  return Array.isArray(body?.formats) ? (body.formats as VideoFormat[]) : [];
}

export function getProjectVideoUrl(projectId: string): string {
  return `${GATEWAY_URL}/v1/projects/${projectId}/video`;
}

export function getProjectThumbnailUrl(projectId: string): string {
  return `${GATEWAY_URL}/v1/projects/${projectId}/thumbnail`;
}

export interface ThumbnailInfo {
  exists: boolean;
  thumbnail_path: string | null;
  /** True when the image is the frame Video Assembly extracted, not the
   * Creator's own upload. */
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

async function parseError(
  response: Response,
): Promise<{ message: string; code?: string; diagnostics?: CodeDiagnostic[] }> {
  try {
    const body = await response.json();
    const code = typeof body?.code === "string" ? body.code : undefined;
    const diagnostics = Array.isArray(body?.diagnostics) ? (body.diagnostics as CodeDiagnostic[]) : undefined;
    if (typeof body?.error === "string") return { message: body.error, code, diagnostics };
    return {
      message: body?.error?.message ?? body?.error_message ?? GENERIC_CONNECTION_ERROR,
      code,
    };
  } catch {
    return { message: GENERIC_CONNECTION_ERROR };
  }
}

const NO_EXTERNAL_PROMPT = "Mục này không gọi AI nên không có prompt để copy.";

async function apiFetch<T>(path: string, init?: RequestInit): Promise<T> {
  if ((init?.headers as Record<string, string> | undefined)?.[EXTERNAL_AI_HEADER] === "describe") {
    throw new ExternalRequestDescribed({
      gateway: GATEWAY_URL,
      method: init?.method ?? "GET",
      path,
      body: typeof init?.body === "string" ? init.body : undefined,
    });
  }
  let response: Response;
  try {
    response = await fetch(`${GATEWAY_URL}${path}`, init);
  } catch {
    if (init?.signal?.aborted) throw new RequestCancelledError();
    throw new ApiError(GENERIC_CONNECTION_ERROR);
  }
  if (!response.ok) {
    if (init?.signal?.aborted) throw new RequestCancelledError();
    const { message, code, diagnostics } = await parseError(response);
    throw new ApiError(message, code, diagnostics, response.status);
  }
  // 202 (a job accepted) carries no body either.
  if (response.status === 204 || response.status === 202) {
    return undefined as T;
  }
  if ((init?.headers as Record<string, string> | undefined)?.[EXTERNAL_AI_HEADER] === "prompt") {
    const { external_prompt: prompt } = (await response.json()) as { external_prompt?: string };
    if (!prompt) throw new ApiError(NO_EXTERNAL_PROMPT);
    throw new ExternalPromptReady(prompt);
  }
  return response.json() as Promise<T>;
}

/** POST multipart bằng XHR — fetch chưa báo được số byte đã gửi (tải ảnh lên). */
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

export interface ProjectPageQuery {
  page: number;
  pageSize: number;
  filter: ProjectListFilter;
  /** Các bước (1-13) cần lọc; rỗng = mọi bước. */
  steps: number[];
}

/** Một trang danh sách video, đã lọc và đếm ở server. */
export function listProjectsPage(query: ProjectPageQuery): Promise<ProjectPage> {
  const params = new URLSearchParams({
    page: String(query.page),
    page_size: String(query.pageSize),
    filter: query.filter,
  });
  if (query.steps.length > 0) params.set("steps", query.steps.join(","));
  return apiFetch<ProjectPage>(`/v1/projects?${params}`);
}

export async function deleteProject(id: string): Promise<void> {
  await apiFetch<undefined>(`/v1/projects/${id}`, { method: "DELETE" });
}

export function startPublishSaga(
  id: string,
  metadata: PublishMetadata,
  /**
   * Chỉ gửi `true` sau khi Creator đã đọc báo cáo QC và chủ
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

/** Một phát hiện QC, kèm mốc thời gian để tua tới chỗ đó. */
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
 * Báo cáo QC của project.
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

/** Id do GUI tự cấp cho một lượt gọi chạy lâu để poll tiến độ trong lúc request còn mở. */
export function newOperationId(): string {
  return globalThis.crypto?.randomUUID?.() ?? `op-${Date.now()}-${Math.random().toString(36).slice(2)}`;
}

const OPERATION_ID_HEADER = "X-Operation-Id";

/** A long call as the GUI follows and can stop it: the id to poll and the signal that cancels the request. */
export interface OperationCall {
  id: string;
  signal: AbortSignal;
}

/** The Creator stopped the request; the server drops the model call when the connection closes. */
export class RequestCancelledError extends Error {
  constructor() {
    super("Request cancelled");
    this.name = "RequestCancelledError";
  }
}

/** True when `err` is the Creator cancelling a call, which is not a failure to report. */
export function isRequestCancelled(err: unknown): boolean {
  return err instanceof RequestCancelledError;
}

/**
 * How a step's AI call runs through an AI outside the system: ask for the prompt to copy, hand back
 * the reply, or only describe the request so it can be made later (nothing is sent).
 */
export type ExternalAi = { mode: "prompt" } | { mode: "paste"; reply: string } | { mode: "describe" };

/** The plain HTTP request of a step's AI call, enough to make it again later with an external-AI mode. */
export interface ExternalRequest {
  /** Base URL of the gateway the request goes to. */
  gateway: string;
  method: string;
  /** Path after the gateway, e.g. `/v1/projects/p1/beats/generate`. */
  path: string;
  /** The step's own JSON fields as a string; absent when the call has none. */
  body?: string;
}

/** The request of an external-AI call was described and not sent; carries what was described. */
export class ExternalRequestDescribed extends Error {
  constructor(readonly request: ExternalRequest) {
    super("External request described");
    this.name = "ExternalRequestDescribed";
  }
}

/** A step's AI call made through an AI outside the system instead of the model. */
export interface ExternalAiCall {
  external: ExternalAi;
}

/** How a step's AI call is made: followed and stoppable, or through an AI outside the system. */
export type AiCall = OperationCall | ExternalAiCall;

const EXTERNAL_AI_HEADER = "X-External-AI";

/** The prompt of an external-AI run was built and no model was asked; it is what the Creator copies. */
export class ExternalPromptReady extends Error {
  constructor(readonly prompt: string) {
    super("External prompt ready");
    this.name = "ExternalPromptReady";
  }
}

function externalOf(call?: AiCall): ExternalAi | undefined {
  return call && "external" in call ? call.external : undefined;
}

/** Headers and signal of a request that carries an operation id or an external-AI mode; empty otherwise. */
function operationInit(call?: AiCall): { headers: Record<string, string>; signal?: AbortSignal } {
  if (!call) return { headers: {} };
  if ("external" in call) return { headers: { [EXTERNAL_AI_HEADER]: call.external.mode } };
  return { headers: { [OPERATION_ID_HEADER]: call.id }, signal: call.signal };
}

/** The JSON body of an AI call: the step's own fields, plus the pasted reply when the call is a paste. */
function operationBody(body: object | undefined, call?: AiCall): string | undefined {
  const external = externalOf(call);
  if (external?.mode === "paste") return JSON.stringify({ ...body, external_reply: external.reply });
  return body === undefined ? undefined : JSON.stringify(body);
}

export function suggestPublishMetadata(id: string, operation?: OperationCall): Promise<SuggestedMetadata> {
  return apiFetch<SuggestedMetadata>(`/v1/projects/${id}/suggest-metadata`, {
    method: "POST",
    ...operationInit(operation),
  });
}

export function getYoutubeAuthStartUrl(projectId: string, clientId?: string): string {
  const params = new URLSearchParams({ state: projectId });
  // Omitted when there is only one configured client — the Publisher picks
  // it, so the Creator never sees a one-option chooser.
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
  // Ý tưởng: viết câu hỏi trung tâm, vì sao thú vị, đã biết và chưa biết từ đề tài.
  | "idea"
  // Creative brief: viết định hướng sáng tạo từ ý tưởng đã duyệt (AI chạy ở server, không chép ra ngoài).
  | "creative_brief"
  // Nghiên cứu: AI đặt câu hỏi từ brief, và prompt chép sang công cụ nghiên cứu bên ngoài.
  | "research_questions"
  | "research_prompt"
  // Câu chuyện: AI gợi ý góc tiếp cận và viết khung câu chuyện.
  | "story_frame"
  | "beat_sheet"
  // Viết kịch bản: AI viết lời thoại từng beat và gợi ý sửa câu.
  | "script_writer"
  // Rà soát và chốt: rà soát cả kịch bản, viết ghi chú bàn giao một beat.
  | "script_review"
  | "script_handoff"
  | "story_architect"
  | "visual_director"
  | "remotion_engineer"
  // Luồng "Chạy bằng AI" có prompt riêng (storyboard xuất JSON, code
  // chỉ viết các hàm shot); luồng Copy giữ ba vai trò trên nguyên vẹn.
  | "visual_director_ai"
  | "remotion_engineer_ai"
  // "Copy to an external AI" prompts, rendered by the server.
  | "remotion_adjust"
  | "thumbnail_design"
  // Luật style hình minh hoạ mà AI vẽ hình phải theo; không render cho AI ngoài.
  | "illustration_style";

/**
 * Một dòng trong thư viện prompt. Mỗi vai trò có một danh sách; tại
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
 * What the caller has in hand but has not saved. Every field is
 * optional; the server fills a blank one with the placeholder the browser used
 * to show.
 */
export interface PromptRenderInput {
  role: PromptRole;
  language: "vi" | "en";
  topic?: string;
  /** The Creator's existing code, for `remotion_adjust`. */
  script?: string;
  previous_output?: string;
  subtitle_mode?: string;
  subtitle_font_size?: string;
  subtitle_position?: string;
  format_id?: string;
  format_version?: number;
  voice_id?: string;
  /** "long" | "short": which frame the frame variables describe. */
  video_output_mode?: string;
  /** The project whose approved creative brief fills `{{creative_brief}}` (story_architect). */
  project_id?: string;
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

/** Một kiểu video mà prompt Biên kịch có thể được bảo dựng. */
export interface VideoArchetype {
  id: string;
  /** Mã ngắn (A, B, ...) — Creator gõ "kiểu: B" vào chủ đề để ép kiểu. */
  code: string;
  name: string;
  /** Chủ đề nào hợp — model chọn kiểu dựa vào đây. */
  when_to_use: string;
  /** Cách gán kiểu này vào các beat của format đang chọn. */
  playbook: string;
  /** Format hợp nhất với kiểu này — wizard gợi ý khi chọn kiểu. "" = không gợi ý. */
  recommended_format_id: string;
  is_system: boolean;
}

export interface VideoArchetypeInput {
  /** Để trống khi tạo mới thì server lấy chữ cái còn trống kế tiếp. */
  code: string;
  name: string;
  when_to_use: string;
  playbook: string;
  recommended_format_id?: string;
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

/** Một ngăn của thư viện hình minh hoạ. */
export interface IllustrationFolder {
  id: string;
  name: string;
  description: string;
  position: number;
  is_system: boolean;
}

export type IllustrationStatus = "draft" | "approved";

/** Một hình của thư viện: có sẵn trong bộ minh hoạ, hoặc do AI/Creator vẽ. */
/** A figure placed in a shot, or a whole-frame place a Scene is drawn in. */
export type IllustrationKind = "figure" | "backdrop";

export interface Illustration {
  id: string;
  /** "backdrop" for a whole-frame place; absent or "figure" for a figure. */
  kind?: IllustrationKind;
  /** Tên component (PascalCase) mà Kỹ sư Remotion gọi. */
  name: string;
  title: string;
  folder_id: string;
  tags: string[];
  description: string;
  /** Một dòng API: tham số và tỉ lệ khung. */
  usage: string;
  code?: string;
  builtin: boolean;
  /** Hình mẫu AI vẽ học theo: nằm ở thư mục "hinh-mau", chỉ xem, không xoá. */
  exemplar: boolean;
  /** Bản Hình mẫu chép từ hình này (id hình gốc); trống khi hình gốc đã bị xoá. */
  source_id?: string;
  /** Hình mẫu gốc: thư mục nó quay về khi "Bỏ làm mẫu". */
  home_folder_id?: string;
  /** Cảnh báo style của phiên bản hiện tại — không chặn lưu. */
  warnings: CodeDiagnostic[];
  status: IllustrationStatus;
  version: number;
  has_preview: boolean;
  /** Kit (ngôn ngữ hình) hình được vẽ cho: video chỉ dùng hình cùng kit. */
  kit_key?: string;
  /** Mức chi tiết của bản vẽ: kỹ, vừa hay bóng. Hạng chính/phụ/phông là vai trong từng video nên không lưu ở đây. */
  detail_level?: DetailLevel;
}

/** Mức chi tiết của một hình trong thư viện. */
export type DetailLevel = "rich" | "medium" | "silhouette";

/** Hạng của hình trong một video: vai của nó trong chuyện đó. */
export type IllustrationTier = "hero" | "support" | "background";

export const DETAIL_LABEL: Record<DetailLevel, string> = { rich: "Kỹ", medium: "Vừa", silhouette: "Bóng" };
export const TIER_LABEL: Record<IllustrationTier, string> = { hero: "Chính", support: "Phụ", background: "Phông" };

/**
 * Hình có mức chi tiết `detail` có đủ cho hạng `tier` không: kỹ dùng cho mọi
 * hạng, vừa cho phụ và phông, bóng chỉ cho phông.
 */
export function detailEnough(detail: DetailLevel | undefined, tier: IllustrationTier): boolean {
  if (detail === "rich") return true;
  if (detail === "medium") return tier !== "hero";
  if (detail === "silhouette") return tier === "background";
  return false;
}

export interface IllustrationInput {
  /** Mức chi tiết hình mới được vẽ (mặc định "medium"). */
  detail_level?: DetailLevel;
  name: string;
  title: string;
  folder_id: string;
  tags: string[];
  description: string;
  usage: string;
  code: string;
}

/** Ảnh xem trước dạng base64, cho nút "Xem trước" trong trình sửa (chưa lưu gì). */
export interface IllustrationTry {
  png: string;
  gif: string;
  warnings?: CodeDiagnostic[];
}

/** Thư mục chỉ nhận hình qua "Đặt làm mẫu". */
export const EXEMPLAR_FOLDER_ID = "hinh-mau";

/** Luật style của kênh và id các Hình mẫu, cũ nhất trước. */
export interface IllustrationStyle {
  rules: string;
  exemplar_ids: string[];
  max_exemplars: number;
}

export function getIllustrationStyle(): Promise<IllustrationStyle> {
  return apiFetch<IllustrationStyle>("/v1/illustration-style");
}

/** AI vẽ một hình mới theo luật style; lưu ở "Chờ duyệt". Có thể mất vài phút. */
export function drawIllustration(input: {
  description: string;
  folder_id: string;
  name?: string;
  model?: string;
  /** Mức chi tiết: "rich" vẽ như vật chính, "silhouette" như phông (mặc định "medium"). */
  detail_level?: DetailLevel;
}): Promise<Illustration> {
  return apiFetch<Illustration>("/v1/admin/illustrations/draw", {
    method: "POST",
    headers: JSON_HEADERS,
    body: JSON.stringify(input),
  });
}

/** AI vẽ lại một hình của bạn theo ghi chú; giữ tên, ra phiên bản mới ở "Chờ duyệt". */
export function redrawIllustration(id: string, note: string, model?: string): Promise<Illustration> {
  return apiFetch<Illustration>(`/v1/admin/illustrations/${id}/redraw`, {
    method: "POST",
    headers: JSON_HEADERS,
    body: JSON.stringify({ note, ...(model ? { model } : {}) }),
  });
}

export async function listIllustrationFolders(): Promise<IllustrationFolder[]> {
  return (await apiFetch<{ folders: IllustrationFolder[] }>("/v1/illustration-folders")).folders;
}

export function createIllustrationFolder(input: { id: string; name: string; description?: string }): Promise<IllustrationFolder> {
  return apiFetch<IllustrationFolder>("/v1/admin/illustration-folders", {
    method: "POST",
    headers: JSON_HEADERS,
    body: JSON.stringify(input),
  });
}

export async function deleteIllustrationFolder(id: string): Promise<void> {
  await apiFetch<undefined>(`/v1/admin/illustration-folders/${id}`, { method: "DELETE" });
}

export async function listIllustrations(filter: { folder?: string; q?: string; status?: IllustrationStatus } = {}): Promise<Illustration[]> {
  const params = new URLSearchParams();
  for (const [k, v] of Object.entries(filter)) if (v) params.set(k, v);
  const qs = params.toString();
  return (await apiFetch<{ illustrations: Illustration[] }>(`/v1/illustrations${qs ? `?${qs}` : ""}`)).illustrations;
}

/** URL ảnh xem trước; có số phiên bản để trình duyệt cache đúng bản. */
export function illustrationPreviewUrl(ill: Pick<Illustration, "id" | "version">, kind: "png" | "gif" = "png"): string {
  return `${GATEWAY_URL}/v1/illustrations/${ill.id}/preview.${kind}?v=${ill.version}`;
}

/** Dựng thử code mà không lưu. Code hỏng → ApiError có `diagnostics`. */
export function tryIllustration(name: string, code: string): Promise<IllustrationTry> {
  return apiFetch<IllustrationTry>("/v1/illustration-tries", {
    method: "POST",
    headers: JSON_HEADERS,
    body: JSON.stringify({ name, code }),
  });
}

export function createIllustration(input: IllustrationInput): Promise<Illustration> {
  return apiFetch<Illustration>("/v1/admin/illustrations", {
    method: "POST",
    headers: JSON_HEADERS,
    body: JSON.stringify(input),
  });
}

/** Sửa code thì hình quay về "chờ duyệt" với phiên bản mới. Hình có sẵn trả 403. */
export function updateIllustration(id: string, input: IllustrationInput): Promise<Illustration> {
  return apiFetch<Illustration>(`/v1/admin/illustrations/${id}`, {
    method: "PUT",
    headers: JSON_HEADERS,
    body: JSON.stringify(input),
  });
}

export function setIllustrationStatus(id: string, status: IllustrationStatus): Promise<Illustration> {
  return apiFetch<Illustration>(`/v1/admin/illustrations/${id}/status`, {
    method: "POST",
    headers: JSON_HEADERS,
    body: JSON.stringify({ status }),
  });
}

export function rerenderIllustration(id: string): Promise<IllustrationTry> {
  return apiFetch<IllustrationTry>(`/v1/admin/illustrations/${id}/rerender`, { method: "POST" });
}

/** Xoá hình. Hình một dự án chưa tới bước Kết quả đang dùng → ApiError 409 nêu tên dự án. */
export async function deleteIllustration(id: string): Promise<void> {
  await apiFetch<undefined>(`/v1/admin/illustrations/${id}`, { method: "DELETE" });
}

/** Chép một hình đã duyệt vào thư mục Hình mẫu (tên mới <Tên>Mau). Trả bản chép. */
export function makeExemplar(id: string): Promise<Illustration> {
  return apiFetch<Illustration>(`/v1/admin/illustrations/${id}/exemplar`, { method: "POST" });
}

/**
 * Bỏ một hình khỏi Hình mẫu. Bản chép bị xoá (trả undefined); Hình mẫu gốc
 * về lại thư mục chủ đề của nó và được trả về.
 */
export function unmakeExemplar(id: string): Promise<Illustration | undefined> {
  return apiFetch<Illustration | undefined>(`/v1/admin/illustrations/${id}/exemplar`, { method: "DELETE" });
}

/** Kết quả nhập lại một hình từ file sao lưu. */
export interface IllustrationImportItem {
  name: string;
  folder_id: string;
  /** created: thêm mới · replaced: ghi đè hình cùng tên · skipped: để nguyên · failed: không nhập được. */
  result: "created" | "replaced" | "skipped" | "failed";
  reason?: string;
  diagnostics?: CodeDiagnostic[];
}

/** Báo cáo một lần khôi phục thư viện hình. */
export interface IllustrationImportReport {
  exported_at: string;
  folders_created: string[];
  folder_errors: string[];
  items: IllustrationImportItem[];
  /** Có khi bộ dựng/cơ sở dữ liệu hỏng giữa chừng: các hình từ đó trở đi chưa được thử. */
  aborted?: string;
  not_processed: number;
}

/** Giới hạn file sao lưu, khớp với api-gateway và authoring-service. */
export const ILLUSTRATION_BACKUP_MAX_BYTES = 256 * 1024 * 1024;

/** Tải toàn bộ thư viện hình (thư mục, code, ảnh PNG/GIF) về một file ZIP. */
export async function exportIllustrationLibrary(): Promise<Blob> {
  let response: Response;
  try {
    response = await fetch(`${GATEWAY_URL}/v1/admin/illustrations/export`);
  } catch {
    throw new ApiError(GENERIC_CONNECTION_ERROR);
  }
  if (!response.ok) {
    const { message, code } = await parseError(response);
    throw new ApiError(message, code);
  }
  return response.blob();
}

/**
 * Khôi phục thư viện hình từ file ZIP đã xuất. Mỗi hình được kiểm tra và dựng
 * lại, nên có thể mất vài phút. `replace`: ghi đè hình cùng tên đang có.
 */
export function importIllustrationLibrary(file: Blob, replace: boolean): Promise<IllustrationImportReport> {
  return apiFetch<IllustrationImportReport>(
    `/v1/admin/illustrations/import?on_conflict=${replace ? "replace" : "skip"}`,
    { method: "POST", headers: { "Content-Type": "application/zip" }, body: file },
  );
}

/** Một hình mà video cần: lập từ storyboard, rồi dùng lại hoặc vẽ mới. */
export type ProjectIllustrationState = "planned" | "drawing" | "drawn" | "reused" | "failed" | "skipped";

export interface ProjectIllustration {
  id: string;
  /** "backdrop": the place of the scenes its shots belong to. */
  kind?: IllustrationKind;
  position: number;
  name: string;
  description: string;
  folder_id: string;
  shots: string[];
  state: ProjectIllustrationState;
  error?: string;
  illustration_id?: string;
  illustration?: Illustration;
  /** Vai của hình trong video này; nền không có hạng. */
  tier?: IllustrationTier;
  /** Một câu AI giải thích vì sao đề xuất hạng đó. */
  tier_reason?: string;
  /** Tiến độ sống khi hình đang được vẽ; không có khi không vẽ. */
  progress?: DrawProgress;
}

/** AI vẽ một hình tới đâu: lượt thử thứ mấy, đang làm gì trong lượt đó. */
export interface DrawProgress {
  attempt: number;
  max_attempts: number;
  /** "waiting" đã gửi · "reasoning" AI đang nghĩ · "writing" AI đang viết code · "checking" đang kiểm tra và dựng ảnh. */
  phase: "waiting" | "reasoning" | "writing" | "checking";
  reasoning_chars: number;
  content_chars: number;
  elapsed_seconds: number;
}

export interface ProjectIllustrations {
  illustrations: ProjectIllustration[];
  /** Mọi hình đã duyệt hoặc bỏ qua, và danh sách không cũ — bước Code được chạy. */
  ready: boolean;
  /** Danh sách lập từ storyboard cũ hơn storyboard đang lưu. */
  stale?: boolean;
}

export function listProjectIllustrations(projectId: string): Promise<ProjectIllustrations> {
  return apiFetch<ProjectIllustrations>(`/v1/projects/${projectId}/illustrations`);
}

/** Lập lại danh sách từ storyboard (thay danh sách cũ; hình đã vẽ vẫn ở thư viện). */
export function planProjectIllustrations(projectId: string): Promise<ProjectIllustrations> {
  return apiFetch<ProjectIllustrations>(`/v1/projects/${projectId}/illustrations/plan`, { method: "POST" });
}

export function drawProjectIllustration(projectId: string, rowId: string): Promise<ProjectIllustration> {
  return apiFetch<ProjectIllustration>(`/v1/projects/${projectId}/illustrations/${rowId}/draw`, { method: "POST" });
}

/**
 * Xoá hình nháp AI vẽ cho video này khỏi thư viện (tránh rác) và bỏ
 * qua hình đó cho video. Hình đã duyệt hoặc dùng lại từ thư viện không xoá được.
 */
export function deleteProjectIllustrationDrawing(projectId: string, rowId: string): Promise<ProjectIllustration> {
  return apiFetch<ProjectIllustration>(`/v1/projects/${projectId}/illustrations/${rowId}/drawing`, { method: "DELETE" });
}

/** Đổi hạng của một hình trong video này. Hình đã vẽ không đủ chi tiết cho hạng mới thì quay về "planned" dưới tên mới. */
export function setProjectIllustrationTier(projectId: string, rowId: string, tier: IllustrationTier): Promise<ProjectIllustration> {
  return apiFetch<ProjectIllustration>(`/v1/projects/${projectId}/illustrations/${rowId}/tier`, {
    method: "PUT",
    headers: JSON_HEADERS,
    body: JSON.stringify({ tier }),
  });
}

export function skipProjectIllustration(projectId: string, rowId: string, skipped: boolean): Promise<ProjectIllustration> {
  return apiFetch<ProjectIllustration>(`/v1/projects/${projectId}/illustrations/${rowId}/skip`, {
    method: "POST",
    headers: JSON_HEADERS,
    body: JSON.stringify({ skipped }),
  });
}

/**
 * Nút "Chạy bằng AI" có nơi nào để gọi không. Hỏi trước khi
 * vẽ nút: một nút bấm vào là lỗi tệ hơn một nút không có kèm lời giải thích.
 */
/**
 * Một model trong danh mục Hive máy chủ cho phép chọn (mỗi bước một model)
 * — `id` là chuỗi gửi thẳng cho Hive, `label` là tên hiển thị.
 * "" luôn là một lựa chọn hợp lệ, nghĩa là "dùng mặc định máy chủ".
 */
export type AuthoringModelOption = {
  id: string;
  label: string;
  /** True = model chạy trên máy này (Ollama); bước Code và Hình minh hoạ có thể hết giờ. */
  local?: boolean;
};

export type LlmStatus = {
  enabled: boolean;
  provider: string;
  reason?: string;
  /** Danh mục model cho picker ở bước 1 — rỗng khi chế độ AI chưa khả dụng. */
  models?: AuthoringModelOption[];
  /** Model mặc định của từng bước, được chọn sẵn ở chỗ chọn model của bước đó. */
  step_defaults?: Record<string, string>;
};

export function getLlmStatus(): Promise<LlmStatus> {
  return apiFetch<LlmStatus>("/v1/llm/status");
}

/**
 * Cách làm bước 1, theo đúng hai giá trị server nhận. Kiểu nằm ở
 * đây vì đây là hợp đồng trên đường truyền; ProjectDraftContext export lại nó
 * kèm ý nghĩa nghiệp vụ.
 */
export type AuthoringMode = "manual" | "ai";

/** Bước 1 — lưu dàn ý câu chuyện (Story Architect) Creator dán vào. */
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

/** Bước 2 — lưu storyboard (Visual Director) Creator dán vào. */
export async function saveAuthoringStoryboard(projectId: string, content: string): Promise<void> {
  await apiFetch<undefined>(`/v1/projects/${projectId}/authoring/storyboard`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ content }),
  });
}

/** Bước Code — lưu code Remotion Creator dán vào. */
export async function saveAuthoringCode(projectId: string, content: string): Promise<void> {
  await apiFetch<undefined>(`/v1/projects/${projectId}/authoring/code`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ content }),
  });
}

/**
 * Cả ba kết quả đã lưu của pipeline soạn kịch bản — dùng để nạp
 * lại trạng thái khi Creator tải lại trang hoặc quay lại một bước trước đó,
 * thay vì chỉ dựa vào draft ở client (localStorage có thể đã mất khi mở lại
 * bằng một trình duyệt/máy khác dùng chung project_id).
 */
export interface AuthoringState {
  /**
   * Cách làm bước 1 đã lưu cho project này: "manual" hoặc "ai".
   * Server luôn trả một trong hai (project cũ đọc ra "manual"), nhưng để
   * optional để một orchestrator chưa nâng cấp không làm vỡ phần rehydrate.
   */
  mode?: AuthoringMode;
  /** "" khi project chưa lưu chủ đề. */
  topic: string;
  story: string;
  storyboard: string;
  code: string;
  /** Model-per-step picker's đã lưu cho project này — "" nghĩa là mặc định. */
  brief_model?: string;
  publish_model?: string;
  research_model?: string;
  story_frame_model?: string;
  beat_sheet_model?: string;
  story_model?: string;
  storyboard_model?: string;
  code_model?: string;
}

export function getAuthoringState(projectId: string): Promise<AuthoringState> {
  return apiFetch<AuthoringState>(`/v1/projects/${projectId}/authoring`);
}

/**
 * Lưu cách làm bước 1 cho project này.
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
  /** Model of the idea step's creative brief; missing or "" reads as the default. */
  brief?: string;
  /** Model that suggests the publish title, description and tags. */
  publish?: string;
  /** Model that writes the research questions; missing or "" reads as the default. */
  research?: string;
  /** Model that proposes the story approaches and writes the story frame; missing or "" reads as the default. */
  story_frame?: string;
  /** Model that writes the beat sheet; missing or "" reads as the default. */
  beat_sheet?: string;
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

/**
 * The default model of every step, as set in Cài đặt: the model a step is
 * pre-selected on and runs on until the Creator picks another.
 */
export interface AiModelSetting {
  /** Model mặc định của từng bước (bước chưa đặt là model đầu danh mục). */
  stepModels: Record<string, string>;
  /** Các bước cấu hình được model, theo thứ tự hiển thị. */
  steps: string[];
}

export async function getAiModelSetting(): Promise<AiModelSetting> {
  const wire = await apiFetch<{ step_models: Record<string, string>; steps: string[] }>("/v1/settings/ai-model");
  return { stepModels: wire.step_models ?? {}, steps: wire.steps ?? [] };
}

/** Sets the default model of one step. 400 for an unknown step or a model that is not listed. */
export async function setAiModelStepDefault(step: string, model: string): Promise<void> {
  await apiFetch<undefined>(`/v1/settings/ai-model/steps/${encodeURIComponent(step)}`, {
    method: "PUT",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ model }),
  });
}

/** Một project khác (cùng ngôn ngữ) có chủ đề trùng sau khi chuẩn hoá. */
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
 * Tạo hàng project ngay khi Creator gõ xong chủ đề (bước 1),
 * thay vì đợi tới POST /v1/sagas/render. `projectId` là id đã sinh sẵn ở
 * client (ProjectDraftContext) — gửi lên để mọi endpoint authoring đã và sẽ
 * gọi với id đó vẫn trỏ đúng một project, không đổi kiến trúc id ở client.
 */
export async function createProjectDraft(
  projectId: string,
  topic: string,
  contentLanguage: "vi" | "en",
  // Optional: the long video this new draft is the vertical short of. The
  // server then creates it as a short, linked to that video both ways.
  shortOf?: string,
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
        ...(shortOf ? { short_of: shortOf } : {}),
      }),
    },
  );
  return { similarProjects: fromWireSimilarProjects(res.similar_projects ?? []) };
}

/**
 * Cấu hình của project. Bước 2 lưu ngôn ngữ, giọng, định dạng, kiểu đầu
 * ra; chất lượng, font, phụ đề, nhạc nền được lưu ở màn Review hoặc khi bước
 * đọc chúng lỗi (ProductionSettingsPanel).
 */
export interface WizardSettingsInput {
  voiceLanguage: "vi" | "en";
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
 * Lưu cấu hình lên server từng field ngay khi Creator đổi — nằm trong hàng
 * project chứ không chỉ trong localStorage, nên mở lại ở máy khác vẫn còn.
 * 409 nếu bước đọc field đó đã chạy (hoặc đang chạy).
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
 * Creator quay lại bước 1 và sửa chủ đề của draft đã tạo.
 * 409 nếu render đã bắt đầu (cùng khoá với authoring saves).
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
  /** "illustrations" = bước Hình minh hoạ, ghi dưới số bước của Code. */
  source: "authoring" | "saga" | "illustrations";
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

/** Một dòng nhật ký cuộc gọi AI của llm-service: phản hồi thô, chi phí và cỡ prompt; prompt đầy đủ lấy bằng `getLlmCall`. */
export interface LlmCall {
  call_id: string;
  created_at: string;
  provider: string;
  model: string;
  role: string;
  step: string;
  phase: string;
  project_id: string;
  response: string;
  ok: boolean;
  error_kind: string;
  error_message: string;
  prompt_tokens: number;
  completion_tokens: number;
  reasoning_tokens: number;
  cached_tokens: number;
  reasoning_chars: number;
  usage_reported: boolean;
  duration_ms: number;
  /** Số ký tự của system + user prompt đã gửi. */
  prompt_chars: number;
  /** Cửa sổ ngữ cảnh gửi cho Ollama; 0 khi nhà cung cấp tự đặt. */
  num_ctx: number;
}

/** Một cuộc gọi kèm prompt đã gửi. */
export interface LlmCallDetail extends LlmCall {
  system_prompt: string;
  user_prompt: string;
}

export interface LlmCallsFilter {
  project_id?: string;
  role?: string;
  ok?: boolean;
  /** Chỉ lấy các dòng cũ hơn mốc này (`next_before` của trang trước). */
  before?: string;
  limit?: number;
}

export interface LlmCallsPage {
  calls: LlmCall[];
  /** Mốc của trang kế tiếp; null khi đã hết. */
  next_before: string | null;
}

export function listLlmCalls(filter: LlmCallsFilter = {}): Promise<LlmCallsPage> {
  const q = new URLSearchParams();
  if (filter.project_id) q.set("project_id", filter.project_id);
  if (filter.role) q.set("role", filter.role);
  if (filter.ok !== undefined) q.set("ok", String(filter.ok));
  if (filter.before) q.set("before", filter.before);
  q.set("limit", String(filter.limit ?? 100));
  return apiFetch<LlmCallsPage>(`/v1/llm/calls?${q.toString()}`);
}

export function getLlmCall(callId: string): Promise<LlmCallDetail> {
  return apiFetch<LlmCallDetail>(`/v1/llm/calls/${encodeURIComponent(callId)}`);
}

/** Chọn đúng một cách: các dòng đã chọn, các dòng cũ hơn N ngày, hoặc tất cả. */
export type LlmCallsDeletion = { ids: string[] } | { older_than_days: number } | { all: true };

export function deleteLlmCalls(selection: LlmCallsDeletion): Promise<{ deleted: number }> {
  return apiFetch<{ deleted: number }>("/v1/llm/calls/delete", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(selection),
  });
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

/** Tạo project mới từ project này, làm lại từ bước `fromStep` (2, 3, 4 hoặc 6 — xem FORK_STEPS). Bản gốc không đổi. */
export function forkProject(id: string, fromStep: number): Promise<ForkResult> {
  return apiFetch<ForkResult>(`/v1/projects/${id}/fork`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ from_step: fromStep }),
  });
}

/** Tiến độ chung của một lượt gọi chạy lâu. */
export interface OperationProgress {
  kind: string;
  phase: string;
  reasoning_chars: number;
  content_chars: number;
  done: number | null;
  total: number | null;
  elapsed_ms: number;
  status: "pending" | "running" | "succeeded" | "failed" | "cancelled";
  error: string | null;
}

/** A call still running for a project, with the id it is polled and cancelled by. */
export interface RunningOperation extends OperationProgress {
  id: string;
}

/** The calls a project still has running, so a page opened again can pick up a call it did not start. */
export async function listRunningOperations(projectId: string): Promise<RunningOperation[]> {
  const res = await apiFetch<{ operations: RunningOperation[] | null }>(`/v1/projects/${projectId}/operations`);
  return res.operations ?? [];
}

/** Stops a running call by its id; 404 when it already ended. */
export function cancelOperation(operationId: string): Promise<void> {
  return apiFetch<void>(`/v1/operations/${operationId}/cancel`, { method: "POST" });
}

export function getOperation(operationId: string): Promise<OperationProgress> {
  return apiFetch<OperationProgress>(`/v1/operations/${operationId}`);
}

/** Review state of one unit: waiting for the Creator, or approved. See ADR-0035. */
export type UnitStatus = "pending" | "approved";

/**
 * One reviewable piece of a step (the brief, later a beat or a scene): its
 * content, the fields the Creator wrote by hand (an AI redo keeps them), and
 * how many times the AI redid it.
 */
export interface ReviewUnit<C> {
  projectId: string;
  kind: string;
  key: string;
  status: UnitStatus;
  content: C;
  creatorFields: string[];
  redoCount: number;
  updatedAt: string;
  /** Written from a unit that has changed since (the brief after its idea). */
  stale: boolean;
}

/** Names one unit and reads its content from the wire. */
export interface UnitRef<C> {
  kind: string;
  key: string;
  parse: (content: unknown) => C;
}

interface reviewUnitWire {
  project_id: string;
  kind: string;
  key: string;
  status: UnitStatus;
  content: unknown;
  creator_fields: string[] | null;
  redo_count: number;
  updated_at: string;
  stale?: boolean;
}

function fromWireUnit<C>(wire: reviewUnitWire, parse: (content: unknown) => C): ReviewUnit<C> {
  return {
    projectId: wire.project_id,
    kind: wire.kind,
    key: wire.key,
    status: wire.status,
    content: parse(wire.content),
    creatorFields: wire.creator_fields ?? [],
    redoCount: wire.redo_count,
    updatedAt: wire.updated_at,
    stale: wire.stale === true,
  };
}

/** Which unit, without how to read it. */
export type UnitAddress = Pick<UnitRef<unknown>, "kind" | "key">;

function unitPath(projectId: string, ref: UnitAddress): string {
  return `/v1/projects/${projectId}/units/${encodeURIComponent(ref.kind)}/${encodeURIComponent(ref.key)}`;
}

/** Saves the Creator's hand edit of some fields; a changed field becomes the Creator's. */
export async function editUnit<C>(
  projectId: string,
  ref: UnitRef<C>,
  fields: Record<string, string>,
): Promise<ReviewUnit<C>> {
  const wire = await apiFetch<reviewUnitWire>(unitPath(projectId, ref), {
    method: "PUT",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ fields }),
  });
  return fromWireUnit(wire, ref.parse);
}

/** Has the AI redo the unit with the Creator's note; the unit goes back to review. */
export async function redoUnit<C>(
  projectId: string,
  ref: UnitRef<C>,
  note: string,
  operation?: AiCall,
): Promise<ReviewUnit<C>> {
  const wire = await apiFetch<reviewUnitWire>(`${unitPath(projectId, ref)}/redo`, {
    method: "POST",
    headers: { "Content-Type": "application/json", ...operationInit(operation).headers },
    signal: operationInit(operation).signal,
    body: operationBody({ note }, operation),
  });
  return fromWireUnit(wire, ref.parse);
}

/** Approves the unit. 400 while the unit is incomplete. */
export async function approveUnit<C>(projectId: string, ref: UnitRef<C>): Promise<ReviewUnit<C>> {
  const wire = await apiFetch<reviewUnitWire>(`${unitPath(projectId, ref)}/approve`, { method: "POST" });
  return fromWireUnit(wire, ref.parse);
}

/** Takes the approval back; the unit waits for review again. */
export async function unapproveUnit<C>(projectId: string, ref: UnitRef<C>): Promise<ReviewUnit<C>> {
  const wire = await apiFetch<reviewUnitWire>(`${unitPath(projectId, ref)}/unapprove`, { method: "POST" });
  return fromWireUnit(wire, ref.parse);
}

/**
 * What a change to the unit sends back in later steps, as the server's
 * dependency tracking says, one line per item. Empty when nothing depends on it yet.
 */
export async function getUnitImpact(projectId: string, ref: UnitAddress): Promise<string[]> {
  const res = await apiFetch<{ items: string[] | null }>(`${unitPath(projectId, ref)}/impact`);
  return res.items ?? [];
}

/** Where an idea stands in review. */
export type ProjectIdeaStatus = "pending" | "approved";

/** The idea the AI made of the Creator's text, with what review needs around it. */
export interface ProjectIdea {
  centralQuestion: string;
  curiosityPoints: string[];
  scope: string;
  /** Fields the Creator wrote or changed; the AI keeps them when it writes again. */
  creatorFields: ProjectIdeaField[];
  status: ProjectIdeaStatus;
  redoCount: number;
  approvedAt: string | null;
  updatedAt: string;
}

/** The idea's fields, in the order the screen shows them. */
export const PROJECT_IDEA_FIELDS = ["central_question", "curiosity_points", "scope"] as const;
export type ProjectIdeaField = (typeof PROJECT_IDEA_FIELDS)[number];

export type IdeaRating = "good" | "fair" | "weak";

export interface IdeaAspect {
  rating: IdeaRating;
  comment: string;
}

/** The AI's reading of an idea before approval; advice only. */
export interface IdeaReview {
  clarity: IdeaAspect;
  researchability: IdeaAspect;
  scopeFit: IdeaAspect;
  unclear: string[];
  suggestions: string[];
  createdAt: string;
}

/** Everything the idea screen shows. */
export interface IdeaScreen {
  idea: ProjectIdea | null;
  /** The text the Creator typed, kept with the project. */
  subject: string;
  language: "vi" | "en" | "";
  /** The text or language changed after the idea was written. */
  inputChanged: boolean;
  review: IdeaReview | null;
  /** The idea changed after the review read it. */
  reviewStale: boolean;
}

interface ideaWire {
  central_question: string;
  curiosity_points: string[] | null;
  scope: string;
  creator_fields: string[] | null;
  status: ProjectIdeaStatus;
  redo_count: number;
  approved_at: string | null;
  updated_at: string;
}

interface ideaReviewWire {
  clarity: IdeaAspect;
  researchability: IdeaAspect;
  scope_fit: IdeaAspect;
  unclear: string[] | null;
  suggestions: string[] | null;
  created_at: string;
}

function fromWireIdea(w: ideaWire): ProjectIdea {
  return {
    centralQuestion: w.central_question,
    curiosityPoints: w.curiosity_points ?? [],
    scope: w.scope,
    creatorFields: (w.creator_fields ?? []).filter((f): f is ProjectIdeaField =>
      (PROJECT_IDEA_FIELDS as readonly string[]).includes(f),
    ),
    status: w.status,
    redoCount: w.redo_count,
    approvedAt: w.approved_at,
    updatedAt: w.updated_at,
  };
}

function fromWireIdeaReview(w: ideaReviewWire): IdeaReview {
  return {
    clarity: w.clarity,
    researchability: w.researchability,
    scopeFit: w.scope_fit,
    unclear: w.unclear ?? [],
    suggestions: w.suggestions ?? [],
    createdAt: w.created_at,
  };
}

/** Reads the first step: the idea, the text it was written from and the AI's review. */
export async function getIdeaScreen(projectId: string): Promise<IdeaScreen> {
  const w = await apiFetch<{
    idea: ideaWire | null;
    subject: string;
    language: string;
    input_changed: boolean;
    review: ideaReviewWire | null;
    review_stale: boolean;
  }>(`/v1/projects/${projectId}/idea`);
  return {
    idea: w.idea ? fromWireIdea(w.idea) : null,
    subject: w.subject,
    language: w.language === "vi" || w.language === "en" ? w.language : "",
    inputChanged: w.input_changed,
    review: w.review ? fromWireIdeaReview(w.review) : null,
    reviewStale: w.review_stale,
  };
}

/**
 * Has the AI write the idea from the Creator's text, keeping the fields the
 * Creator wrote (a model call: pass an operation id to follow it). 503 when no
 * language model is configured, 502 when the reply could not be read.
 */
export async function generateProjectIdea(projectId: string, note: string, operation?: AiCall): Promise<ProjectIdea> {
  const wire = await apiFetch<ideaWire>(`/v1/projects/${projectId}/idea/generate`, {
    method: "POST",
    headers: { "Content-Type": "application/json", ...operationInit(operation).headers },
    signal: operationInit(operation).signal,
    body: operationBody({ note }, operation),
  });
  return fromWireIdea(wire);
}

/** A hand edit: only the fields given change. */
export interface ProjectIdeaPatch {
  centralQuestion?: string;
  curiosityPoints?: string[];
  scope?: string;
}

/** Saves a hand edit. 400 when the list holds more than the server allows. */
export async function editProjectIdea(projectId: string, patch: ProjectIdeaPatch): Promise<ProjectIdea> {
  const body: Record<string, unknown> = {};
  if (patch.centralQuestion !== undefined) body.central_question = patch.centralQuestion;
  if (patch.curiosityPoints !== undefined) body.curiosity_points = patch.curiosityPoints;
  if (patch.scope !== undefined) body.scope = patch.scope;
  const wire = await apiFetch<ideaWire>(`/v1/projects/${projectId}/idea`, {
    method: "PUT",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(body),
  });
  return fromWireIdea(wire);
}

/** Approves the idea; 400 while a field is empty. */
export async function approveProjectIdea(projectId: string): Promise<ProjectIdea> {
  return fromWireIdea(await apiFetch<ideaWire>(`/v1/projects/${projectId}/idea/approve`, { method: "POST" }));
}

/** Takes the approval back. */
export async function unapproveProjectIdea(projectId: string): Promise<ProjectIdea> {
  return fromWireIdea(await apiFetch<ideaWire>(`/v1/projects/${projectId}/idea/unapprove`, { method: "POST" }));
}

/** Has the AI read the idea for clarity, researchability and scope (a model call). */
export async function reviewProjectIdea(projectId: string, operation?: AiCall): Promise<IdeaReview> {
  const wire = await apiFetch<ideaReviewWire>(`/v1/projects/${projectId}/idea/review`, {
    method: "POST",
    headers: { ...JSON_HEADERS, ...operationInit(operation).headers },
    signal: operationInit(operation).signal,
    body: operationBody(undefined, operation),
  });
  return fromWireIdeaReview(wire);
}

/** Where a brief stands in review. */
export type ProjectBriefStatus = "pending" | "approved";

/** The brief's fields, in the order the screen shows them. */
export const PROJECT_BRIEF_FIELDS = ["audience", "duration_seconds", "tone", "learning_objectives", "avoid"] as const;
export type ProjectBriefField = (typeof PROJECT_BRIEF_FIELDS)[number];

/** The brief the AI made of the approved idea, with what review needs around it. */
export interface ProjectBrief {
  audience: string;
  /** Target length in seconds; the screen shows it as h:mm:ss. */
  durationSeconds: number;
  tone: string;
  learningObjectives: string[];
  avoid: string[];
  /** Fields the Creator wrote or changed; the AI keeps them when it writes again. */
  creatorFields: ProjectBriefField[];
  status: ProjectBriefStatus;
  redoCount: number;
  approvedAt: string | null;
  updatedAt: string;
}

/** The AI's reading of a brief before approval; advice only. */
export interface BriefReview {
  audienceFit: IdeaAspect;
  durationFit: IdeaAspect;
  objectives: IdeaAspect;
  avoidList: IdeaAspect;
  unclear: string[];
  suggestions: string[];
  createdAt: string;
}

/** The approved idea a brief is written from. */
export interface BriefIdea {
  centralQuestion: string;
  curiosityPoints: string[];
  scope: string;
}

/** Everything the brief screen shows. */
export interface BriefScreen {
  brief: ProjectBrief | null;
  /** The approved idea from step 1; null while it is not approved. */
  idea: BriefIdea | null;
  subject: string;
  language: "vi" | "en" | "";
  videoMode: "long" | "short";
  /** The lengths, in seconds, a brief may set for this video form. */
  minDurationSeconds: number;
  maxDurationSeconds: number;
  /** The approved idea is not the one the brief was written from. */
  ideaChanged: boolean;
  review: BriefReview | null;
  /** The brief changed after the review read it. */
  reviewStale: boolean;
}

interface projectBriefWire {
  audience: string;
  duration_seconds: number;
  tone: string;
  learning_objectives: string[] | null;
  avoid: string[] | null;
  creator_fields: string[] | null;
  status: ProjectBriefStatus;
  redo_count: number;
  approved_at: string | null;
  updated_at: string;
}

interface briefReviewWire {
  audience_fit: IdeaAspect;
  duration_fit: IdeaAspect;
  objectives: IdeaAspect;
  avoid_list: IdeaAspect;
  unclear: string[] | null;
  suggestions: string[] | null;
  created_at: string;
}

function fromWireProjectBrief(w: projectBriefWire): ProjectBrief {
  return {
    audience: w.audience,
    durationSeconds: w.duration_seconds,
    tone: w.tone,
    learningObjectives: w.learning_objectives ?? [],
    avoid: w.avoid ?? [],
    creatorFields: (w.creator_fields ?? []).filter((f): f is ProjectBriefField =>
      (PROJECT_BRIEF_FIELDS as readonly string[]).includes(f),
    ),
    status: w.status,
    redoCount: w.redo_count,
    approvedAt: w.approved_at,
    updatedAt: w.updated_at,
  };
}

function fromWireBriefReview(w: briefReviewWire): BriefReview {
  return {
    audienceFit: w.audience_fit,
    durationFit: w.duration_fit,
    objectives: w.objectives,
    avoidList: w.avoid_list,
    unclear: w.unclear ?? [],
    suggestions: w.suggestions ?? [],
    createdAt: w.created_at,
  };
}

/** Reads the second step: the brief, the approved idea it comes from and the AI's review. */
export async function getBriefScreen(projectId: string): Promise<BriefScreen> {
  const w = await apiFetch<{
    brief: projectBriefWire | null;
    idea: { central_question: string; curiosity_points: string[] | null; scope: string } | null;
    subject: string;
    language: string;
    video_mode: string;
    min_duration_seconds: number;
    max_duration_seconds: number;
    idea_changed: boolean;
    review: briefReviewWire | null;
    review_stale: boolean;
  }>(`/v1/projects/${projectId}/creative-brief`);
  return {
    brief: w.brief ? fromWireProjectBrief(w.brief) : null,
    idea: w.idea
      ? { centralQuestion: w.idea.central_question, curiosityPoints: w.idea.curiosity_points ?? [], scope: w.idea.scope }
      : null,
    subject: w.subject,
    language: w.language === "vi" || w.language === "en" ? w.language : "",
    videoMode: w.video_mode === "short" ? "short" : "long",
    minDurationSeconds: w.min_duration_seconds,
    maxDurationSeconds: w.max_duration_seconds,
    ideaChanged: w.idea_changed,
    review: w.review ? fromWireBriefReview(w.review) : null,
    reviewStale: w.review_stale,
  };
}

/**
 * Has the AI write the brief from the approved idea, keeping the fields the
 * Creator wrote (a model call: pass an operation id to follow it). 409 while
 * the idea is not approved, 503 when no language model is configured, 502 when
 * the reply could not be read.
 */
export async function generateProjectBrief(projectId: string, note: string, operation?: AiCall): Promise<ProjectBrief> {
  const wire = await apiFetch<projectBriefWire>(`/v1/projects/${projectId}/creative-brief/generate`, {
    method: "POST",
    headers: { "Content-Type": "application/json", ...operationInit(operation).headers },
    signal: operationInit(operation).signal,
    body: operationBody({ note }, operation),
  });
  return fromWireProjectBrief(wire);
}

/** A hand edit: only the fields given change. */
export interface ProjectBriefPatch {
  audience?: string;
  durationSeconds?: number;
  tone?: string;
  learningObjectives?: string[];
  avoid?: string[];
}

/** Saves a hand edit. 400 when a list holds more lines than the server allows. */
export async function editProjectBrief(projectId: string, patch: ProjectBriefPatch): Promise<ProjectBrief> {
  const body: Record<string, unknown> = {};
  if (patch.audience !== undefined) body.audience = patch.audience;
  if (patch.durationSeconds !== undefined) body.duration_seconds = patch.durationSeconds;
  if (patch.tone !== undefined) body.tone = patch.tone;
  if (patch.learningObjectives !== undefined) body.learning_objectives = patch.learningObjectives;
  if (patch.avoid !== undefined) body.avoid = patch.avoid;
  const wire = await apiFetch<projectBriefWire>(`/v1/projects/${projectId}/creative-brief`, {
    method: "PUT",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(body),
  });
  return fromWireProjectBrief(wire);
}

/** Approves the brief; 400 while a field is empty or the length is outside the video form's range. */
export async function approveProjectBrief(projectId: string): Promise<ProjectBrief> {
  return fromWireProjectBrief(await apiFetch<projectBriefWire>(`/v1/projects/${projectId}/creative-brief/approve`, { method: "POST" }));
}

/** Takes the approval back. */
export async function unapproveProjectBrief(projectId: string): Promise<ProjectBrief> {
  return fromWireProjectBrief(await apiFetch<projectBriefWire>(`/v1/projects/${projectId}/creative-brief/unapprove`, { method: "POST" }));
}

/** Has the AI read the brief for audience, length, objectives and the avoid list (a model call). */
export async function reviewProjectBrief(projectId: string, operation?: AiCall): Promise<BriefReview> {
  const wire = await apiFetch<briefReviewWire>(`/v1/projects/${projectId}/creative-brief/review`, {
    method: "POST",
    headers: { ...JSON_HEADERS, ...operationInit(operation).headers },
    signal: operationInit(operation).signal,
    body: operationBody(undefined, operation),
  });
  return fromWireBriefReview(wire);
}

// --- Ngôn ngữ hình -------------------------------------------------------------

/** Ba văn bản luật của một phiên bản ngôn ngữ hình. */
export interface VisualLanguageRules {
  illustration_rules: string;
  backdrop_rules: string;
  scene_rules: string;
}

export interface VisualLanguage {
  id: string;
  name: string;
  /** Ghi chú tham khảo cho người xem, không đưa vào prompt. */
  note: string;
  description: string;
  kit_key: string;
  is_system: boolean;
  is_active: boolean;
  /** Phiên bản mới nhất và luật của nó. */
  version: number;
  rules: VisualLanguageRules;
  created_at: string;
}

/** Ngôn ngữ (và phiên bản) một project đã ghim lúc tạo. */
export interface PinnedVisualLanguage {
  visual_language_id: string;
  visual_language_version: number;
  kit_key: string;
  name: string;
  rules: VisualLanguageRules;
}

/** Một ảnh mẫu của kit: cảnh "fig" (lòng quả sung) hoặc "space" (vũ trụ), khung "wide" 16:9 hoặc "tall" 9:16. */
export interface LanguageSample {
  kit_key: string;
  kit_version: number;
  scene: "fig" | "space";
  frame: "wide" | "tall";
  file: string;
}

export type VisualLanguageInput = {
  name: string;
  note: string;
  description: string;
  rules: VisualLanguageRules;
};

export function listVisualLanguages(): Promise<VisualLanguage[]> {
  return apiFetch<VisualLanguage[]>("/v1/visual-languages");
}

export function activateVisualLanguage(id: string): Promise<void> {
  return apiFetch<void>(`/v1/admin/visual-languages/${id}/activate`, { method: "POST" });
}

/** Sao chép một ngôn ngữ (kể cả của hệ thống) thành bản của Creator. */
export function copyVisualLanguage(id: string): Promise<VisualLanguage> {
  return apiFetch<VisualLanguage>(`/v1/admin/visual-languages/${id}/copy`, { method: "POST" });
}

/** Sửa một ngôn ngữ của Creator: tạo phiên bản mới; video đã ghim giữ bản cũ. Ngôn ngữ hệ thống trả 403. */
export function updateVisualLanguage(id: string, input: VisualLanguageInput): Promise<VisualLanguage> {
  return apiFetch<VisualLanguage>(`/v1/admin/visual-languages/${id}`, {
    method: "PUT",
    headers: JSON_HEADERS,
    body: JSON.stringify(input),
  });
}

export function deleteVisualLanguage(id: string): Promise<void> {
  return apiFetch<void>(`/v1/admin/visual-languages/${id}`, { method: "DELETE" });
}

export function listLanguageSamples(id: string): Promise<LanguageSample[]> {
  return apiFetch<LanguageSample[]>(`/v1/visual-languages/${id}/samples`);
}

/** Dựng lại ảnh mẫu của kit (rendering cache theo nội dung nên cái không đổi trả về ngay). */
export function rebuildLanguageSamples(id: string): Promise<LanguageSample[]> {
  return apiFetch<LanguageSample[]>(`/v1/admin/visual-languages/${id}/samples`, { method: "POST" });
}

export function getProjectVisualLanguage(projectId: string): Promise<PinnedVisualLanguage> {
  return apiFetch<PinnedVisualLanguage>(`/v1/projects/${projectId}/visual-language`);
}

/** Đường dẫn ảnh mẫu do gateway phát từ thư mục ảnh tĩnh dùng chung. */
export function languageSampleUrl(sample: Pick<LanguageSample, "file">): string {
  return `${GATEWAY_URL}/v1/stills/${sample.file}`;
}

/** What a fact is worth: settled, still argued, not known, or a common mistaken belief. */
export type FactType = "fact" | "hypothesis" | "unknown" | "misconception";
export const FACT_TYPES: readonly FactType[] = ["fact", "hypothesis", "unknown", "misconception"];

/** How the check of a fact's source link stands: found, not found, unreadable, no link, or running. */
export type LinkState = "ok" | "bad" | "na" | "none" | "run";

/** One research question; `mine` is a question the Creator wrote (a regeneration keeps it). */
export interface ResearchQuestion {
  id: string;
  text: string;
  mine: boolean;
  /** Written by the AI from a brief that has changed since. */
  stale: boolean;
}

export interface FactLinkCheck {
  state: LinkState;
  why: string;
  /** The page sentence closest to a quote that was not found. */
  nearest: string;
  checkedAt: string;
}

/** One fact of the table with its source. `mine` lists the fields the Creator wrote. */
export interface ResearchFact {
  id: string;
  question: string;
  claimVi: string;
  type: FactType;
  sourceTitle: string;
  publisher: string;
  year: string;
  url: string;
  quote: string;
  confidence: string;
  note: string;
  must: boolean;
  keep: boolean;
  link: FactLinkCheck;
  mine: string[];
}

/** One answer the Creator pasted: what was read from it and what was left out. */
export interface PasteLog {
  n: number;
  at: string;
  added: string[];
  updated: string[];
  rejected: { row: number; reason: string }[];
}

export interface QuestionCoverage {
  question: string;
  sources: number;
  badFacts: string[];
  /** Needs another round: under two independent sources, or a fact whose quote was not found. */
  weak: boolean;
}

export interface ResearchGate {
  status: UnitStatus;
  skipped: boolean;
  /** The step no longer holds the flow back: the table is approved or the step skipped. */
  resolved: boolean;
}

/** The whole research screen as the server holds it. */
/** The approved idea (step 1) the research is planned from. */
export interface ResearchIdea {
  centralQuestion: string;
  curiosityPoints: string[];
  scope: string;
}

/** The approved brief (step 2) the research is planned from. */
export interface ResearchBrief {
  audience: string;
  durationSeconds: number;
  tone: string;
  learningObjectives: string[];
  avoid: string[];
}

export interface ResearchState {
  gate: ResearchGate;
  questions: ResearchQuestion[];
  facts: ResearchFact[];
  pastes: PasteLog[];
  coverage: QuestionCoverage[];
  briefApproved: boolean;
  questionsStale: boolean;
  /** Null while the idea is not approved. */
  idea: ResearchIdea | null;
  /** Null while the brief is not approved. */
  brief: ResearchBrief | null;
}

interface researchFactWire {
  id: string;
  question: string;
  claim_vi: string;
  type: FactType;
  source_title: string;
  publisher: string;
  year: string;
  url: string;
  quote: string;
  confidence: string;
  note: string;
  must: boolean;
  keep: boolean;
  link: { state: LinkState; why?: string; nearest?: string; checked_at?: string };
  mine: string[] | null;
}

interface pasteLogWire {
  n: number;
  at: string;
  added: string[] | null;
  updated: string[] | null;
  rejected: { row: number; reason: string }[] | null;
}

interface researchStateWire {
  gate: ResearchGate;
  questions: ResearchQuestion[] | null;
  facts: researchFactWire[] | null;
  pastes: pasteLogWire[] | null;
  coverage: { question: string; sources: number; bad_facts: string[] | null; weak: boolean }[] | null;
  brief_approved: boolean;
  questions_stale: boolean;
  idea: { central_question: string; curiosity_points: string[] | null; scope: string } | null;
  brief: {
    audience: string;
    duration_seconds: number;
    tone: string;
    learning_objectives: string[] | null;
    avoid: string[] | null;
  } | null;
}

function fromWireFact(w: researchFactWire): ResearchFact {
  return {
    id: w.id,
    question: w.question,
    claimVi: w.claim_vi,
    type: w.type,
    sourceTitle: w.source_title,
    publisher: w.publisher,
    year: w.year,
    url: w.url,
    quote: w.quote,
    confidence: w.confidence,
    note: w.note,
    must: w.must,
    keep: w.keep,
    link: { state: w.link.state, why: w.link.why ?? "", nearest: w.link.nearest ?? "", checkedAt: w.link.checked_at ?? "" },
    mine: w.mine ?? [],
  };
}

function fromWirePaste(w: pasteLogWire): PasteLog {
  return { n: w.n, at: w.at, added: w.added ?? [], updated: w.updated ?? [], rejected: w.rejected ?? [] };
}

/** Reads the research step: questions, facts with their link checks, paste log, coverage. */
export async function getResearch(projectId: string): Promise<ResearchState> {
  const w = await apiFetch<researchStateWire>(`/v1/projects/${projectId}/research`);
  return {
    gate: w.gate,
    questions: w.questions ?? [],
    facts: (w.facts ?? []).map(fromWireFact),
    pastes: (w.pastes ?? []).map(fromWirePaste),
    coverage: (w.coverage ?? []).map((c) => ({ question: c.question, sources: c.sources, badFacts: c.bad_facts ?? [], weak: c.weak })),
    briefApproved: w.brief_approved,
    questionsStale: w.questions_stale,
    idea: w.idea
      ? { centralQuestion: w.idea.central_question, curiosityPoints: w.idea.curiosity_points ?? [], scope: w.idea.scope }
      : null,
    brief: w.brief
      ? {
          audience: w.brief.audience,
          durationSeconds: w.brief.duration_seconds,
          tone: w.brief.tone,
          learningObjectives: w.brief.learning_objectives ?? [],
          avoid: w.brief.avoid ?? [],
        }
      : null,
  };
}

/**
 * Has the AI write the research questions from the approved brief (a model
 * call: pass an operation id to follow it). The Creator's own questions stay
 * unless `replaceEdited`. 409 without a brief, 503 without a language model.
 */
export async function generateResearchQuestions(
  projectId: string,
  options: { replaceEdited: boolean; note?: string },
  operation?: AiCall,
): Promise<ResearchQuestion[]> {
  const res = await apiFetch<{ questions: ResearchQuestion[] | null }>(`/v1/projects/${projectId}/research/questions/generate`, {
    method: "POST",
    headers: { ...JSON_HEADERS, ...operationInit(operation).headers },
    signal: operationInit(operation).signal,
    body: operationBody({ replace_edited: options.replaceEdited, note: options.note ?? "" }, operation),
  });
  return res.questions ?? [];
}

/** Adds a question the Creator wrote. */
export async function addResearchQuestion(projectId: string, text: string): Promise<ResearchQuestion> {
  return apiFetch<ResearchQuestion>(`/v1/projects/${projectId}/research/questions`, {
    method: "POST",
    headers: JSON_HEADERS,
    body: JSON.stringify({ text }),
  });
}

/** Removes a question; the facts that answered it stay. */
export async function deleteResearchQuestion(projectId: string, id: string): Promise<void> {
  await apiFetch<undefined>(`/v1/projects/${projectId}/research/questions/${encodeURIComponent(id)}`, { method: "DELETE" });
}

/** The unit a research question is saved through. */
function researchQuestionUnit(id: string): UnitRef<unknown> {
  return { kind: "research_question", key: id, parse: (c) => c };
}

/** Saves the Creator's wording of a question; the question becomes theirs. */
export async function editResearchQuestion(projectId: string, id: string, text: string): Promise<void> {
  await editUnit(projectId, researchQuestionUnit(id), { text });
}

/** The prompt to copy into an outside research tool. */
export async function getResearchPrompt(projectId: string): Promise<string> {
  return (await apiFetch<{ prompt: string }>(`/v1/projects/${projectId}/research/prompt`)).prompt;
}

/** The prompt for another round: only the questions still short of sources. 409 when none is. */
export async function getFollowUpPrompt(projectId: string): Promise<string> {
  return (await apiFetch<{ prompt: string }>(`/v1/projects/${projectId}/research/follow-up-prompt`)).prompt;
}

/** What reading a pasted answer did. 400 when the answer holds no research JSON block. */
export interface PasteResult {
  added: string[];
  updated: string[];
  rejected: { row: number; reason: string }[];
  log: PasteLog;
}

export async function pasteResearch(projectId: string, raw: string): Promise<PasteResult> {
  const w = await apiFetch<{
    added: string[] | null;
    updated: string[] | null;
    rejected: { row: number; reason: string }[] | null;
    log: pasteLogWire;
  }>(`/v1/projects/${projectId}/research/paste`, {
    method: "POST",
    headers: JSON_HEADERS,
    body: JSON.stringify({ raw }),
  });
  return { added: w.added ?? [], updated: w.updated ?? [], rejected: w.rejected ?? [], log: fromWirePaste(w.log) };
}

/** A fact the Creator adds by hand. */
export interface FactInput {
  question: string;
  claimVi: string;
  type: FactType;
  sourceTitle?: string;
  publisher?: string;
  year?: string;
  url?: string;
  quote?: string;
}

export async function addFact(projectId: string, input: FactInput): Promise<ResearchFact> {
  const w = await apiFetch<researchFactWire>(`/v1/projects/${projectId}/research/facts`, {
    method: "POST",
    headers: JSON_HEADERS,
    body: JSON.stringify({
      question: input.question,
      claim_vi: input.claimVi,
      type: input.type,
      source_title: input.sourceTitle ?? "",
      publisher: input.publisher ?? "",
      year: input.year ?? "",
      url: input.url ?? "",
      quote: input.quote ?? "",
    }),
  });
  return fromWireFact(w);
}

/** Changes a fact's type, must-say mark or kept/dropped state straight in the table. */
export async function setFactFlags(
  projectId: string,
  id: string,
  flags: { type?: FactType; must?: boolean; keep?: boolean },
): Promise<ResearchFact> {
  const w = await apiFetch<researchFactWire>(`/v1/projects/${projectId}/research/facts/${encodeURIComponent(id)}`, {
    method: "PUT",
    headers: JSON_HEADERS,
    body: JSON.stringify(flags),
  });
  return fromWireFact(w);
}

/** The text fields of a fact the Creator can edit, by the names the server uses. */
export type FactTextField = "claim_vi" | "source_title" | "publisher" | "year" | "url" | "quote" | "note" | "question";

function factUnit(id: string): UnitRef<unknown> {
  return { kind: "fact", key: id, parse: (c) => c };
}

/** Saves the Creator's changes to a fact's text; a new link or quote is checked again. */
export async function editFact(projectId: string, id: string, fields: Partial<Record<FactTextField, string>>): Promise<void> {
  await editUnit(projectId, factUnit(id), fields as Record<string, string>);
}

/** What changing or dropping a fact sends back in later steps. Empty while no later step uses it. */
export async function getFactImpact(projectId: string, id: string): Promise<string[]> {
  return getUnitImpact(projectId, factUnit(id));
}

/** Checks the links of the given facts again (every kept fact with a link when none is named). */
export async function verifyFacts(projectId: string, factIds: string[]): Promise<void> {
  await apiFetch<undefined>(`/v1/projects/${projectId}/research/verify`, {
    method: "POST",
    headers: JSON_HEADERS,
    body: JSON.stringify({ fact_ids: factIds }),
  });
}

/** Skips the research step, or comes back to do it. */
export async function setResearchSkipped(projectId: string, skipped: boolean): Promise<ResearchGate> {
  return apiFetch<ResearchGate>(`/v1/projects/${projectId}/research/skip`, {
    method: "PUT",
    headers: JSON_HEADERS,
    body: JSON.stringify({ skipped }),
  });
}

const RESEARCH_GATE_UNIT: UnitRef<unknown> = { kind: "research", key: "main", parse: (c) => c };

/** Approves the fact table. 400 when it holds no kept fact. */
export async function approveResearch(projectId: string): Promise<void> {
  await approveUnit(projectId, RESEARCH_GATE_UNIT);
}

/** Takes the approval of the fact table back. */
export async function unapproveResearch(projectId: string): Promise<void> {
  await unapproveUnit(projectId, RESEARCH_GATE_UNIT);
}

/** The fields of the story, in the order the screen shows them. */
export const STORY_FIELD_KEYS = ["central_question", "core_answer", "narrative_arc"] as const;
export type StoryFieldKey = (typeof STORY_FIELD_KEYS)[number];

/** The most beats the narrative arc may hold, and the fewest. */
export const MAX_ARC_BEATS = 12;
export const MIN_ARC_BEATS = 3;

/** One beat of the narrative arc: its role in the telling, what it says, and the research facts it rests on. */
export interface ArcBeat {
  beat: string;
  text: string;
  facts: string[];
}

/** The story the AI or the Creator wrote. */
export interface Story {
  centralQuestion: string;
  coreAnswer: string;
  narrativeArc: ArcBeat[];
  /** The fields the Creator wrote or edited; an AI run keeps them. */
  creatorFields: StoryFieldKey[];
  status: UnitStatus;
  updatedAt: string;
}

/** One kept research fact as the story step lists it. */
export interface StoryFact {
  id: string;
  claimVi: string;
  type: FactType;
  /** The script has to state it. */
  must: boolean;
  /** A beat of the arc rests on it. */
  cited: boolean;
  link: LinkState;
}

/** Everything the story screen shows. */
export interface StoryState {
  /** The story's status; "pending" while there is no story. */
  status: UnitStatus;
  /** null until the AI or the Creator wrote one. */
  story: Story | null;
  /** The kept research facts the AI reads. */
  facts: StoryFact[];
  /** The approved brief the AI reads; null while the brief step is not approved. */
  brief: ResearchBrief | null;
  briefApproved: boolean;
  researchResolved: boolean;
  researchSkipped: boolean;
  /** The approved brief or the kept facts differ from the ones the story was written from. */
  briefStale: boolean;
  researchStale: boolean;
}

interface storyWire {
  central_question: string;
  core_answer: string;
  narrative_arc: { beat: string; text: string; facts: string[] | null }[] | null;
  creator_fields: StoryFieldKey[] | null;
  status: UnitStatus;
  updated_at: string;
}

interface storyStateWire {
  story: storyWire | null;
  facts: { id: string; claim_vi: string; type: FactType; must: boolean; cited: boolean; link: LinkState }[] | null;
  brief: {
    audience: string;
    duration_seconds: number;
    tone: string;
    learning_objectives: string[] | null;
    avoid: string[] | null;
  } | null;
  brief_approved: boolean;
  research_resolved: boolean;
  research_skipped: boolean;
  brief_stale: boolean;
  research_stale: boolean;
}

function fromWireStoryBody(w: storyWire): Story {
  return {
    centralQuestion: w.central_question,
    coreAnswer: w.core_answer,
    narrativeArc: (w.narrative_arc ?? []).map((b) => ({ beat: b.beat, text: b.text, facts: b.facts ?? [] })),
    creatorFields: w.creator_fields ?? [],
    status: w.status,
    updatedAt: w.updated_at,
  };
}

function fromWireStory(w: storyStateWire): StoryState {
  return {
    status: w.story?.status ?? "pending",
    story: w.story ? fromWireStoryBody(w.story) : null,
    facts: (w.facts ?? []).map((f) => ({
      id: f.id,
      claimVi: f.claim_vi,
      type: f.type,
      must: f.must,
      cited: f.cited,
      link: f.link,
    })),
    brief: w.brief
      ? {
          audience: w.brief.audience,
          durationSeconds: w.brief.duration_seconds,
          tone: w.brief.tone,
          learningObjectives: w.brief.learning_objectives ?? [],
          avoid: w.brief.avoid ?? [],
        }
      : null,
    briefApproved: w.brief_approved,
    researchResolved: w.research_resolved,
    researchSkipped: w.research_skipped,
    briefStale: w.brief_stale,
    researchStale: w.research_stale,
  };
}

/** Reads the story step: the story, the kept facts and brief the AI reads, and what the step waits for. */
export async function getStory(projectId: string): Promise<StoryState> {
  return fromWireStory(await apiFetch<storyStateWire>(`/v1/projects/${projectId}/story`));
}

/**
 * Has the AI write the story from the approved brief and the kept facts,
 * keeping the fields the Creator wrote (a model call: pass an operation id to
 * follow it). The story goes to review. 409 while the brief or the research
 * step is not done, 503 without a language model, 502 when the reply could
 * not be read.
 */
export async function generateStory(projectId: string, note: string, operation?: AiCall): Promise<Story> {
  return fromWireStoryBody(
    await apiFetch<storyWire>(`/v1/projects/${projectId}/story/generate`, {
      method: "POST",
      headers: { ...JSON_HEADERS, ...operationInit(operation).headers },
      signal: operationInit(operation).signal,
      body: operationBody({ note }, operation),
    }),
  );
}

/** A hand edit: only the fields given change. */
export interface StoryPatch {
  centralQuestion?: string;
  coreAnswer?: string;
  narrativeArc?: ArcBeat[];
}

/** Saves a hand edit. 400 for a beat citing a fact the table does not hold, or more beats than allowed. */
export async function editStory(projectId: string, patch: StoryPatch): Promise<Story> {
  const body: Record<string, unknown> = {};
  if (patch.centralQuestion !== undefined) body.central_question = patch.centralQuestion;
  if (patch.coreAnswer !== undefined) body.core_answer = patch.coreAnswer;
  if (patch.narrativeArc !== undefined) body.narrative_arc = patch.narrativeArc;
  return fromWireStoryBody(
    await apiFetch<storyWire>(`/v1/projects/${projectId}/story`, {
      method: "PUT",
      headers: JSON_HEADERS,
      body: JSON.stringify(body),
    }),
  );
}

/** Approves the story; 400 while a field is empty or the arc has fewer than three beats. */
export async function approveStory(projectId: string): Promise<Story> {
  return fromWireStoryBody(await apiFetch<storyWire>(`/v1/projects/${projectId}/story/approve`, { method: "POST" }));
}

/** Takes the approval back. */
export async function unapproveStory(projectId: string): Promise<Story> {
  return fromWireStoryBody(await apiFetch<storyWire>(`/v1/projects/${projectId}/story/unapprove`, { method: "POST" }));
}

/** The functions a beat can have, in the order a video usually runs them. */
export const BEAT_FNS = ["Hook", "Đặt vấn đề", "Giải thích cơ chế", "Cao trào", "Mở rộng hệ quả", "Payoff"] as const;
export type BeatFn = (typeof BEAT_FNS)[number];

/** How far the duration buttons move a beat, in seconds. */
export const BEAT_DURATION_STEP = 5;

/** The shortest a beat may be, in seconds. */
export const BEAT_MIN_SECONDS = 5;

/** The fields of a beat the Creator edits as text in the dialog. */
export type BeatTextField = "title" | "goal" | "content" | "next";

/** One beat of the sheet. `mine` names the fields the Creator wrote; an AI run keeps them. */
export interface Beat {
  key: string;
  fn: BeatFn;
  title: string;
  goal: string;
  content: string;
  facts: string[];
  /** Target length in seconds. */
  dur: number;
  /** How the beat joins the one after it; "" for the last beat. */
  next: string;
  status: UnitStatus;
  mine: string[];
  stale: boolean;
  /** Facts the script must state that no other beat holds, so deleting this beat drops them. */
  onlyMust: string[];
  /** The beat is not the hook and cites no fact, so nothing backs what it says. */
  noFact: boolean;
  /** The story line the beat was built from; "" for a beat the Creator added. */
  step: string;
}

/** One kept fact the beats can use. */
export interface BeatFact {
  id: string;
  claimVi: string;
  type: FactType;
  must: boolean;
  link: LinkState;
}

/** What the AI reads besides the kept facts, as the prompt gets it. */
export interface BeatInput {
  topic: string;
  /** The approved brief; "" while the brief step is not approved. */
  brief: string;
  briefApproved: boolean;
  /** The approved story in the fields the beats are built from; "" while the story is not approved. */
  story: string;
}

/** One line of the structure check. */
export interface BeatCheck {
  ok: boolean;
  text: string;
}

/** Where the step stands: no beat yet, reviewing the skeleton, or writing and reviewing the content. */
export type BeatPhase = "empty" | "skeleton" | "content";

/** Everything the beat sheet screen shows. */
export interface BeatState {
  /** Approved when the sheet has beats and every one is approved. */
  status: UnitStatus;
  storyApproved: boolean;
  beats: Beat[];
  facts: BeatFact[];
  input: BeatInput;
  checks: BeatCheck[];
  totalSeconds: number;
  targetLabel: string;
  phase: BeatPhase;
  /** Beats whose goal and content are not written yet. */
  unwritten: number;
  /** Beats the last content run resized so the total fits the target. */
  rebalanced: number;
  /** How many beats have approved dialogue; moving a beat changes how it joins its neighbours. */
  scriptApproved: number;
}

interface beatStateWire {
  status: UnitStatus;
  story_approved: boolean;
  beats:
    | {
        key: string;
        fn: BeatFn;
        title: string;
        goal: string;
        content: string;
        facts: string[] | null;
        dur: number;
        next: string;
        status: UnitStatus;
        mine: string[] | null;
        stale: boolean;
        only_must: string[] | null;
        no_fact: boolean | null;
        step: string | null;
      }[]
    | null;
  facts: { id: string; claim_vi: string; type: FactType; must: boolean; link: LinkState }[] | null;
  input: { topic: string; brief: string; brief_approved: boolean; story: string } | null;
  checks: BeatCheck[] | null;
  total_seconds: number;
  target_label: string;
  phase: BeatPhase;
  unwritten: number;
  rebalanced: number;
  script_approved: number;
}

function fromWireBeats(w: beatStateWire): BeatState {
  return {
    status: w.status,
    storyApproved: w.story_approved,
    beats: (w.beats ?? []).map((b) => ({ ...b, facts: b.facts ?? [], mine: b.mine ?? [], onlyMust: b.only_must ?? [], noFact: b.no_fact ?? false, step: b.step ?? "" })),
    facts: (w.facts ?? []).map((f) => ({ id: f.id, claimVi: f.claim_vi, type: f.type, must: f.must, link: f.link })),
    input: { topic: w.input?.topic ?? "", brief: w.input?.brief ?? "", briefApproved: w.input?.brief_approved ?? false, story: w.input?.story ?? "" },
    checks: w.checks ?? [],
    totalSeconds: w.total_seconds,
    targetLabel: w.target_label,
    phase: w.phase,
    unwritten: w.unwritten,
    rebalanced: w.rebalanced,
    scriptApproved: w.script_approved ?? 0,
  };
}

/** Reads the beat sheet step: the beats in order, the kept facts, the structure checks and whether the story is approved. */
export async function getBeats(projectId: string): Promise<BeatState> {
  return fromWireBeats(await apiFetch<beatStateWire>(`/v1/projects/${projectId}/beats`));
}

/**
 * Has the AI write the whole sheet from the approved story (a model call: pass
 * an operation id to follow it). Beats the Creator wrote any part of keep that
 * part; the others are replaced and every beat goes to review.
 */
export async function generateBeats(projectId: string, note: string, operation?: AiCall, onlyUnwritten = false): Promise<BeatState> {
  return fromWireBeats(
    await apiFetch<beatStateWire>(`/v1/projects/${projectId}/beats/generate`, {
      method: "POST",
      headers: { ...JSON_HEADERS, ...operationInit(operation).headers },
      signal: operationInit(operation).signal,
      body: operationBody({ note, only_unwritten: onlyUnwritten }, operation),
    }),
  );
}

const beatPath = (projectId: string, key: string) => `/v1/projects/${projectId}/beats/${key}`;

/** Saves the Creator's edit of some fields of one beat; the beat goes back to review. */
export async function editBeat(projectId: string, key: string, fields: Record<string, string>): Promise<BeatState> {
  return fromWireBeats(
    await apiFetch<beatStateWire>(beatPath(projectId, key), { method: "PUT", headers: JSON_HEADERS, body: JSON.stringify({ fields }) }),
  );
}

/** Has the AI write one beat again with the Creator's note (a model call: pass an operation id to follow it). */
export async function redoBeat(projectId: string, key: string, note: string, operation?: AiCall): Promise<BeatState> {
  return fromWireBeats(
    await apiFetch<beatStateWire>(`${beatPath(projectId, key)}/redo`, {
      method: "POST",
      headers: { ...JSON_HEADERS, ...operationInit(operation).headers },
      signal: operationInit(operation).signal,
      body: operationBody({ note }, operation),
    }),
  );
}

/** Approves one beat; the server refuses a beat without a goal. */
export async function approveBeat(projectId: string, key: string): Promise<BeatState> {
  return fromWireBeats(await apiFetch<beatStateWire>(`${beatPath(projectId, key)}/approve`, { method: "POST" }));
}

/** Takes the approval of one beat back; it waits for review again. */
export async function unapproveBeat(projectId: string, key: string): Promise<BeatState> {
  return fromWireBeats(await apiFetch<beatStateWire>(`${beatPath(projectId, key)}/unapprove`, { method: "POST" }));
}

/** What a change to the beat sends back in later steps (the dialogue written for it), one line per item. */
export async function getBeatImpact(projectId: string, ref: UnitAddress): Promise<string[]> {
  const res = await apiFetch<{ items: string[] | null }>(`${beatPath(projectId, ref.key)}/impact`);
  return res.items ?? [];
}

/**
 * Adds a beat after the beat `after` ("" on an empty sheet). With `blank` the
 * server adds an empty beat for the Creator to fill in, which works without a
 * language model; otherwise the AI writes it (a model call).
 */
export async function addBeat(
  projectId: string,
  after: string,
  blank: boolean,
  note: string,
  operation?: AiCall,
): Promise<BeatState> {
  return fromWireBeats(
    await apiFetch<beatStateWire>(`/v1/projects/${projectId}/beats`, {
      method: "POST",
      headers: { ...JSON_HEADERS, ...operationInit(operation).headers },
      signal: operationInit(operation).signal,
      body: operationBody({ after, blank, note }, operation),
    }),
  );
}

/**
 * Builds the skeleton of the sheet from the approved story, with no model: a
 * hook, one beat per step of the chain of cause and effect, and a payoff. It
 * replaces the beats now on the sheet; the server refuses once the skeleton is
 * approved.
 */
export async function buildBeatSkeleton(projectId: string): Promise<BeatState> {
  return fromWireBeats(await apiFetch<beatStateWire>(`/v1/projects/${projectId}/beats/skeleton`, { method: "POST" }));
}

/** Approves the functions and facts of the beats, so the AI may write their content. */
export async function approveBeatSkeleton(projectId: string): Promise<BeatState> {
  return fromWireBeats(await apiFetch<beatStateWire>(`/v1/projects/${projectId}/beats/skeleton/approve`, { method: "POST" }));
}

/** Sets the order of the beats; `order` names every beat once. */
export async function reorderBeats(projectId: string, order: string[]): Promise<BeatState> {
  return fromWireBeats(
    await apiFetch<beatStateWire>(`/v1/projects/${projectId}/beats/order`, {
      method: "PUT",
      headers: JSON_HEADERS,
      body: JSON.stringify({ order }),
    }),
  );
}

/** Removes a beat from the sheet. */
export async function deleteBeat(projectId: string, key: string): Promise<BeatState> {
  return fromWireBeats(await apiFetch<beatStateWire>(`/v1/projects/${projectId}/beats/${encodeURIComponent(key)}`, { method: "DELETE" }));
}

/** The unit the dialogue of one beat is saved, redone and approved through; the key is the beat's key. */
export function scriptBeatUnit(key: string): UnitRef<unknown> {
  return { kind: "script_beat", key, parse: (c) => c };
}

/** The reading of one line: none (never asked, or the wording changed since), pending, running, ready or failed. */
export type ScriptAudioState = "none" | "pending" | "running" | "ready" | "failed";

export interface ScriptAudio {
  state: ScriptAudioState;
  /** Name of the voice file under /v1/projects/{id}/audio/; set when ready. */
  file?: string;
  error?: string;
}

/** A flag over one line: too long to say in one breath, or hard for a voice to read. */
export interface ScriptLineFlag {
  kind: "Câu dài" | "Khó nghe";
  msg: string;
}

export interface ScriptLine {
  /** `<beat key>.<n>`, never reused inside its beat. */
  id: string;
  text: string;
  /** What the voice reads when it differs from `text` (a formula said in words); absent when the text is read as written. */
  spoken?: string;
  facts: string[];
  /** Length in seconds: the voice file's when there is one for the current wording, else an estimate. */
  seconds: number;
  estimated: boolean;
  flag: ScriptLineFlag | null;
  audio: ScriptAudio;
}

export interface ScriptJoin {
  ok: boolean;
  msg: string;
}

/** The dialogue of one beat as the script screen shows it. */
export interface ScriptBeat {
  key: string;
  /** Position in the sheet from 1. */
  number: number;
  fn: BeatFn;
  title: string;
  goal: string;
  /** The length the beat sheet gave the beat. */
  targetSeconds: number;
  /** The beat has dialogue. */
  written: boolean;
  status: UnitStatus;
  mine: string[];
  /** The beat changed since its dialogue was written. */
  stale: boolean;
  lines: ScriptLine[];
  seconds: number;
  /** How the beat joins the one before it; null for the first beat and when there is nothing to join. */
  join: ScriptJoin | null;
}

export interface ScriptCheck {
  ok: boolean;
  text: string;
}

export interface ScriptStats {
  lines: number;
  beats: number;
  written: number;
  /** Lines with no voice file for their current wording. */
  estimated: number;
  totalSeconds: number;
  openFlags: number;
  openJoins: number;
}

/** Everything the script screen shows. */
export interface ScriptState {
  /** Approved when every beat has dialogue and every beat is approved. */
  status: UnitStatus;
  beatsApproved: boolean;
  beats: ScriptBeat[];
  checks: ScriptCheck[];
  stats: ScriptStats;
  targetLabel: string;
  inRange: boolean;
  /** The voice the lines are read with; "" is the default voice. */
  voiceId: string;
}

interface scriptStateWire {
  status: UnitStatus;
  beats_approved: boolean;
  beats:
    | {
        key: string;
        number: number;
        fn: BeatFn;
        title: string;
        goal: string;
        target_seconds: number;
        written: boolean;
        status: UnitStatus;
        mine: string[] | null;
        stale: boolean;
        lines: { id: string; text: string; facts: string[] | null; seconds: number; estimated: boolean; flag: ScriptLineFlag | null; audio: ScriptAudio }[] | null;
        seconds: number;
        join: ScriptJoin | null;
      }[]
    | null;
  checks: ScriptCheck[] | null;
  stats: {
    lines: number;
    beats: number;
    written: number;
    estimated: number;
    total_seconds: number;
    open_flags: number;
    open_joins: number;
  };
  target_label: string;
  in_range: boolean;
  voice_id: string;
}

function fromWireScript(w: scriptStateWire): ScriptState {
  return {
    status: w.status,
    beatsApproved: w.beats_approved,
    beats: (w.beats ?? []).map((b) => ({
      key: b.key,
      number: b.number,
      fn: b.fn,
      title: b.title,
      goal: b.goal,
      targetSeconds: b.target_seconds,
      written: b.written,
      status: b.status,
      mine: b.mine ?? [],
      stale: b.stale,
      lines: (b.lines ?? []).map((l) => ({ ...l, facts: l.facts ?? [] })),
      seconds: b.seconds,
      join: b.join,
    })),
    checks: w.checks ?? [],
    stats: {
      lines: w.stats.lines,
      beats: w.stats.beats,
      written: w.stats.written,
      estimated: w.stats.estimated,
      totalSeconds: w.stats.total_seconds,
      openFlags: w.stats.open_flags,
      openJoins: w.stats.open_joins,
    },
    targetLabel: w.target_label,
    inRange: w.in_range,
    voiceId: w.voice_id,
  };
}

/** Reads the script step: the dialogue of every beat, its flags, the checks and the reading of each line. */
export async function getScript(projectId: string): Promise<ScriptState> {
  return fromWireScript(await apiFetch<scriptStateWire>(`/v1/projects/${projectId}/script`));
}

/**
 * Has the AI write the dialogue of the beats that have none, one beat after
 * another (a model call: pass an operation id to follow it). With `beat` only
 * that beat, which must have no dialogue yet.
 */
export async function generateScript(projectId: string, beat: string, note: string, operation?: AiCall): Promise<ScriptState> {
  return fromWireScript(
    await apiFetch<scriptStateWire>(`/v1/projects/${projectId}/script/generate`, {
      method: "POST",
      headers: { ...JSON_HEADERS, ...operationInit(operation).headers },
      signal: operationInit(operation).signal,
      body: operationBody({ beat, note }, operation),
    }),
  );
}

/**
 * Asks the AI for one replacement sentence for a flagged line (`line`) or for
 * the first line of a beat that joins badly with the one before it (`join`).
 * Nothing is stored: the Creator applies the text with an edit of the line.
 */
export async function suggestScriptLine(projectId: string, kind: "line" | "join", lineId: string, operation?: AiCall): Promise<string> {
  const reply = await apiFetch<{ text: string }>(`/v1/projects/${projectId}/script/suggest`, {
    method: "POST",
    headers: { ...JSON_HEADERS, ...operationInit(operation).headers },
    signal: operationInit(operation).signal,
    body: operationBody({ kind, line_id: lineId }, operation),
  });
  return reply.text;
}

/** Puts the lines in the queue to be read in the project's voice; lines that already have a current file are left alone. */
export async function speakScriptLines(projectId: string, lineIds: string[]): Promise<ScriptState> {
  return fromWireScript(
    await apiFetch<scriptStateWire>(`/v1/projects/${projectId}/script/speak`, {
      method: "POST",
      headers: JSON_HEADERS,
      body: JSON.stringify({ line_ids: lineIds }),
    }),
  );
}

/** Where a find-and-replace over the dialogue writes: the wording the voice reads, or the text on screen. */
export type ScriptReplaceField = "spoken" | "text";

/** One line a find-and-replace changes, as it reads before and after. */
export interface ScriptReplaceMatch {
  beat: string;
  line: string;
  before: string;
  after: string;
}

/** What a find-and-replace changes, or would change: approved beats are never touched and only counted. */
export interface ScriptReplaceResult {
  matches: ScriptReplaceMatch[];
  skippedApproved: { beat: string; lines: number }[];
  applied: boolean;
}

/** Previews (`apply` false) or applies a whole-word replace over the dialogue of the beats not yet approved. */
export async function replaceInScript(
  projectId: string,
  options: { find: string; replace: string; field: ScriptReplaceField; matchCase: boolean; apply: boolean },
): Promise<ScriptReplaceResult> {
  const wire = await apiFetch<{
    matches: ScriptReplaceMatch[] | null;
    skipped_approved: { beat: string; lines: number }[] | null;
    applied: boolean;
  }>(`/v1/projects/${projectId}/script/replace`, {
    method: "POST",
    headers: JSON_HEADERS,
    body: JSON.stringify({
      find: options.find,
      replace: options.replace,
      field: options.field,
      match_case: options.matchCase,
      apply: options.apply,
    }),
  });
  return { matches: wire.matches ?? [], skippedApproved: wire.skipped_approved ?? [], applied: wire.applied };
}

/** Where the browser plays a line's voice file from. */
export function scriptAudioUrl(projectId: string, file: string): string {
  return `${GATEWAY_URL}/v1/projects/${projectId}/audio/${encodeURIComponent(file)}`;
}

/** The unit the handoff note of one beat is saved, redone and approved through; the key is the beat's key. */
export function handoffBeatUnit(key: string): UnitRef<unknown> {
  return { kind: "handoff_beat", key, parse: (c) => c };
}

/** Where a review finding stands: still to deal with, a passed check, information, dealt with, or accepted with a reason. */
export type FindingState = "open" | "ok" | "info" | "fixed" | "accepted";

/** One line of the review of the whole script. */
export interface ReviewFinding {
  id: string;
  source: "machine" | "ai";
  group: string;
  groupLabel: string;
  state: FindingState;
  title: string;
  detail: string;
  /** The line the finding is about, if any. */
  line: string;
  /** A replacement for `line` the Creator can apply. */
  fix: string;
  /** A line to add to the handoff note of a beat. */
  handoff: { beat: string; section: string; text: string } | null;
  /** The finding is dealt with at the research step. */
  research: boolean;
  why: string;
}

export interface HandoffSection {
  key: string;
  label: string;
  /** The roles that read this section, by key. */
  roles: string[];
}

export interface HandoffNote {
  must: string[];
  terms: string[];
  exact: string[];
  free: string[];
  unsure: string[];
}

export interface LockBeat {
  key: string;
  number: number;
  fn: string;
  title: string;
  goal: string;
  written: boolean;
  status: UnitStatus;
  mine: string[];
  /** The dialogue the note was written from has changed since. */
  stale: boolean;
  note: HandoffNote;
  lines: number;
}

export interface GlossaryTerm {
  term: string;
  say: string;
  draw: string;
  avoid: string;
  mine: boolean;
}

export interface LockCondition {
  ok: boolean;
  text: string;
}

export interface LockFact {
  id: string;
  type: string;
  link: string;
}

export interface LockPackage {
  topic: string;
  audience: string;
  targetLabel: string;
  totalSeconds: number;
  lines: number;
  beats: number;
  chain: { fn: string; title: string; next: string }[];
  facts: LockFact[];
  hypotheses: number;
  unmatched: number;
  noSource: boolean;
}

export interface LockVersion {
  version: number;
  lockedAt: string;
  note: string;
}

/** Everything the lock screen shows. */
export interface ScriptLockState {
  scriptReady: boolean;
  /** The AI has reviewed the script at least once. */
  reviewed: boolean;
  findings: ReviewFinding[];
  openCount: number;
  aiNotes: string[];
  myNotes: string[];
  glossary: GlossaryTerm[];
  beats: LockBeat[];
  conditions: LockCondition[];
  canLock: boolean;
  package: LockPackage;
  handoff: { beats: number; sections: Record<string, number>; glossary: number };
  sections: HandoffSection[];
  versions: LockVersion[];
  lockedVersion: number;
  nextVersion: number;
  /** What changed since the latest locked version. */
  draft: string[];
  skippedResearch: boolean;
}

interface lockStateWire {
  script_ready: boolean;
  reviewed: boolean;
  findings:
    | {
        id: string;
        source: "machine" | "ai";
        group: string;
        group_label: string;
        state: FindingState;
        title: string;
        detail: string;
        line?: string;
        fix?: string;
        handoff?: { beat: string; section: string; text: string } | null;
        research?: boolean;
        why?: string;
      }[]
    | null;
  open_count: number;
  ai_notes: string[] | null;
  my_notes: string[] | null;
  glossary: GlossaryTerm[] | null;
  beats:
    | {
        key: string;
        number: number;
        fn: string;
        title: string;
        goal: string;
        written: boolean;
        status: UnitStatus;
        mine: string[] | null;
        stale: boolean;
        note: Partial<HandoffNote> | null;
        lines: number;
      }[]
    | null;
  conditions: LockCondition[] | null;
  can_lock: boolean;
  package: {
    topic: string;
    audience: string;
    target_label: string;
    total_seconds: number;
    lines: number;
    beats: number;
    chain: { fn: string; title: string; next: string }[] | null;
    facts: LockFact[] | null;
    hypotheses: number;
    unmatched: number;
    no_source: boolean;
  } | null;
  handoff: { beats: number; sections: Record<string, number> | null; glossary: number } | null;
  sections: HandoffSection[] | null;
  versions: { version: number; locked_at: string; note: string }[] | null;
  locked_version: number;
  next_version: number;
  draft: string[] | null;
  skipped_research: boolean;
}

function fromWireLock(w: lockStateWire): ScriptLockState {
  const p = w.package;
  return {
    scriptReady: w.script_ready,
    reviewed: w.reviewed ?? false,
    findings: (w.findings ?? []).map((f) => ({
      id: f.id,
      source: f.source,
      group: f.group,
      groupLabel: f.group_label,
      state: f.state,
      title: f.title,
      detail: f.detail,
      line: f.line ?? "",
      fix: f.fix ?? "",
      handoff: f.handoff ?? null,
      research: f.research ?? false,
      why: f.why ?? "",
    })),
    openCount: w.open_count,
    aiNotes: w.ai_notes ?? [],
    myNotes: w.my_notes ?? [],
    glossary: w.glossary ?? [],
    beats: (w.beats ?? []).map((b) => ({
      ...b,
      mine: b.mine ?? [],
      note: {
        must: b.note?.must ?? [],
        terms: b.note?.terms ?? [],
        exact: b.note?.exact ?? [],
        free: b.note?.free ?? [],
        unsure: b.note?.unsure ?? [],
      },
    })),
    conditions: w.conditions ?? [],
    canLock: w.can_lock,
    package: {
      topic: p?.topic ?? "",
      audience: p?.audience ?? "",
      targetLabel: p?.target_label ?? "",
      totalSeconds: p?.total_seconds ?? 0,
      lines: p?.lines ?? 0,
      beats: p?.beats ?? 0,
      chain: p?.chain ?? [],
      facts: p?.facts ?? [],
      hypotheses: p?.hypotheses ?? 0,
      unmatched: p?.unmatched ?? 0,
      noSource: p?.no_source ?? false,
    },
    handoff: { beats: w.handoff?.beats ?? 0, sections: w.handoff?.sections ?? {}, glossary: w.handoff?.glossary ?? 0 },
    sections: w.sections ?? [],
    versions: (w.versions ?? []).map((v) => ({ version: v.version, lockedAt: v.locked_at, note: v.note })),
    lockedVersion: w.locked_version,
    nextVersion: w.next_version,
    draft: w.draft ?? [],
    skippedResearch: w.skipped_research,
  };
}

async function lockCall(projectId: string, path: string, body?: unknown, operation?: AiCall): Promise<ScriptLockState> {
  const init = operationInit(operation);
  return fromWireLock(
    await apiFetch<lockStateWire>(`/v1/projects/${projectId}/script-lock${path}`, {
      method: "POST",
      headers: { ...JSON_HEADERS, ...init.headers },
      signal: init.signal,
      body: operationBody(body as object | undefined, operation),
    }),
  );
}

/** Reads the lock step: the review, the handoff notes, the glossary and the locked versions. */
export async function getScriptLock(projectId: string): Promise<ScriptLockState> {
  return fromWireLock(await apiFetch<lockStateWire>(`/v1/projects/${projectId}/script-lock`));
}

/** Has the AI read the whole script and report what to deal with (a model call: pass an operation to follow it). */
export function reviewScript(projectId: string, operation?: AiCall): Promise<ScriptLockState> {
  return lockCall(projectId, "/review", undefined, operation);
}

/** Replaces the finding's line with the fix it carries; the beat goes back to review. */
export function applyFinding(projectId: string, findingId: string): Promise<ScriptLockState> {
  return lockCall(projectId, `/findings/${encodeURIComponent(findingId)}/apply`);
}

/** Adds the line a finding proposes to its beat's handoff note. */
export function addFindingToHandoff(projectId: string, findingId: string): Promise<ScriptLockState> {
  return lockCall(projectId, `/findings/${encodeURIComponent(findingId)}/handoff`);
}

/** Accepts a finding as it stands, with the reason. */
export function acceptFinding(projectId: string, findingId: string, why: string): Promise<ScriptLockState> {
  return lockCall(projectId, `/findings/${encodeURIComponent(findingId)}/accept`, { why });
}

export function addReviewNote(projectId: string, text: string): Promise<ScriptLockState> {
  return lockCall(projectId, "/notes", { text });
}

export function addGlossaryTerm(projectId: string, term: { term: string; say: string; draw: string; avoid: string }): Promise<ScriptLockState> {
  return lockCall(projectId, "/glossary", term);
}

/** Has the AI write the handoff note of every beat that has none, or only of `beat` (a model call). */
export function generateHandoff(projectId: string, beat: string, operation?: AiCall): Promise<ScriptLockState> {
  return lockCall(projectId, "/handoff/generate", { beat }, operation);
}

/** Keeps a note as it is after the dialogue it was written from changed, and approves it. */
export function confirmHandoff(projectId: string, beat: string): Promise<ScriptLockState> {
  return lockCall(projectId, `/handoff/${encodeURIComponent(beat)}/confirm`);
}

/** Freezes the script and its handoff as the next version. */
export function lockScript(projectId: string, note: string): Promise<ScriptLockState> {
  return lockCall(projectId, "/lock", { note });
}
