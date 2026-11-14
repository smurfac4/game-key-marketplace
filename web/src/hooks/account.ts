import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { api, json } from "../api/client";
import type { AccountSummary, Profile, Favorite } from "../api/types";
import { useAuth } from "../auth/AuthContext";
import { preferences } from "../content/preferences";
export function useAccount() {
  const { user } = useAuth();
  return useQuery({
    queryKey: ["account", user?.id, preferences().currency],
    queryFn: ({ signal }) =>
      api<{ data: AccountSummary }>("/account", { signal }),
    enabled: !!user,
    staleTime: 15000,
  });
}
export function useWishlist() {
  const { user } = useAuth();
  return useQuery({
    queryKey: ["wishlist", user?.id, preferences().currency],
    queryFn: ({ signal }) => api<{ data: Favorite[] }>("/wishlist", { signal }),
    enabled: !!user,
    staleTime: 15000,
  });
}
export function useProfileAction() {
  const client = useQueryClient();
  return useMutation({
    mutationFn: (profile: Profile) =>
      api<{ data: AccountSummary }>("/account", {
        method: "PATCH",
        body: json(profile),
      }),
    onSuccess: () => client.invalidateQueries({ queryKey: ["account"] }),
  });
}
export function useFavoriteAction() {
  const client = useQueryClient();
  return useMutation({
    mutationFn: ({ id, remove }: { id: number; remove: boolean }) =>
      api(`/wishlist/${id}`, { method: remove ? "DELETE" : "PUT" }),
    onSuccess: async () => {
      await Promise.all([
        client.invalidateQueries({ queryKey: ["wishlist"] }),
        client.invalidateQueries({ queryKey: ["account"] }),
      ]);
    },
  });
}
