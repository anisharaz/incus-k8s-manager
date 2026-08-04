import { useEffect, useState } from "react";
import { Link, Navigate, useNavigate, useParams } from "react-router";
import { AlertCircle, ArrowLeft } from "lucide-react";

import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { Alert, AlertDescription } from "@/components/ui/alert";
import { Skeleton } from "@/components/ui/skeleton";
import { TerminalPane } from "@/components/TerminalPane";
import { FullPageSpinner } from "@/components/FullPageSpinner";
import { api } from "@/lib/api";
import { cn } from "@/lib/utils";
import { useAuth } from "@/context";
import type { Cluster, ClusterNode } from "@/lib/types";

// Dedicated full-page terminal view — deliberately outside ProtectedLayout
// (no sidebar/header) so the terminal itself gets as much screen as
// possible. Not linked from the sidebar; only reachable via a node's
// terminal button, which navigates here directly.
export function TerminalPage() {
  const { clusterId, nodeId } = useParams();
  const navigate = useNavigate();
  const { status } = useAuth();

  const [clusters, setClusters] = useState<Cluster[]>([]);
  const [nodes, setNodes] = useState<ClusterNode[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    let isMounted = true;
    api
      .get<{ clusters: Cluster[] }>("/api/v1/clusters")
      .then((data) => {
        if (isMounted) setClusters(data.clusters ?? []);
      })
      .catch(() => {
        // the per-cluster node fetch below surfaces the real error
      });
    return () => {
      isMounted = false;
    };
  }, []);

  useEffect(() => {
    if (!clusterId) return;
    let isMounted = true;

    const load = async () => {
      setLoading(true);
      setError(null);
      try {
        const data = await api.get<{ nodes: ClusterNode[] }>(
          `/api/v1/clusters/${clusterId}/nodes`,
        );
        if (!isMounted) return;
        const fetchedNodes = data.nodes ?? [];
        setNodes(fetchedNodes);

        const current = fetchedNodes.find((n) => n.id === nodeId);
        if (!current) {
          const fallback =
            fetchedNodes.find((n) => n.status === "running") ??
            fetchedNodes[0];
          if (fallback) {
            navigate(`/terminal/${clusterId}/${fallback.id}`, {
              replace: true,
            });
          }
        }
      } catch (err) {
        if (isMounted) {
          setError(
            err instanceof Error ? err.message : "Failed to load nodes",
          );
        }
      } finally {
        if (isMounted) setLoading(false);
      }
    };

    load();
    return () => {
      isMounted = false;
    };
  }, [clusterId, nodeId, navigate]);

  if (status === "loading") {
    return <FullPageSpinner />;
  }
  if (status !== "authenticated") {
    return <Navigate to="/login" replace />;
  }
  if (!clusterId || !nodeId) {
    return <Navigate to="/clusters" replace />;
  }

  const activeNode = nodes.find((n) => n.id === nodeId);

  return (
    <div className="flex h-screen w-full flex-col bg-background">
      <header className="flex shrink-0 flex-wrap items-center gap-3 border-b bg-card px-4 py-3">
        <Link
          to={`/clusters/${clusterId}`}
          className="inline-flex items-center gap-1 text-sm font-medium text-muted-foreground transition-colors hover:text-foreground"
        >
          <ArrowLeft className="h-4 w-4" />
          Back
        </Link>

        <Select
          value={clusterId}
          onValueChange={(next) => {
            if (next === clusterId) return;
            navigate(`/terminal/${next}/_`, { replace: true });
          }}
        >
          <SelectTrigger className="w-[220px]">
            <SelectValue placeholder="Select a cluster" />
          </SelectTrigger>
          <SelectContent>
            {clusters.map((c) => (
              <SelectItem key={c.id} value={c.id}>
                {c.name}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>

        <div className="flex flex-1 flex-wrap items-center gap-1">
          {nodes.map((node) => {
            const isActive = node.id === nodeId;
            const isRunning = node.status === "running";
            return (
              <Button
                key={node.id}
                variant={isActive ? "secondary" : "ghost"}
                size="sm"
                disabled={!isRunning}
                title={!isRunning ? "Node must be running to open a terminal" : undefined}
                onClick={() =>
                  navigate(`/terminal/${clusterId}/${node.id}`, {
                    replace: true,
                  })
                }
                className={cn("gap-1.5", isActive && "pointer-events-none")}
              >
                {node.name}
                <Badge variant="outline" className="text-[10px] capitalize">
                  {node.role}
                </Badge>
              </Button>
            );
          })}
        </div>
      </header>

      <main className="min-h-0 flex-1 p-3">
        {error && (
          <Alert variant="destructive" className="mb-3">
            <AlertCircle className="h-4 w-4" />
            <AlertDescription>{error}</AlertDescription>
          </Alert>
        )}

        {loading && !activeNode ? (
          <Skeleton className="h-full w-full rounded-lg" />
        ) : activeNode ? (
          <TerminalPane
            key={`${clusterId}-${nodeId}`}
            clusterId={clusterId}
            nodeId={nodeId}
            className="h-full w-full overflow-hidden rounded-lg bg-black p-2"
          />
        ) : (
          !loading && (
            <div className="flex h-full items-center justify-center text-sm text-muted-foreground">
              No running nodes in this cluster.
            </div>
          )
        )}
      </main>
    </div>
  );
}
