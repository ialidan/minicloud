import { Navigate, Outlet } from "react-router-dom";
import { useAuth } from "@/lib/auth";

export function AdminRoute() {
  const { user } = useAuth();
  if (!user || user.role !== "admin") {
    return <Navigate to="/files" replace />;
  }
  return <Outlet />;
}
