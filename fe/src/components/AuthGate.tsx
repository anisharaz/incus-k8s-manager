import { Navigate } from "react-router";
import { useAuth } from "@/context";
import { FullPageSpinner } from "./FullPageSpinner";
import { Login } from "@/pages/Login";
import { RegisterAdmin } from "@/pages/RegisterAdmin";

// Route element for /login. Bootstrap-vs-login is a one-time, server-derived
// state, so it lives at a single route rather than two separate URLs.
export function AuthGate() {
  const { status } = useAuth();

  switch (status) {
    case "loading":
      return <FullPageSpinner />;
    case "needs-bootstrap":
      return <RegisterAdmin />;
    case "needs-login":
      return <Login />;
    case "authenticated":
      return <Navigate to="/clusters" replace />;
  }
}
