import type { FactType } from "../../api/client";

/** The names the table and the forms give each kind of fact. */
export const FACT_TYPE_NAMES: Record<FactType, string> = {
  fact: "Sự thật",
  hypothesis: "Giả thuyết",
  unknown: "Chưa biết",
  misconception: "Ngộ nhận phổ biến",
};
