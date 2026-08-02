import { useEffect, useState } from "react";
import { Network as NetworkIcon, AlertCircle, Trash2 } from "lucide-react";
import { toast } from "sonner";

import { Badge } from "@/components/ui/badge";
import { Skeleton } from "@/components/ui/skeleton";
import { Alert, AlertDescription } from "@/components/ui/alert";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
  AlertDialogTrigger,
} from "@/components/ui/alert-dialog";
import { Button } from "@/components/ui/button";
import { CreateNetworkDialog } from "@/components/CreateNetworkDialog";
import { api, ApiError } from "@/lib/api";
import type { ClusterNetwork, ClusterNetworkStatus } from "@/lib/types";

const statusVariant: Record<
  ClusterNetworkStatus,
  "default" | "secondary" | "destructive"
> = {
  creating: "secondary",
  ready: "default",
  failed: "destructive",
};

export function Networks() {
  const [networks, setNetworks] = useState<ClusterNetwork[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const fetchNetworks = async () => {
    try {
      setLoading(true);
      setError(null);
      const data = await api.get<{ networks: ClusterNetwork[] }>(
        "/api/v1/networks",
      );
      setNetworks(data.networks ?? []);
    } catch (err) {
      setError(
        err instanceof Error ? err.message : "Failed to fetch networks",
      );
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    // No polling — networks are created/deleted synchronously (no
    // background job), so a manual refetch after each action is enough.
    let isMounted = true;
    const load = async () => {
      if (isMounted) await fetchNetworks();
    };
    load();
    return () => {
      isMounted = false;
    };
  }, []);

  async function handleDelete(network: ClusterNetwork) {
    try {
      await api.delete(`/api/v1/networks/${network.id}`);
      toast.success(`Network "${network.name}" deleted`);
    } catch (err) {
      toast.error(
        err instanceof ApiError ? err.message : "Failed to delete network",
      );
    } finally {
      fetchNetworks();
    }
  }

  return (
    <div className="space-y-6">
      <section className="rounded-3xl border bg-card p-6 shadow-sm">
        <div className="flex flex-col gap-4 md:flex-row md:items-end md:justify-between">
          <div>
            <div className="mb-3 inline-flex items-center gap-2 rounded-full bg-muted px-3 py-1 text-xs font-medium uppercase tracking-[0.2em] text-muted-foreground">
              <NetworkIcon className="h-3.5 w-3.5" />
              Networks
            </div>
            <h2 className="text-3xl font-semibold tracking-tight text-foreground">
              Cluster networks
            </h2>
            <p className="mt-2 max-w-2xl text-sm text-muted-foreground">
              Incus bridge networks that clusters are launched onto. Create
              one before creating a cluster.
            </p>
          </div>
          <CreateNetworkDialog onSuccess={fetchNetworks} />
        </div>
      </section>

      {error && (
        <Alert variant="destructive">
          <AlertCircle className="h-4 w-4" />
          <AlertDescription>{error}</AlertDescription>
        </Alert>
      )}

      {loading ? (
        <div className="space-y-2">
          <Skeleton className="h-10 w-full" />
          <Skeleton className="h-10 w-full" />
          <Skeleton className="h-10 w-full" />
        </div>
      ) : networks.length === 0 ? (
        <div className="rounded-3xl border bg-card p-12 text-center">
          <p className="text-sm text-muted-foreground">No networks yet.</p>
          <p className="mt-2 text-sm text-muted-foreground">
            Click "Create Network" to get started.
          </p>
        </div>
      ) : (
        <div className="rounded-3xl border bg-card p-2 shadow-sm">
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>Name</TableHead>
                <TableHead>CIDR</TableHead>
                <TableHead>Gateway</TableHead>
                <TableHead>Status</TableHead>
                <TableHead>Created</TableHead>
                <TableHead className="text-right">Actions</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {networks.map((network) => (
                <TableRow key={network.id}>
                  <TableCell className="font-medium">
                    {network.name}
                  </TableCell>
                  <TableCell>{network.cidr}</TableCell>
                  <TableCell>{network.gateway}</TableCell>
                  <TableCell>
                    <Badge
                      variant={statusVariant[network.status]}
                      className="capitalize"
                    >
                      {network.status}
                    </Badge>
                  </TableCell>
                  <TableCell>
                    {new Date(network.createdAt).toLocaleDateString()}
                  </TableCell>
                  <TableCell className="text-right">
                    <AlertDialog>
                      <AlertDialogTrigger asChild>
                        <Button
                          variant="ghost"
                          size="icon"
                          aria-label="Delete network"
                        >
                          <Trash2 className="h-4 w-4 text-destructive" />
                        </Button>
                      </AlertDialogTrigger>
                      <AlertDialogContent>
                        <AlertDialogHeader>
                          <AlertDialogTitle>
                            Delete network "{network.name}"?
                          </AlertDialogTitle>
                          <AlertDialogDescription>
                            This deletes the underlying Incus bridge network
                            too. This can't be undone, and will fail if a
                            cluster still uses it.
                          </AlertDialogDescription>
                        </AlertDialogHeader>
                        <AlertDialogFooter>
                          <AlertDialogCancel>Cancel</AlertDialogCancel>
                          <AlertDialogAction
                            onClick={() => handleDelete(network)}
                          >
                            Delete
                          </AlertDialogAction>
                        </AlertDialogFooter>
                      </AlertDialogContent>
                    </AlertDialog>
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        </div>
      )}
    </div>
  );
}
