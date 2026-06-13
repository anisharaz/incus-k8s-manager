import { useEffect, useState } from "react";
import { Link } from "react-router";
import { ArrowRight, CircleDot, ServerCog, AlertCircle } from "lucide-react";
import { CreateClusterDialog } from "@/components/CreateClusterDialog";

interface Cluster {
  id: string;
  name: string;
  status: string;
  message?: string;
  jobId?: string;
  ip?: string;
  createdAt: string;
  updatedAt: string;
}

const statusStyles: Record<string, string> = {
  creating: "bg-blue-100 text-blue-700 ring-blue-200",
  ready: "bg-emerald-100 text-emerald-700 ring-emerald-200",
  failed: "bg-rose-100 text-rose-700 ring-rose-200",
  deleting: "bg-amber-100 text-amber-700 ring-amber-200",
};

export function Clusters() {
  const [clusters, setClusters] = useState<Cluster[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const fetchClusters = async () => {
    try {
      setLoading(true);
      setError(null);
      const response = await fetch("/api/v1/clusters");
      if (!response.ok) {
        throw new Error("Failed to fetch clusters");
      }
      const data = await response.json();
      setClusters(data.clusters || []);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to fetch clusters");
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchClusters();
    // Refresh clusters every 5 seconds
    const interval = setInterval(fetchClusters, 5000);
    return () => clearInterval(interval);
  }, []);

  return (
    <div className="space-y-6">
      <section className="rounded-3xl border border-slate-200 bg-white p-6 shadow-sm">
        <div className="flex flex-col gap-4 md:flex-row md:items-end md:justify-between">
          <div>
            <div className="mb-3 inline-flex items-center gap-2 rounded-full bg-slate-100 px-3 py-1 text-xs font-medium uppercase tracking-[0.2em] text-slate-600">
              <ServerCog className="h-3.5 w-3.5" />
              Clusters
            </div>
            <h2 className="text-3xl font-semibold tracking-tight text-slate-900">
              Kubernetes clusters
            </h2>
            <p className="mt-2 max-w-2xl text-sm text-slate-600">
              Browse the clusters created in your environment. Select one to
              inspect the cluster-specific route and details.
            </p>
          </div>
          <div className="flex items-center gap-4">
            <div className="text-sm text-slate-500">
              {clusters.length} clusters registered
            </div>
            <CreateClusterDialog onSuccess={fetchClusters} />
          </div>
        </div>
      </section>

      {error && (
        <div className="rounded-3xl border border-rose-200 bg-rose-50 p-4">
          <div className="flex items-center gap-2">
            <AlertCircle className="h-5 w-5 text-rose-600" />
            <p className="text-sm text-rose-700">{error}</p>
          </div>
        </div>
      )}

      {loading && (
        <div className="grid gap-4 lg:grid-cols-2">
          {[1, 2, 3, 4].map((i) => (
            <div
              key={i}
              className="rounded-3xl border border-slate-200 bg-white p-5 shadow-sm animate-pulse"
            >
              <div className="space-y-4">
                <div className="h-6 bg-slate-200 rounded w-3/4"></div>
                <div className="h-4 bg-slate-200 rounded w-full"></div>
              </div>
            </div>
          ))}
        </div>
      )}

      {!loading && clusters.length === 0 && (
        <div className="rounded-3xl border border-slate-200 bg-white p-12 text-center">
          <p className="text-sm text-slate-600">No clusters created yet.</p>
          <p className="mt-2 text-sm text-slate-600">
            Click the "Create Cluster" button to get started.
          </p>
        </div>
      )}

      {!loading && clusters.length > 0 && (
        <section className="grid gap-4 lg:grid-cols-2">
          {clusters.map((cluster) => (
            <Link
              key={cluster.id}
              to={`/clusters/${cluster.id}`}
              className="group rounded-3xl border border-slate-200 bg-white p-5 shadow-sm transition-transform duration-200 hover:-translate-y-0.5 hover:border-slate-300 hover:shadow-md"
            >
              <div className="flex items-start justify-between gap-4">
                <div>
                  <div className="flex items-center gap-2">
                    <h3 className="text-lg font-semibold text-slate-900">
                      {cluster.name}
                    </h3>
                    <span
                      className={`inline-flex items-center gap-1 rounded-full px-2.5 py-1 text-xs font-medium ring-1 ${statusStyles[cluster.status]}`}
                    >
                      <CircleDot className="h-3 w-3 fill-current" />
                      {cluster.status}
                    </span>
                  </div>
                  <p className="mt-2 text-sm text-slate-600">{cluster.id}</p>
                  {cluster.message && (
                    <p className="mt-1 text-xs text-slate-500">
                      {cluster.message}
                    </p>
                  )}
                </div>
                <ArrowRight className="mt-1 h-5 w-5 text-slate-400 transition-transform group-hover:translate-x-1" />
              </div>

              <div className="mt-5 grid grid-cols-2 gap-4 text-sm text-slate-600">
                <div>
                  <p className="text-xs uppercase tracking-wide text-slate-400">
                    Status
                  </p>
                  <p className="mt-1 font-medium text-slate-900 capitalize">
                    {cluster.status}
                  </p>
                </div>
                <div>
                  <p className="text-xs uppercase tracking-wide text-slate-400">
                    Created
                  </p>
                  <p className="mt-1 font-medium text-slate-900">
                    {new Date(cluster.createdAt).toLocaleDateString()}
                  </p>
                </div>
              </div>
            </Link>
          ))}
        </section>
      )}
    </div>
  );
}
