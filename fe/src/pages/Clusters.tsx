import { useEffect, useState } from "react";
import { Link } from "react-router";
import { ArrowRight, ServerCog, AlertCircle } from "lucide-react";
import { CreateClusterDialog } from "@/components/CreateClusterDialog";
import { Badge } from "@/components/ui/badge";
import { Skeleton } from "@/components/ui/skeleton";
import { Alert, AlertDescription } from "@/components/ui/alert";
import { api } from "@/lib/api";
import type { Cluster, ClusterStatus } from "@/lib/types";

const statusVariant: Record<
  ClusterStatus,
  "default" | "secondary" | "destructive" | "outline"
> = {
  creating: "secondary",
  ready: "default",
  failed: "destructive",
  deleting: "outline",
};

export function Clusters() {
  const [clusters, setClusters] = useState<Cluster[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const fetchClusters = async () => {
    try {
      setLoading(true);
      setError(null);
      const data = await api.get<{ clusters: Cluster[] }>(
        "/api/v1/clusters",
      );
      setClusters(data.clusters ?? []);
    } catch (err) {
      setError(
        err instanceof Error ? err.message : "Failed to fetch clusters",
      );
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    let isMounted = true;
    const load = async () => {
      if (isMounted) await fetchClusters();
    };
    load();
    // Provisioning takes minutes, so keep polling while this page is open.
    const interval = setInterval(load, 5000);
    return () => {
      isMounted = false;
      clearInterval(interval);
    };
  }, []);

  return (
    <div className="space-y-6">
      <section className="rounded-3xl border bg-card p-6 shadow-sm">
        <div className="flex flex-col gap-4 md:flex-row md:items-end md:justify-between">
          <div>
            <div className="mb-3 inline-flex items-center gap-2 rounded-full bg-muted px-3 py-1 text-xs font-medium uppercase tracking-[0.2em] text-muted-foreground">
              <ServerCog className="h-3.5 w-3.5" />
              Clusters
            </div>
            <h2 className="text-3xl font-semibold tracking-tight text-foreground">
              Kubernetes clusters
            </h2>
            <p className="mt-2 max-w-2xl text-sm text-muted-foreground">
              Browse the clusters created in your environment. Select one to
              inspect the cluster-specific route and details.
            </p>
          </div>
          <div className="flex items-center gap-4">
            <div className="text-sm text-muted-foreground">
              {clusters.length} clusters registered
            </div>
            <CreateClusterDialog onSuccess={fetchClusters} />
          </div>
        </div>
      </section>

      {error && (
        <Alert variant="destructive">
          <AlertCircle className="h-4 w-4" />
          <AlertDescription>{error}</AlertDescription>
        </Alert>
      )}

      {loading && (
        <div className="grid gap-4 lg:grid-cols-2">
          {[1, 2, 3, 4].map((i) => (
            <div
              key={i}
              className="rounded-3xl border bg-card p-5 shadow-sm space-y-4"
            >
              <Skeleton className="h-6 w-3/4" />
              <Skeleton className="h-4 w-full" />
            </div>
          ))}
        </div>
      )}

      {!loading && clusters.length === 0 && (
        <div className="rounded-3xl border bg-card p-12 text-center">
          <p className="text-sm text-muted-foreground">
            No clusters created yet.
          </p>
          <p className="mt-2 text-sm text-muted-foreground">
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
              className="group rounded-3xl border bg-card p-5 shadow-sm transition-transform duration-200 hover:-translate-y-0.5 hover:shadow-md"
            >
              <div className="flex items-start justify-between gap-4">
                <div>
                  <div className="flex items-center gap-2">
                    <h3 className="text-lg font-semibold text-foreground">
                      {cluster.name}
                    </h3>
                    <Badge
                      variant={statusVariant[cluster.status]}
                      className="capitalize"
                    >
                      {cluster.status}
                    </Badge>
                  </div>
                  <p className="mt-2 text-sm text-muted-foreground">
                    {cluster.id}
                  </p>
                  {cluster.message && (
                    <p className="mt-1 text-xs text-muted-foreground">
                      {cluster.message}
                    </p>
                  )}
                </div>
                <ArrowRight className="mt-1 h-5 w-5 text-muted-foreground transition-transform group-hover:translate-x-1" />
              </div>

              <div className="mt-5 grid grid-cols-2 gap-4 text-sm text-muted-foreground">
                <div>
                  <p className="text-xs uppercase tracking-wide text-muted-foreground/70">
                    Status
                  </p>
                  <p className="mt-1 font-medium text-foreground capitalize">
                    {cluster.status}
                  </p>
                </div>
                <div>
                  <p className="text-xs uppercase tracking-wide text-muted-foreground/70">
                    Created
                  </p>
                  <p className="mt-1 font-medium text-foreground">
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
