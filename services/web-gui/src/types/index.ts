export interface RenderInput {
  project_id: string;
  script_content: string;
  voice_language: "vi" | "en";
  background_music_path?: string;
  tts_enabled: boolean;
  voice_id?: string;
  /** CR-015 — "off" | "track" | "burn_in" | "both" (ADR-0027). */
  subtitle_mode: string;
  subtitle_style?: SubtitleStylePayload;
  render_quality?: "720p30" | "1080p60" | "4k60";
  video_format_id?: string;
  background_music_volume?: number;
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
  /**
   * Which TTS engine produces this voice — "edge", "azure" or "google".
   * Azure and Edge publish the same voice names, so the label alone cannot
   * tell them apart; this is what the GUI badges. Left as a plain string so a
   * catalogue that gains an engine does not fail to parse here.
   */
  engine: string;
  sample_audio_url: string;
}

export interface Scene {
  scene_index: number;
  [key: string]: unknown;
}

export interface Project {
  project_id: string;
  status: string;
  /**
   * The project's content language. The wire name is historical (CR-008 §C2):
   * it now drives subtitles, metadata and prompts, not just the TTS voice.
   */
  voice_language: "vi" | "en";
  video_path?: string;
  scenes: Scene[];
  youtube_video_url?: string;
  error_message?: string;
  /**
   * CR-015 FR39.4 — absent when no caption track was requested; otherwise
   * "uploaded" | "skipped_no_scope" | "failed". A skipped/failed caption
   * would otherwise be invisible (the video itself published fine).
   */
  caption_status?: string;
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
  /** Which connected channel to publish to. Omitted => the default channel. */
  channel_id?: string;
}

/**
 * One configured OAuth client = one GCP project = one quota bucket
 * (~6 uploads/day). Adding a client_secret file raises that ceiling;
 * connecting more channels to the same client does not (CR-012, ADR-0026).
 */
export interface YoutubeApp {
  client_id: string;
  label: string;
  project_id: string;
  source_file: string;
  redirect_ok: boolean;
  /** The exact URI to register in Cloud Console, when redirect_ok is false. */
  redirect_uri_hint: string | null;
}

/** One connected YouTube channel. */
export interface YoutubeAccount {
  channel_id: string;
  channel_title: string;
  client_id: string;
  app_label: string;
  is_default: boolean;
  /**
   * CR-015 FR40.2 — false for a channel connected before force-ssl was
   * requested. Publishing still works; only the caption track is skipped.
   */
  has_caption_scope: boolean;
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


/** Một beat trong hình dạng video (CR-019 FR51). */
export interface FormatBeat {
  id: string;
  role: string;
  min_seconds: number;
  max_seconds: number;
  required: boolean;
  max_repeat: number;
}

/**
 * Hình dạng lặp lại của một video.
 *
 * Là dữ liệu chứ không phải hằng số trong mã nguồn: beat nào một chủ đề cần thì
 * thay đổi rất nhiều, nên một bộ beat cố định sẽ sai ngay ở chủ đề đầu tiên
 * không vừa khuôn. Creator nhân bản rồi sửa (FR51.5).
 */
export interface VideoFormat {
  id: string;
  name: string;
  version: number;
  min_seconds: number;
  max_seconds: number;
  beats: FormatBeat[];
}
