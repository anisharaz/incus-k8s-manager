import { createContext } from "react";
import type { User } from "@/lib/types";

export type AuthStatus =
  | "loading"
  | "needs-bootstrap"
  | "needs-login"
  | "authenticated";

export interface AuthContextType {
  status: AuthStatus;
  user: User | null;
  login: (username: string, password: string) => Promise<void>;
  registerAdmin: (username: string, password: string) => Promise<void>;
  logout: () => Promise<void>;
  refresh: () => Promise<void>;
}

export const AuthContext = createContext<AuthContextType | undefined>(
  undefined,
);
