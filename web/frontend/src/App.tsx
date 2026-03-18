import { Routes, Route, Navigate } from "react-router-dom";
import { LoginPage } from "@/pages/login";
import { FilesPage } from "@/pages/files";
import { AdminUsersPage } from "@/pages/admin-users";
import { AppLayout } from "@/components/layout/app-layout";
import { ProtectedRoute } from "@/components/layout/protected-route";
import { AdminRoute } from "@/components/layout/admin-route";

export function App() {
  return (
    <Routes>
      <Route path="/login" element={<LoginPage />} />
      <Route element={<ProtectedRoute />}>
        <Route element={<AppLayout />}>
          <Route path="/files" element={<FilesPage />} />
          <Route path="/files/*" element={<FilesPage />} />
          <Route element={<AdminRoute />}>
            <Route path="/admin/users" element={<AdminUsersPage />} />
          </Route>
        </Route>
      </Route>
      <Route path="*" element={<Navigate to="/files" replace />} />
    </Routes>
  );
}
