// 模型目录模块：集中维护可选模型与选择规则，避免在入口脚本中重复定义。
export const MAX_SELECTED_MODELS = 4;

export const MODEL_CATALOG = [
  { key: "deepseek", label: "DeepSeek", icon: "assets/DeepSeek.svg" },
  { key: "doubao", label: "豆包", icon: "assets/Doubao.png" },
  { key: "kimi", label: "Kimi", icon: "assets/Kimi.png" },
  { key: "qwen", label: "千问", icon: "assets/Qwen.png" },
  { key: "glm", label: "智谱GLM", icon: "assets/GLM.png" },
];

export const DEFAULT_MODEL_KEY = MODEL_CATALOG[0]?.key || "deepseek";

export const MODEL_LABELS = MODEL_CATALOG.reduce((labels, item) => {
  labels[item.key] = item.label;
  return labels;
}, {});

export const MODEL_ICONS = MODEL_CATALOG.reduce((icons, item) => {
  icons[item.key] = item.icon || "";
  return icons;
}, {});

export const getModelLabel = (modelKey = "") =>
  MODEL_LABELS[modelKey] || modelKey || "未知模型";

export const getModelIcon = (modelKey = "") => MODEL_ICONS[modelKey] || "";

export const normalizeModelSelection = (inputModels = []) => {
  const seen = new Set();
  const normalized = [];
  (inputModels || []).forEach((item) => {
    const key = String(item || "").trim().toLowerCase();
    if (!MODEL_LABELS[key] || seen.has(key)) return;
    seen.add(key);
    normalized.push(key);
  });
  if (normalized.length === 0) return [DEFAULT_MODEL_KEY];
  return normalized.slice(0, MAX_SELECTED_MODELS);
};
