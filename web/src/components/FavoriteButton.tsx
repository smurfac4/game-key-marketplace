import { useLocation, useNavigate } from "react-router-dom";
import { useAuth } from "../auth/AuthContext";
import { useWishlist, useFavoriteAction } from "../hooks/account";
import { errorText } from "../api/client";
import { copy } from "../content/copy";
import { Icon } from "./Icon";
export function FavoriteButton({
  id,
  name,
  compact = false,
}: {
  id: number;
  name: string;
  compact?: boolean;
}) {
  const { user } = useAuth();
  const location = useLocation();
  const navigate = useNavigate();
  const wishlist = useWishlist();
  const action = useFavoriteAction();
  const selected = wishlist.data?.data.some((p) => p.id === id) ?? false;
  return (
    <div className={compact ? "absolute right-2 top-2 z-10" : "my-5"}>
      <button
        type="button"
        aria-label={`${selected ? copy.favoriteRemove : copy.favoriteAdd}: ${name}`}
        aria-pressed={selected}
        disabled={action.isPending || (!!user && wishlist.isPending)}
        className={`inline-flex min-h-11 items-center justify-center gap-2 rounded-full border border-white/30 bg-[#211046]/95 p-3 transition hover:border-accent hover:text-accent ${selected ? "text-accent" : "text-ink"}`}
        onClick={() => {
          if (!user) {
            navigate(
              `/login?next=${encodeURIComponent(location.pathname + location.search)}`,
            );
            return;
          }
          if (wishlist.isError) {
            void wishlist.refetch();
            return;
          }
          action.mutate({ id, remove: selected });
        }}
      >
        <Icon
          name="heart"
          width="20"
          height="20"
          fill={selected ? "currentColor" : "none"}
        />
        {!compact && (selected ? copy.favorites : copy.favoriteAdd)}
      </button>
      {(action.isError || wishlist.isError) && (
        <p
          role="alert"
          className={
            compact
              ? "absolute right-0 top-12 w-52 rounded border border-line bg-surface p-3 text-xs"
              : "mt-2 text-sm text-accent"
          }
        >
          {errorText(action.error ?? wishlist.error)}
        </p>
      )}
    </div>
  );
}
