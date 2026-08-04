import { useStatus } from "@/context";
import { ModeToggle } from "./ModeToggle";

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
  const { status, loading, error } = useStatus();

  return (
    <header className="border-b bg-background">
      <div className="container mx-auto flex items-center justify-between px-4 py-4">
        <h1 className="text-2xl font-bold text-foreground">
          KOI <span className="text-base font-normal text-muted-foreground">— Kubernetes on Incus</span>
        </h1>

        <div className="flex items-center gap-6">
          <div className="flex items-center gap-2">
            <span className="text-sm font-medium text-muted-foreground">
              Incus Status:
            </span>
            {loading ? (
              <span className="text-sm text-muted-foreground">
                Loading...
              </span>
            ) : error ? (
              <span className="text-sm text-destructive">Error: {error}</span>
            ) : (
              <div className="flex items-center gap-2">
                <div
                  className={`h-3 w-3 rounded-full ${getStatusColor(status?.incus || "")}`}
                />
                <span className="text-sm font-medium text-foreground">
                  {status?.incus}
                </span>
              </div>
            )}
          </div>
          <ModeToggle />
        </div>
      </div>
    </header>
  );
}
