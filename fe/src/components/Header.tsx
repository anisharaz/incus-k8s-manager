import { useEffect, useState } from "react";

interface Status {
  status: {
    incus: string;
  };
}

const getStatusColor = (status: string) => {
  switch (status.toLowerCase()) {
    case "running":
      return "bg-green-500";
    case "stopped":
      return "bg-yellow-500";
    case "not found":
      return "bg-red-500";
    default:
      return "bg-gray-500";
  }
};

export function Header() {
  const [status, setStatus] = useState<Status | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    const fetchStatus = async () => {
      try {
        const response = await fetch("/api/v1/status");
        if (!response.ok) {
          throw new Error("Failed to fetch status");
        }
        const data = await response.json();
        setStatus(data);
        setError(null);
      } catch (err) {
        setError(err instanceof Error ? err.message : "Unknown error");
        setStatus(null);
      } finally {
        setLoading(false);
      }
    };

    fetchStatus();
    const interval = setInterval(fetchStatus, 5000); // Refresh every 5 seconds

    return () => clearInterval(interval);
  }, []);

  return (
    <header className="border-b border-gray-200 bg-white">
      <div className="container mx-auto flex items-center justify-between px-4 py-4">
        <h1 className="text-2xl font-bold text-gray-900">Incus K8s Manager</h1>

        <div className="flex items-center gap-6">
          <div className="flex items-center gap-2">
            <span className="text-sm font-medium text-gray-600">
              Incus Status:
            </span>
            {loading ? (
              <span className="text-sm text-gray-500">Loading...</span>
            ) : error ? (
              <span className="text-sm text-red-600">Error: {error}</span>
            ) : (
              <div className="flex items-center gap-2">
                <div
                  className={`h-3 w-3 rounded-full ${getStatusColor(status?.status.incus || "")}`}
                />
                <span className="text-sm font-medium text-gray-900">
                  {status?.status.incus}
                </span>
              </div>
            )}
          </div>
        </div>
      </div>
    </header>
  );
}
