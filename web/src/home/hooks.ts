import { useQuery } from "@tanstack/react-query";
import { api } from "../api/client";
import type { Banner, Block, Product } from "../api/types";
function useBlock<T>(slug: string) {
  return useQuery({
    queryKey: ["home", slug],
    queryFn: ({ signal }) => api<Block<T>>(`/home/${slug}`, { signal }),
    staleTime: 60000,
    retry: 1,
  });
}
export function useBanners() {
  return useBlock<Banner>("banners");
}
export function useBestSellers() {
  return useBlock<Product>("best-sellers");
}
export function useDeals() {
  return useBlock<Product>("deals");
}
export function useNewReleases() {
  return useBlock<Product>("new-releases");
}
