import {
  createContext,
  useContext,
  useCallback,
  type ReactNode,
} from "react";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { api, ApiError } from "@/lib/api";
import type {
  User,
  SetupCheckResponse,
  AuthResponse,
} from "@/lib/types";

interface AuthContextValue {
  user: User | null;
  needsSetup: boolean;
  isLoading: boolean;
  login: (username: string, password: string) => Promise<void>;
  setup: (token: string, username: string, password: string) => Promise<void>;
  logout: () => Promise<void>;
}

const AuthContext = createContext<AuthContextValue | null>(null);

export function AuthProvider({ children }: { children: ReactNode }) {
  const queryClient = useQueryClient();

  const setupQuery = useQuery({
    queryKey: ["auth-setup"],
    queryFn: () => api.get<SetupCheckResponse>("/auth/setup"),
    staleTime: Infinity,
    retry: false,
  });

  const meQuery = useQuery({
    queryKey: ["auth-user"],
    queryFn: () => api.get<AuthResponse>("/auth/me"),
    enabled: setupQuery.isSuccess && !setupQuery.data.needs_setup,
    retry: false,
  });

  const login = useCallback(
    async (username: string, password: string) => {
      await api.post<AuthResponse>("/auth/login", { username, password });
      await queryClient.invalidateQueries({ queryKey: ["auth-user"] });
      await queryClient.invalidateQueries({ queryKey: ["auth-setup"] });
    },
    [queryClient],
  );

  const setup = useCallback(
    async (token: string, username: string, password: string) => {
      await api.post<AuthResponse>("/auth/setup", {
        token,
        username,
        password,
      });
      await queryClient.invalidateQueries({ queryKey: ["auth-setup"] });
      await queryClient.invalidateQueries({ queryKey: ["auth-user"] });
    },
    [queryClient],
  );

  const logout = useCallback(async () => {
    await api.post("/auth/logout");
    queryClient.setQueryData(["auth-user"], null);
    queryClient.removeQueries({ queryKey: ["auth-user"] });
  }, [queryClient]);

  const needsSetup = setupQuery.data?.needs_setup ?? false;

  // User is from /auth/me response (which wraps in {user: ...})
  const user = meQuery.data?.user ?? null;

  // Loading if setup check is pending, or if user query is pending when enabled
  const isLoading =
    setupQuery.isLoading ||
    (setupQuery.isSuccess && !needsSetup && meQuery.isLoading);

  return (
    <AuthContext.Provider
      value={{ user, needsSetup, isLoading, login, setup, logout }}
    >
      {children}
    </AuthContext.Provider>
  );
}

export function useAuth(): AuthContextValue {
  const ctx = useContext(AuthContext);
  if (!ctx) {
    throw new Error("useAuth must be used within AuthProvider");
  }
  return ctx;
}

export { ApiError };
