import { Button } from "@/components/ui/button";
import { useStatus } from "@/context";
import { TaskDashboard } from "@/components/TaskDashboard";

export function Home() {
  const { status, loading, error, refetch } = useStatus();

  return (
    <div className="mb-8">
      <h2 className="text-2xl font-bold mb-4">Status Dashboard</h2>

      <div className="bg-white rounded-lg shadow p-6 mb-4">
        <div className="flex justify-between items-center">
          <div>
            <h3 className="text-lg font-semibold mb-2">Incus Service</h3>
            {loading ? (
              <p className="text-gray-500">Loading...</p>
            ) : error ? (
              <p className="text-red-500">Error: {error}</p>
            ) : (
              <div>
                <p className="text-gray-700">
                  Status:{" "}
                  <span
                    className={`font-bold ${
                      status?.incus === "running"
                        ? "text-green-600"
                        : status?.incus === "stopped"
                          ? "text-yellow-600"
                          : "text-red-600"
                    }`}
                  >
                    {status?.incus}
                  </span>
                </p>
              </div>
            )}
          </div>
          <Button onClick={refetch} disabled={loading}>
            {loading ? "Refreshing..." : "Refresh"}
          </Button>
        </div>
      </div>

      <div className="mb-8">
        <TaskDashboard />
      </div>
    </div>
  );
}
