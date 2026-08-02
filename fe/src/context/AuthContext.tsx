import { type ReactNode, useCallback, useEffect, useState } from "react";
import { AuthContext, type AuthStatus } from "./auth.context";
import type { AuthStatusResponse, User } from "@/lib/types";
import { api, ApiError, setUnauthorizedHandler } from "@/lib/api";

export function AuthProvider({ children }: { children: ReactNode }) {
  const [status, setStatus] = useState<AuthStatus>("loading");
  const [user, setUser] = useState<User | null>(null);

  const refresh = useCallback(async () => {
    setStatus("loading");
    try {
      const bootstrap = await api.get<AuthStatusResponse>(
        "/api/v1/auth/status",
      );
      if (!bootstrap.adminCreated) {
        setUser(null);
        setStatus("needs-bootstrap");
        return;
      }

      const me = await api.get<{ user: User }>("/api/v1/auth/me");
      setUser(me.user);
      setStatus("authenticated");
    } catch {
      // A 401 from /auth/me already triggers the global unauthorized
      // handler below, which drops us to "needs-login". Catching here too
      // covers non-401 failures (e.g. the backend being unreachable) as a
      // safe fallback, rather than leaving the UI on a spinner forever.
      setUser(null);
      setStatus("needs-login");
    }
  }, []);

  useEffect(() => {
    setUnauthorizedHandler(() => {
      setUser(null);
      setStatus("needs-login");
    });

    let isMounted = true;
    const boot = async () => {
      if (isMounted) await refresh();
    };
    boot();

    return () => {
      isMounted = false;
      setUnauthorizedHandler(null);
    };
  }, [refresh]);

  const login = useCallback(async (username: string, password: string) => {
    const data = await api.post<{ user: User }>("/api/v1/auth/login", {
      username,
      password,
    });
    setUser(data.user);
    setStatus("authenticated");
  }, []);

  const registerAdmin = useCallback(
    async (username: string, password: string) => {
      try {
        const data = await api.post<{ user: User }>(
          "/api/v1/auth/register-admin",
          { username, password },
        );
        setUser(data.user);
        setStatus("authenticated");
      } catch (err) {
        // Race: another tab/request already bootstrapped the admin.
        // Swap to the login form instead of leaving this one stuck.
        if (err instanceof ApiError && err.code === 409) {
          setStatus("needs-login");
        }
        throw err;
      }
    },
    [],
  );

  const logout = useCallback(async () => {
    try {
      await api.post("/api/v1/auth/logout");
    } finally {
      setUser(null);
      setStatus("needs-login");
    }
  }, []);

  return (
    <AuthContext.Provider
      value={{ status, user, login, registerAdmin, logout, refresh }}
    >
      {children}
    </AuthContext.Provider>
  );
}
