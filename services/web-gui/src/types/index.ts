export interface RenderInput {
  project_id: string;
  script_content: string;
  voice_language: "vi" | "en";
  background_music_path?: string;
  tts_enabled: boolean;
  voice_id?: string;
  /** "off" | "track" | "burn_in" | "both" (ADR-0027). */
  subtitle_mode: string;
  subtitle_style?: SubtitleStylePayload;
  render_quality?: "480p15" | "720p30" | "1080p60" | "4k60";
  /** feature/remotion-engine — "manim" (default) | "remotion". */
  render_engine?: "manim" | "remotion";
  /** Font for text drawn inside a Remotion video; empty means Be Vietnam Pro. */
  video_font?: string;
  video_format_id?: string;
  background_music_volume?: number;
  /** "long" | "short" — empty means "long". */
  video_output_mode?: "long" | "short";
  /** Links this project to another covering the same topic. */
  companion_project_id?: string;
  /**
   * Cổng duyệt dàn ý; bỏ trống thì server hiểu là bật.
   *
   * Bước 4 (Validate) CHÍNH LÀ cổng đó: nó là màn hình Creator dừng
   * lại để xem dàn ý và cảnh báo trước khi tốn TTS/render. Tắt cổng nghĩa là
   * bước 4 không có gì để dừng và saga chạy thẳng sang bước 5, nên nơi nào
   * nộp saga từ wizard đều gửi `true` tường minh thay vì dựa vào mặc định.
   */
  review_enabled?: boolean;
}

export interface SubtitleStylePayload {
  font_family?: string;
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
  narration_text: string;
  /** Khung hình lúc câu này được nói, dạng "Text×2, Arrow". */
  visual?: string;
  duration_seconds?: number;
  audio_path?: string;
  /** Các trường khác của scene vẫn đi qua nguyên trạng. */
  [key: string]: unknown;
}

export interface Project {
  project_id: string;
  status: string;
  /** Bước wizard (1-7) nên mở lại: bước đã xác nhận xa nhất hoặc bước saga đang ở. */
  wizard_step?: number;
  /** Màn wizard mở lần cuối trên draft; "" / vắng mặt nếu chưa ghi. Không còn dùng: vị trí lấy từ flow_step. */
  wizard_route?: string;
  /** Vị trí trong flow 13 bước (server suy ra từ trạng thái + nội dung đã có). */
  flow_step?: number;
  run_state?: "idle" | "running" | "failed" | "done" | "cancelled";
  /** Project mà bản này được tạo từ đó (fork); vắng mặt nếu không phải bản fork. */
  forked_from?: string;
  /**
   * The project's content language. The wire name is historical:
   * it now drives subtitles, metadata and prompts, not just the TTS voice.
   */
  voice_language: "vi" | "en";
  video_path?: string;
  scenes: Scene[];
  youtube_video_url?: string;
  error_message?: string;
  /**
   * Absent when no caption track was requested; otherwise
   * "uploaded" | "skipped_no_scope" | "failed". A skipped/failed caption
   * would otherwise be invisible (the video itself published fine).
   */
  caption_status?: string;
  /** Dữ liệu dựng màn duyệt dàn ý; chỉ có mặt khi cổng duyệt bật. */
  review_enabled?: boolean;
  beats?: BeatOccurrence[];
  validation_warnings?: string[];
  /**
   * Đủ dữ liệu để gọi lại `startRenderSaga` cho ĐÚNG project_id này ở một
   * `render_quality` khác (render lại chất lượng cao hơn khi chốt final). Không optional theo nghĩa "có thể thiếu dữ liệu" —
   * orchestrator luôn trả các trường này — nhưng đánh dấu optional vì test cũ
   * dựng `Project` tối giản không cần khai báo hết.
   */
  script_content?: string;
  background_music_path?: string;
  background_music_volume?: number;
  tts_enabled?: boolean;
  voice_id?: string;
  subtitle_mode?: string;
  subtitle_style?: SubtitleStylePayload;
  render_quality?: "480p15" | "720p30" | "1080p60" | "4k60";
  /** feature/remotion-engine — "manim" (default) | "remotion". */
  render_engine?: "manim" | "remotion";
  /** Font for text drawn inside a Remotion video; empty means Be Vietnam Pro. */
  video_font?: string;
  video_format_id?: string;
  /** "long" | "short" — empty means "long". */
  video_output_mode?: "long" | "short";
  /** Id của project cùng chủ đề (bản dài/bản ngắn kia), nếu có. */
  companion_project_id?: string;
}

/** Một beat lượt dry quan sát được, gắn vào câu lời thoại mở đầu nó. */
export interface BeatOccurrence {
  scene_index: number;
  id: string;
}

export interface ProgressMessage {
  project_id: string;
  step: string;
  status: "in_progress" | "completed" | "failed";
  scene_index?: number;
  scene_total?: number;
  /**
   * Heartbeat from a long render. Carries no percentage on
   * purpose: Manim gives no reliable total animation count, and a fabricated
   * percentage that stalls or jumps backwards is worse than an honest clock.
   */
  elapsed_seconds?: number;
  animation_index?: number;
  /** 0-100, Remotion renders only (renderMedia knows its frame total); Manim sends none. */
  render_percent?: number;
  /** 0-100, assemble_video: ffmpeg's own -progress position over the target length. */
  merge_percent?: number;
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
 * connecting more channels to the same client does not (see ADR-0026).
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
   * False for a channel connected before force-ssl was
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
  /** "manim" | "remotion" — which engine rendered (or will render) this project's video. */
  render_engine: string;
  /** 1-7, bước wizard project đang ở. */
  wizard_step?: number;
  /** Chủ đề (ý tưởng) — dùng làm tên project trong danh sách. */
  topic?: string;
  /** Vị trí trong flow 13 bước và trạng thái chạy. */
  flow_step?: number;
  run_state?: "idle" | "running" | "failed" | "done" | "cancelled";
  forked_from?: string;
  /** Tên dự án nguồn — có thể nằm ở trang khác. */
  forked_from_topic?: string;
}

/** Nhóm lọc của danh sách video (lọc ở server). */
export type ProjectListFilter = "all" | "running" | "waiting" | "problem" | "done";

/** Số dự án trong từng nhóm lọc, tính trên toàn bộ danh sách. */
export type ProjectListCounts = Record<ProjectListFilter, number>;

/** Một trang của GET /v1/projects?page=…. */
export interface ProjectPage {
  projects: ProjectSummary[];
  /** Số dự án sau khi lọc. */
  total: number;
  /** Trang server thật sự trả (trang vượt quá cuối thì là trang cuối). */
  page: number;
  page_size: number;
  counts: ProjectListCounts;
}


/** Một beat trong hình dạng video. */
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
 * không vừa khuôn. Creator nhân bản rồi sửa.
 */
export interface VideoFormat {
  id: string;
  name: string;
  version: number;
  min_seconds: number;
  max_seconds: number;
  beats: FormatBeat[];
}
