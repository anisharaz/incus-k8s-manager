import { Navigate, Outlet } from "react-router";
import { Header } from "./Header";
import { SidebarProvider } from "./ui/sidebar";
import { AppSidebar } from "./Sidebar";
import { Toaster } from "./ui/sonner";
import { FullPageSpinner } from "./FullPageSpinner";
import { StatusProvider, JobProvider, useAuth } from "@/context";

// Shell for every authenticated page (sidebar + header + routed content).
// StatusProvider/JobProvider live here, not globally in main.tsx, so their
// polling only ever runs while a session actually exists — JobProvider in
// particular hits an auth-required endpoint every 3s, which would otherwise
// spam 401s while sitting on the login screen.
export function ProtectedLayout() {
  const { status } = useAuth();

  if (status === "loading") {
    return <FullPageSpinner />;
  }

  if (status !== "authenticated") {
    return <Navigate to="/login" replace />;
  }

  return (
    <StatusProvider>
      <JobProvider>
        <SidebarProvider>
          <div className="flex w-full min-h-screen">
            <AppSidebar />
            <div className="flex-1 flex flex-col">
              <Header />
              <main className="flex-1 container mx-auto px-4 py-8">
                <Outlet />
              </main>
            </div>
          </div>
        </SidebarProvider>
        <Toaster />
      </JobProvider>
    </StatusProvider>
  );
}
