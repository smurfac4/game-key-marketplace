import { useSyncExternalStore } from "react";
export type Preferences = {
  language: "en" | "ru" | "lt";
  currency: "EUR" | "USD" | "GBP";
};
const listeners = new Set<() => void>();
let current: Preferences = { language: "en", currency: "EUR" };
try {
  const saved = JSON.parse(
    localStorage.getItem("keyspace-preferences") ?? "null",
  );
  if (saved && ["EUR", "USD", "GBP"].includes(saved.currency))
    current = {
      language:
        saved.language === "ru" || saved.language === "lt"
          ? saved.language
          : "en",
      currency: saved.currency,
    };
  localStorage.setItem("keyspace-preferences", JSON.stringify(current));
} catch {}
const urlCurrency = new URLSearchParams(location.search).get("currency");
if (urlCurrency && ["EUR", "USD", "GBP"].includes(urlCurrency))
  current = { ...current, currency: urlCurrency as Preferences["currency"] };
export function preferences() {
  return current;
}
export function setPreferences(next: Preferences) {
  current = next;
  try {
    localStorage.setItem("keyspace-preferences", JSON.stringify(next));
  } catch {}
  document.documentElement.lang = next.language;
  listeners.forEach((fn) => fn());
}
export function usePreferences() {
  return useSyncExternalStore((fn) => {
    listeners.add(fn);
    return () => {
      listeners.delete(fn);
    };
  }, preferences);
}
document.documentElement.lang = current.language;

export function locale() {
  return { en: "en-GB", ru: "ru-RU", lt: "lt-LT" }[current.language];
}
