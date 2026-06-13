import { type ReactNode, useEffect, useState } from "react";
import { StatusContext } from "./status.context";

export function StatusProvider({ children }: { children: ReactNode }) {
  const [status, setStatus] = useState<{ incus: string } | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const fetchStatus = async () => {
    try {
      setLoading(true);
      setError(null);
      const response = await fetch("http://localhost:8000/api/v1/status");

      if (!response.ok) {
        throw new Error(`HTTP error! status: ${response.status}`);
      }

      const data = await response.json();
      setStatus(data.status);
    } catch (err) {
      const message =
        err instanceof Error ? err.message : "Failed to fetch status";
      setError(message);
      console.error("Status fetch error:", err);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    // Initial fetch
    let isMounted = true;

    const initFetch = async () => {
      if (isMounted) {
        await fetchStatus();
      }
    };

    initFetch();

    // Poll for updates every 30 seconds
    const interval = setInterval(() => {
      if (isMounted) {
        fetchStatus();
      }
    }, 30000);

    return () => {
      isMounted = false;
      clearInterval(interval);
    };
  }, []);

  return (
    <StatusContext.Provider
      value={{ status, loading, error, refetch: fetchStatus }}
    >
      {children}
    </StatusContext.Provider>
  );
}
