import { useEffect, useState } from "react";
import { Link, useParams } from "react-router";
import { ArrowLeft, ServerCog, AlertCircle } from "lucide-react";

import { Badge } from "@/components/ui/badge";
import { Card, CardContent } from "@/components/ui/card";
import { Progress } from "@/components/ui/progress";
import { Skeleton } from "@/components/ui/skeleton";
import { Alert, AlertTitle, AlertDescription } from "@/components/ui/alert";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { AddNodeDialog } from "@/components/AddNodeDialog";
import { api } from "@/lib/api";
import type {
  Cluster,
  ClusterNode,
  ClusterStatus,
  Job,
  NodeStatus,
} from "@/lib/types";

const clusterStatusVariant: Record<
  ClusterStatus,
  "default" | "secondary" | "destructive" | "outline"
> = {
  creating: "secondary",
  ready: "default",
  failed: "destructive",
  deleting: "outline",
};

const nodeStatusVariant: Record<
  NodeStatus,
  "default" | "secondary" | "destructive" | "outline"
> = {
  creating: "secondary",
  running: "default",
  stopped: "outline",
  failed: "destructive",
  deleting: "outline",
};

export function ClusterDetail() {
  const { clusterId } = useParams();
  const [cluster, setCluster] = useState<Cluster | null>(null);
  const [nodes, setNodes] = useState<ClusterNode[]>([]);
  const [nodeJobs, setNodeJobs] = useState<Record<string, Job>>({});
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    if (!clusterId) return;
    let isMounted = true;

    const fetchAll = async () => {
      try {
        setError(null);
        const [clusterData, nodesData] = await Promise.all([
          api.get<{ cluster: Cluster }>(`/api/v1/clusters/${clusterId}`),
          api.get<{ nodes: ClusterNode[] }>(
            `/api/v1/clusters/${clusterId}/nodes`,
          ),
        ]);
        if (!isMounted) return;
        setCluster(clusterData.cluster);
        setNodes(nodesData.nodes ?? []);

        // Only nodes still provisioning need their job polled for
        // stage/progress — a finished node's outcome is fully captured by
        // its own status/message already.
        const creatingNodes = (nodesData.nodes ?? []).filter(
          (n) => n.status === "creating",
        );
        const jobEntries = await Promise.all(
          creatingNodes.map(async (node) => {
            try {
              const jobData = await api.get<{ job: Job }>(
                `/api/v1/jobs/${node.jobId}`,
              );
              return [node.id, jobData.job] as const;
            } catch {
              return null;
            }
          }),
        );
        if (!isMounted) return;
        setNodeJobs((prev) => {
          const next = { ...prev };
          for (const entry of jobEntries) {
            if (entry) next[entry[0]] = entry[1];
          }
          return next;
        });
      } catch (err) {
        if (!isMounted) return;
        setError(
          err instanceof Error ? err.message : "Failed to fetch cluster",
        );
      } finally {
        if (isMounted) setLoading(false);
      }
    };

    fetchAll();
    const interval = setInterval(fetchAll, 3000);
    return () => {
      isMounted = false;
      clearInterval(interval);
    };
  }, [clusterId]);

  if (loading) {
    return (
      <div className="rounded-3xl border bg-card p-6 shadow-sm space-y-4">
        <Skeleton className="h-8 w-1/3" />
        <Skeleton className="h-4 w-1/2" />
      </div>
    );
  }

  if (error || !cluster) {
    return (
      <div className="rounded-3xl border bg-card p-6 shadow-sm">
        <p className="text-sm uppercase tracking-[0.2em] text-muted-foreground">
          Cluster
        </p>
        <h2 className="mt-2 text-2xl font-semibold text-foreground">
          Cluster not found
        </h2>
        <p className="mt-2 text-muted-foreground">
          {error ||
            "The cluster id you opened does not match any known cluster."}
        </p>
        <Link
          to="/clusters"
          className="mt-6 inline-flex items-center gap-2 rounded-lg bg-primary px-4 py-2 text-sm font-medium text-primary-foreground transition-colors hover:bg-primary/90"
        >
          <ArrowLeft className="h-4 w-4" />
          Back to clusters
        </Link>
      </div>
    );
  }

  const master = nodes.find((n) => n.role === "master");
  const masterJob = master ? nodeJobs[master.id] : undefined;
  const canAddWorker = cluster.status === "ready" && master?.status === "running";

  return (
    <div className="space-y-6">
      <Link
        to="/clusters"
        className="inline-flex items-center gap-2 text-sm font-medium text-muted-foreground transition-colors hover:text-foreground"
      >
        <ArrowLeft className="h-4 w-4" />
        Back to clusters
      </Link>

      <section className="rounded-3xl border bg-card p-6 shadow-sm">
        <div className="flex flex-col gap-4 md:flex-row md:items-start md:justify-between">
          <div>
            <div className="mb-3 inline-flex items-center gap-2 rounded-full bg-muted px-3 py-1 text-xs font-medium uppercase tracking-[0.2em] text-muted-foreground">
              <ServerCog className="h-3.5 w-3.5" />
              Cluster detail
            </div>
            <h2 className="text-3xl font-semibold tracking-tight text-foreground">
              {cluster.name}
            </h2>
            <p className="mt-2 text-sm text-muted-foreground">
              Cluster ID: {cluster.id}
            </p>
          </div>
          <div className="rounded-2xl bg-muted px-4 py-3 text-sm text-muted-foreground">
            <p className="flex items-center gap-2 font-medium text-foreground">
              Status:
              <Badge
                variant={clusterStatusVariant[cluster.status]}
                className="capitalize"
              >
                {cluster.status}
              </Badge>
            </p>
            {master?.ip && <p className="mt-1">Master IP: {master.ip}</p>}
          </div>
        </div>

        <div className="mt-6 grid gap-4 sm:grid-cols-2 xl:grid-cols-4">
          <DetailCard label="Status" value={cluster.status} />
          <DetailCard label="CNI" value={cluster.cni} />
          <DetailCard
            label="Created"
            value={new Date(cluster.createdAt).toLocaleString()}
          />
          {master?.ip && <DetailCard label="Master IP" value={master.ip} />}
          <DetailCard
            label="Last Updated"
            value={new Date(cluster.updatedAt).toLocaleString()}
          />
        </div>

        {cluster.message && (
          <Alert className="mt-6">
            <AlertDescription>{cluster.message}</AlertDescription>
          </Alert>
        )}
      </section>

      {masterJob && masterJob.status !== "succeeded" && (
        <section className="rounded-3xl border bg-card p-6 shadow-sm">
          <h3 className="text-lg font-semibold text-foreground mb-4">
            Creation Progress
          </h3>

          <div className="space-y-4">
            <div>
              <div className="flex items-center justify-between mb-2">
                <span className="text-sm font-medium text-muted-foreground">
                  {masterJob.stage}
                </span>
                <span className="text-sm font-medium text-foreground">
                  {masterJob.progress}%
                </span>
              </div>
              <Progress value={masterJob.progress} />
            </div>

            <div>
              <p className="text-sm text-muted-foreground font-medium">
                Status
              </p>
              <p className="mt-1 text-sm text-foreground capitalize">
                {masterJob.status}
              </p>
            </div>

            <div>
              <p className="text-sm text-muted-foreground font-medium">
                Message
              </p>
              <p className="mt-1 text-sm text-foreground">
                {masterJob.message}
              </p>
            </div>

            {masterJob.error && (
              <Alert variant="destructive">
                <AlertCircle className="h-4 w-4" />
                <AlertTitle>Provisioning failed</AlertTitle>
                <AlertDescription className="whitespace-pre-wrap break-words">
                  {masterJob.error}
                </AlertDescription>
              </Alert>
            )}
          </div>
        </section>
      )}

      <section className="rounded-3xl border bg-card p-6 shadow-sm">
        <div className="flex items-center justify-between mb-4">
          <h3 className="text-lg font-semibold text-foreground">Nodes</h3>
          <AddNodeDialog
            clusterId={cluster.id}
            disabled={!canAddWorker}
            disabledReason={
              !canAddWorker
                ? "The cluster's master must be ready and running first"
                : undefined
            }
          />
        </div>

        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>Name</TableHead>
              <TableHead>Role</TableHead>
              <TableHead>Status</TableHead>
              <TableHead>IP</TableHead>
              <TableHead>Message</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {nodes.map((node) => {
              const job = nodeJobs[node.id];
              return (
                <TableRow key={node.id}>
                  <TableCell className="font-medium">{node.name}</TableCell>
                  <TableCell>
                    <Badge variant="outline" className="capitalize">
                      {node.role}
                    </Badge>
                  </TableCell>
                  <TableCell>
                    <div className="flex flex-col gap-1">
                      <Badge
                        variant={nodeStatusVariant[node.status]}
                        className="w-fit capitalize"
                      >
                        {node.status}
                      </Badge>
                      {node.status === "creating" && job && (
                        <span className="text-xs text-muted-foreground">
                          {job.stage} ({job.progress}%)
                        </span>
                      )}
                    </div>
                  </TableCell>
                  <TableCell>{node.ip || "—"}</TableCell>
                  <TableCell className="max-w-xs truncate text-sm text-muted-foreground">
                    {node.message}
                  </TableCell>
                </TableRow>
              );
            })}
          </TableBody>
        </Table>
      </section>
    </div>
  );
}

function DetailCard({ label, value }: { label: string; value: string }) {
  return (
    <Card size="sm">
      <CardContent>
        <p className="text-xs uppercase tracking-wide text-muted-foreground">
          {label}
        </p>
        <p className="mt-2 text-base font-medium text-foreground">{value}</p>
      </CardContent>
    </Card>
  );
}
