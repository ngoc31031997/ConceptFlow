export interface RenderInput {
  project_id: string;
  script_content: string;
  plugin_id: string;
  category_hint: string;
  voice_language: "vi" | "en";
  background_music_path?: string;
}

export interface Scene {
  scene_index: number;
  [key: string]: unknown;
}

export interface Project {
  project_id: string;
  status: string;
  video_path?: string;
  scenes: Scene[];
  plugin_id: string;
  category_hint: string;
  voice_language: string;
  youtube_video_url?: string;
  error_message?: string;
}

export interface ProgressMessage {
  project_id: string;
  step: string;
  status: "in_progress" | "completed" | "failed";
  scene_index?: number;
  scene_total?: number;
  error_message?: string;
}

export interface Plugin {
  plugin_id: string;
  name: string;
  supported_categories: string[];
}

export interface PublishMetadata {
  youtube_title: string;
  description?: string;
  tags?: string[];
  visibility: "public" | "unlisted" | "private";
}

export interface SagaStartedResponse {
  saga_id: string;
  status: string;
}

export interface ProjectSummary {
  project_id: string;
  status: string;
  video_path?: string;
  error_message?: string;
  updated_at: string;
}
