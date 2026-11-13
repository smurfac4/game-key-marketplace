import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { api, json } from "../api/client";
import type { Cart } from "../api/types";
import { useAuth } from "../auth/AuthContext";
export function useCart() {
  const { user } = useAuth();
  return useQuery({
    queryKey: ["cart", user?.id],
    queryFn: ({ signal }) => api<{ data: Cart }>("/cart", { signal }),
    enabled: !!user,
    retry: false,
  });
}
export function useCartAction() {
  const client = useQueryClient();
  const { user } = useAuth();
  return useMutation({
    mutationFn: ({
      method,
      offer,
      quantity,
    }: {
      method: "POST" | "PATCH" | "DELETE";
      offer: number;
      quantity?: number;
    }) =>
      api<{ data: Cart }>(
        method === "POST" ? "/cart/items" : `/cart/items/${offer}`,
        {
          method,
          body:
            method === "DELETE"
              ? undefined
              : json(
                  method === "POST"
                    ? { offer_id: offer, quantity }
                    : { quantity },
                ),
        },
      ),
    onSuccess: (result) => {
      client.setQueryData(["cart", user?.id], result);
    },
  });
}
