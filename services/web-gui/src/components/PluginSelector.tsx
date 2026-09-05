import { useEffect, useState } from "react";
import { getPlugins } from "../api/client";
import type { Plugin } from "../types";

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
    <select
      data-testid="new-project-plugin-select"
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
  );
}
