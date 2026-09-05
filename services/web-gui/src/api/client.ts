import type {
  Plugin,
  Project,
  ProgressMessage,
  PublishMetadata,
  RenderInput,
  SagaStartedResponse,
} from "../types";

const GATEWAY_URL = import.meta.env.VITE_API_BASE_URL;

export const GENERIC_CONNECTION_ERROR = "Không thể kết nối máy chủ, thử lại sau";

export class ApiError extends Error {}

async function parseErrorMessage(response: Response): Promise<string> {
  try {
    const body = await response.json();
    return body?.error?.message ?? body?.error_message ?? GENERIC_CONNECTION_ERROR;
  } catch {
    return GENERIC_CONNECTION_ERROR;
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
    throw new ApiError(await parseErrorMessage(response));
  }
  return response.json() as Promise<T>;
}

export function getPlugins(): Promise<Plugin[]> {
  return apiFetch<Plugin[]>("/v1/plugins");
}

export function startRenderSaga(input: RenderInput): Promise<SagaStartedResponse> {
  return apiFetch<SagaStartedResponse>("/v1/sagas/render", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(input),
  });
}

export function getProject(id: string): Promise<Project> {
  return apiFetch<Project>(`/v1/projects/${id}`);
}

export function retryProject(id: string): Promise<SagaStartedResponse> {
  return apiFetch<SagaStartedResponse>(`/v1/projects/${id}/retry`, { method: "POST" });
}

export function startPublishSaga(
  id: string,
  metadata: PublishMetadata,
): Promise<SagaStartedResponse> {
  return apiFetch<SagaStartedResponse>("/v1/sagas/publish", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ project_id: id, ...metadata }),
  });
}

export function getYoutubeAuthStartUrl(): string {
  return `${GATEWAY_URL}/v1/auth/youtube/start`;
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
