export type Language = "en" | "zh" | "ja";

export const LANGUAGE_STORAGE_KEY = "solo-vpn-language";

export const languageOptions: Array<{ value: Language; label: string }> = [
  { value: "en", label: "English" },
  { value: "zh", label: "中文" },
  { value: "ja", label: "日本語" }
];

export function normalizeLanguage(value: string | null | undefined): Language {
  if (value === "zh" || value === "zh-CN") {
    return "zh";
  }
  if (value === "ja" || value === "ja-JP") {
    return "ja";
  }
  return "en";
}

export function detectInitialLanguage(): Language {
  if (typeof window === "undefined") {
    return "en";
  }

  const stored = window.localStorage.getItem(LANGUAGE_STORAGE_KEY);
  if (stored) {
    return normalizeLanguage(stored);
  }

  return "en";
}

export function dateLocale(language: Language) {
  switch (language) {
    case "zh":
      return "zh-CN";
    case "ja":
      return "ja-JP";
    default:
      return "en-US";
  }
}
