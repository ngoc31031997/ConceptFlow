import { useEffect, useState } from "react";
import { getPlugins } from "../api/client";
import type { Plugin } from "../types";
import glass from "../styles/glass.module.css";
import styles from "./VoiceLanguageSelector.module.css";

interface CategorySelectorProps {
  pluginId: string | null;
  value: string | null;
  onChange: (category: string) => void;
}

const CATEGORY_LABELS: Record<string, string> = {
  algorithm: "Thuật toán",
  concept: "Khái niệm",
};

export function CategorySelector({ pluginId, value, onChange }: CategorySelectorProps) {
  const [plugins, setPlugins] = useState<Plugin[]>([]);

  useEffect(() => {
    getPlugins().then(setPlugins);
  }, []);

  const categories = plugins.find((p) => p.plugin_id === pluginId)?.supported_categories ?? [];

  if (!pluginId || categories.length === 0) return null;

  return (
    <div className={glass.card} style={{ padding: 22 }}>
      <div className={glass.cardTitle} style={{ marginBottom: 14 }}>
        Danh mục nội dung
      </div>
      <div className={styles.switch} data-testid="new-project-category-select">
        {categories.map((category) => (
          <button
            key={category}
            type="button"
            className={`${styles.option} ${value === category ? styles.active : ""}`}
            aria-pressed={value === category}
            onClick={() => onChange(category)}
          >
            {CATEGORY_LABELS[category] ?? category}
          </button>
        ))}
      </div>
      <div className={glass.cardHint}>Xác định cách phân loại nội dung — do Creator chọn cho toàn bộ project</div>
    </div>
  );
}
