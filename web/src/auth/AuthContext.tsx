import { createContext, useContext, type ReactNode } from "react";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { Navigate, useLocation } from "react-router-dom";
import { api, ApiError, json } from "../api/client";
import type { User } from "../api/types";
import { Skeleton, ErrorState } from "../components/ui";
type AuthValue = {
  user: User | null | undefined;
  pending: boolean;
  error: unknown;
  signIn: (email: string, password: string) => Promise<void>;
  signOut: () => Promise<void>;
  acceptSession: (user: User) => void;
};
const AuthContext = createContext<AuthValue | null>(null);
export function AuthProvider({ children }: { children: ReactNode }) {
  const client = useQueryClient();
  const query = useQuery({
    queryKey: ["me"],
    queryFn: async ({ signal }) => {
      try {
        return (
          await api<{
            data: User;
          }>("/me", { signal })
        ).data;
      } catch (e) {
        if (e instanceof ApiError && e.status === 401) return null;
        throw e;
      }
    },
    retry: false,
    staleTime: 30000,
  });
  async function signIn(email: string, password: string) {
    const result = await api<{
      data: User;
    }>("/auth/login", {
      method: "POST",
      body: json({ email, password }),
    });
    acceptSession(result.data);
  }
  function acceptSession(user: User) {
    client.removeQueries({ predicate: (q) => q.queryKey[0] !== "me" });
    client.setQueryData(["me"], user);
  }
  async function signOut() {
    await api("/auth/logout", { method: "POST" });
    client.removeQueries({ predicate: (q) => q.queryKey[0] !== "me" });
    client.setQueryData(["me"], null);
  }
  return (
    <AuthContext.Provider
      value={{
        user: query.data,
        pending: query.isPending,
        error: query.error,
        signIn,
        signOut,
        acceptSession,
      }}
    >
      {children}
    </AuthContext.Provider>
  );
}
export function useAuth() {
  const auth = useContext(AuthContext);
  if (!auth) throw new Error("AuthProvider missing");
  return auth;
}
export function Protected({ children }: { children: ReactNode }) {
  const auth = useAuth();
  const location = useLocation();
  const client = useQueryClient();
  if (auth.pending)
    return (
      <div className="container-page">
        <Skeleton className="h-72" />
      </div>
    );
  if (auth.error)
    return (
      <div className="container-page">
        <ErrorState
          error={auth.error}
          retry={() => void client.invalidateQueries({ queryKey: ["me"] })}
        />
      </div>
    );
  if (!auth.user)
    return (
      <Navigate
        to={`/login?next=${encodeURIComponent(location.pathname + location.search)}`}
        replace
      />
    );
  return children;
}
