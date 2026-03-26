import { clsx, type ClassValue } from "clsx";
import { twMerge } from "tailwind-merge";
import { dateLocale, type Language } from "@/lib/i18n";

export function cn(...inputs: ClassValue[]) {
  return twMerge(clsx(inputs));
}

export function formatLastSeen(value: string | null | undefined, language: Language) {
  if (!value) {
    switch (language) {
      case "zh":
        return "暂无流量";
      case "ja":
        return "まだトラフィックなし";
      default:
        return "No traffic yet";
    }
  }

  const dt = new Date(value);
  if (Number.isNaN(dt.getTime())) {
    switch (language) {
      case "zh":
        return "未知";
      case "ja":
        return "不明";
      default:
        return "Unknown";
    }
  }

  return dt.toLocaleString(dateLocale(language));
}
