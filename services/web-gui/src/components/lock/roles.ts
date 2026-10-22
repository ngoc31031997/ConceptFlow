/** The AI roles after the script that read its handoff, by the key the server gives them. */
export const ROLE_LABELS: Record<string, string> = {
  art_world: "Mỹ thuật",
  storyboard: "Storyboard",
  art_color: "Mỹ thuật lượt 2",
  voice: "Giọng",
  illustration: "Hình minh hoạ",
  code: "Code",
  validate: "Validate",
};

/** What each later step reads from a locked version (FR-32). */
export const STEPS_READ: { step: string; reads: string }[] = [
  { step: "Mỹ thuật lượt 1", reads: "giọng điệu và ngoài phạm vi của brief, thuật ngữ, phải giữ chính xác" },
  { step: "Storyboard", reads: "lời thoại theo beat, mục tiêu và chuyển tiếp của beat, bắt buộc truyền đạt, được phép sáng tạo, điểm chưa chắc" },
  { step: "Mỹ thuật lượt 2", reads: "chức năng của từng beat để chia color key" },
  { step: "Giọng", reads: "lời thoại, thuật ngữ và cách đọc" },
  { step: "Hình minh hoạ", reads: "phải giữ chính xác, thuật ngữ, điểm chưa chắc" },
  { step: "Code", reads: "được phép sáng tạo" },
  { step: "Validate", reads: "phải giữ chính xác: kiểm hình đã dựng theo danh sách này" },
];
