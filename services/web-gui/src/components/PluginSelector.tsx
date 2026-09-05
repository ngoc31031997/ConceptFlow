import { useEffect, useState } from "react";
import { getPlugins } from "../api/client";
import type { Plugin } from "../types";
import glass from "../styles/glass.module.css";
import styles from "./PluginSelector.module.css";

interface PluginSelectorProps {
  value: string | null;
  onChange: (pluginId: string) => void;
}

export function PluginSelector({ value, onChange }: PluginSelectorProps) {
  const [plugins, setPlugins] = useState<Plugin[]>([]);
  const [isLoading, setIsLoading] = useState(true);

  useEffect(() => {
    getPlugins()
      .then(setPlugins)
      .finally(() => setIsLoading(false));
  }, []);

  return (
    <div className={glass.card} style={{ padding: 22 }}>
      <div className={glass.cardTitle} style={{ marginBottom: 14 }}>
        Loại nội dung
      </div>
      <div className={styles.wrap}>
        <select
          data-testid="new-project-plugin-select"
          className={`${glass.select} ${styles.select}`}
          value={value ?? ""}
          onChange={(event) => onChange(event.target.value)}
          disabled={isLoading}
        >
          <option value="" disabled>
            {isLoading ? "Đang tải..." : "Chọn loại nội dung"}
          </option>
          {plugins.map((plugin) => (
            <option key={plugin.plugin_id} value={plugin.plugin_id}>
              {plugin.name}
            </option>
          ))}
        </select>
        <span className={styles.chevron}>
          <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
            <path d="M6 9l6 6 6-6" />
          </svg>
        </span>
      </div>
      <div className={glass.cardHint}>Xác định cách phân loại scene và phong cách hoạt hình</div>
    </div>
  );
}
