export interface RenderInput {
  project_id: string;
  script_content: string;
  voice_language: "vi" | "en";
  background_music_path?: string;
  tts_enabled: boolean;
  voice_id?: string;
  subtitles_enabled: boolean;
  subtitle_style?: SubtitleStylePayload;
}

export interface SubtitleStylePayload {
  font_size: "small" | "medium" | "large";
  text_color: string;
  background_opacity: number;
  position: "bottom" | "top";
}

export interface Voice {
  voice_id: string;
  language: "vi" | "en";
  gender: "female" | "male";
  quality: string;
  label: string;
  sample_audio_url: string;
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
  /**
   * Heartbeat from a long render (CR-003 FR11.4). Carries no percentage on
   * purpose: Manim gives no reliable total animation count, and a fabricated
   * percentage that stalls or jumps backwards is worse than an honest clock.
   */
  elapsed_seconds?: number;
  animation_index?: number;
  error_message?: string;
}

export interface PublishMetadata {
  youtube_title: string;
  description?: string;
  tags?: string[];
  visibility: "public" | "unlisted" | "private";
  publish_at?: string;
  thumbnail_path?: string;
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
