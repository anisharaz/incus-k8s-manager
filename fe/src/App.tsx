import { Button } from "./components/ui/button";
import { Header } from "./components/Header";
import { useStatus } from "./context";

function Home() {
  const { status, loading, error, refetch } = useStatus();

  return (
    <div className="min-h-screen bg-gray-50">
      <Header />
      <main className="container mx-auto px-4 py-8">
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
        </div>

        <div className="mb-8">
          <Button>Click me</Button>
        </div>
      </main>
    </div>
  );
}

export default Home;
