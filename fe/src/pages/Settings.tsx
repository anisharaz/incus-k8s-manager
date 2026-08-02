import type { ReactNode } from "react";
import { Settings2, Server } from "lucide-react";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { useAuth, useStatus } from "@/context";

export function Settings() {
  const { user } = useAuth();
  const { status, loading } = useStatus();

  return (
    <div className="space-y-6">
      <div>
        <p className="text-sm uppercase tracking-[0.2em] text-muted-foreground">
          Settings
        </p>
        <h2 className="mt-2 text-3xl font-semibold tracking-tight text-foreground">
          Account & environment
        </h2>
      </div>

      <Card>
        <CardHeader>
          <div className="flex items-center gap-2">
            <Settings2 className="h-4 w-4 text-muted-foreground" />
            <CardTitle>Account</CardTitle>
          </div>
          <CardDescription>Your current session.</CardDescription>
        </CardHeader>
        <CardContent className="space-y-3">
          <Row label="Username" value={user?.username ?? "—"} />
          <Row
            label="Role"
            value={
              <Badge
                variant={user?.role === "admin" ? "default" : "secondary"}
                className="capitalize"
              >
                {user?.role ?? "—"}
              </Badge>
            }
          />
          <Row
            label="Member since"
            value={
              user ? new Date(user.createdAt).toLocaleDateString() : "—"
            }
          />
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <div className="flex items-center gap-2">
            <Server className="h-4 w-4 text-muted-foreground" />
            <CardTitle>Incus</CardTitle>
          </div>
          <CardDescription>
            Status of the Incus daemon backing this instance.
          </CardDescription>
        </CardHeader>
        <CardContent>
          <Row
            label="Status"
            value={
              loading ? (
                "Loading..."
              ) : (
                <span className="capitalize">
                  {status?.incus ?? "unknown"}
                </span>
              )
            }
          />
        </CardContent>
      </Card>
    </div>
  );
}

function Row({ label, value }: { label: string; value: ReactNode }) {
  return (
    <div className="flex items-center justify-between gap-4 text-sm">
      <span className="text-muted-foreground">{label}</span>
      <span className="font-medium text-foreground">{value}</span>
    </div>
  );
}
