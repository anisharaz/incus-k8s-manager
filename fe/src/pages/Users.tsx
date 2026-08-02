import { useEffect, useState } from "react";
import { Navigate } from "react-router";
import { Users as UsersIcon, AlertCircle } from "lucide-react";

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
import { CreateUserDialog } from "@/components/CreateUserDialog";
import { api } from "@/lib/api";
import { useAuth } from "@/context";
import type { User } from "@/lib/types";

export function Users() {
  const { user: currentUser } = useAuth();
  const [users, setUsers] = useState<User[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const fetchUsers = async () => {
    try {
      setLoading(true);
      setError(null);
      const data = await api.get<{ users: User[] }>("/api/v1/users");
      setUsers(data.users ?? []);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to fetch users");
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    // No polling — users are created rarely and only by an admin action in
    // this same UI, so a manual refetch after creating one is enough.
    let isMounted = true;
    const load = async () => {
      if (isMounted) await fetchUsers();
    };
    load();
    return () => {
      isMounted = false;
    };
  }, []);

  // Backend already 403s non-admins, but redirect before even trying —
  // there's nothing useful to show a regular user on this page.
  if (currentUser?.role !== "admin") {
    return <Navigate to="/clusters" replace />;
  }

  return (
    <div className="space-y-6">
      <section className="rounded-3xl border bg-card p-6 shadow-sm">
        <div className="flex flex-col gap-4 md:flex-row md:items-end md:justify-between">
          <div>
            <div className="mb-3 inline-flex items-center gap-2 rounded-full bg-muted px-3 py-1 text-xs font-medium uppercase tracking-[0.2em] text-muted-foreground">
              <UsersIcon className="h-3.5 w-3.5" />
              Users
            </div>
            <h2 className="text-3xl font-semibold tracking-tight text-foreground">
              User accounts
            </h2>
            <p className="mt-2 max-w-2xl text-sm text-muted-foreground">
              Regular users you've created. Each owns their own networks,
              clusters, and nodes, separate from everyone else's.
            </p>
          </div>
          <CreateUserDialog onSuccess={fetchUsers} />
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
      ) : users.length === 0 ? (
        <div className="rounded-3xl border bg-card p-12 text-center">
          <p className="text-sm text-muted-foreground">No users yet.</p>
          <p className="mt-2 text-sm text-muted-foreground">
            Click "Create User" to add one.
          </p>
        </div>
      ) : (
        <div className="rounded-3xl border bg-card p-2 shadow-sm">
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>Username</TableHead>
                <TableHead>Role</TableHead>
                <TableHead>Created</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {users.map((user) => (
                <TableRow key={user.id}>
                  <TableCell className="font-medium">
                    {user.username}
                  </TableCell>
                  <TableCell>
                    <Badge
                      variant={user.role === "admin" ? "default" : "secondary"}
                      className="capitalize"
                    >
                      {user.role}
                    </Badge>
                  </TableCell>
                  <TableCell>
                    {new Date(user.createdAt).toLocaleDateString()}
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
