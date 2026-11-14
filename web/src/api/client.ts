import { preferences } from "../content/preferences";
import { lithuanianErrors } from "../content/lt";
import { copy } from "../content/copy";
export class ApiError extends Error {
  constructor(
    public status: number,
    public code: string,
    message: string,
    public field?: string,
  ) {
    super(message);
  }
}
export async function api<T>(
  path: string,
  options: RequestInit = {},
): Promise<T> {
  const method = options.method ?? "GET";
  const headers = new Headers(options.headers);
  headers.set("Accept-Language", preferences().language);
  if (options.body) headers.set("Content-Type", "application/json");
  if (!["GET", "HEAD", "OPTIONS"].includes(method))
    headers.set("X-CSRF-Protection", "1");
  if (
    (method === "GET" ||
      path.startsWith("/cart") ||
      path.startsWith("/account") ||
      path.startsWith("/wishlist")) &&
    (path.startsWith("/products") ||
      path.startsWith("/home/") ||
      path.startsWith("/search/suggest") ||
      path.startsWith("/cart") ||
      path.startsWith("/account") ||
      path.startsWith("/wishlist"))
  )
    headers.set("X-Display-Currency", preferences().currency);
  const response = await fetch(`/api${path}`, {
    ...options,
    method,
    headers,
    credentials: "include",
  });
  if (!response.ok) {
    const body = await response.json().catch(() => null);
    throw new ApiError(
      response.status,
      body?.error?.code ?? "request_failed",
      body?.error?.message ?? copy.error,
      body?.error?.field,
    );
  }
  if (response.status === 204) return undefined as T;
  return response.json() as Promise<T>;
}
export const json = (value: unknown) => JSON.stringify(value);
export function errorText(error: unknown) {
  if (preferences().language === "lt" && error instanceof ApiError)
    return lithuanianErrors[error.code] ?? copy.error;
  if (preferences().language === "ru" && error instanceof ApiError) {
    const messages: Record<string, string> = {
      unauthorized: "Войдите в аккаунт или проверьте данные входа.",
      email_taken: "Этот адрес уже зарегистрирован.",
      rate_limited: "Слишком много запросов. Подождите и повторите.",
      out_of_stock: "Ключи закончились. Обновите корзину.",
      payment_declined: "Тестовая оплата отклонена. Корзина сохранена.",
      invalid_card: "Используйте поддерживаемую тестовую карту.",
      idempotency_conflict:
        "Запрос оплаты изменился. Обновите страницу оформления заказа.",
      timeout: "Сервер не ответил вовремя. Повторите запрос.",
      empty_cart: "Корзина пуста.",
      currency_mismatch:
        "Для одной корзины выбирайте предложения в одной исходной валюте.",
    };
    return messages[error.code] ?? copy.error;
  }
  return error instanceof ApiError ? error.message : copy.error;
}
export function safeNext(value: string | null) {
  return value?.startsWith("/") &&
    !value.startsWith("//") &&
    !value.includes("\\") &&
    !/^\/(login|register)(\?|$)/.test(value)
    ? value
    : "/";
}
