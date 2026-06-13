import { createContext } from "react";

export interface StatusContextType {
  status: {
    incus: string;
  } | null;
  loading: boolean;
  error: string | null;
  refetch: () => void;
}

export const StatusContext = createContext<StatusContextType | undefined>(
  undefined,
);
