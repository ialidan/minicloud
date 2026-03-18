import { useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { UserPlus, Loader2 } from "lucide-react";
import { api } from "@/lib/api";
import { useToast } from "@/lib/toast";
import type { User } from "@/lib/types";
import { Button } from "@/components/ui/button";
import { UserTable } from "@/features/admin/components/user-table";
import { AddUserModal } from "@/features/admin/components/add-user-modal";
import { EditUserModal } from "@/features/admin/components/edit-user-modal";
import { useCreateUser, useUpdateUser } from "@/features/admin/hooks/use-admin-users";

export function AdminUsersPage() {
  const { data, isLoading } = useQuery({
    queryKey: ["admin-users"],
    queryFn: () => api.get<{ users: User[] }>("/admin/users"),
  });

  const { toast } = useToast();
  const createUser = useCreateUser();
  const updateUser = useUpdateUser();

  const [addOpen, setAddOpen] = useState(false);
  const [editTarget, setEditTarget] = useState<User | null>(null);

  async function handleCreate(userData: {
    username: string;
    password: string;
    email?: string;
    role?: string;
  }) {
    await createUser.mutateAsync(userData);
    toast("User created", "success");
  }

  async function handleEdit(userData: {
    id: string;
    email?: string;
    role?: string;
    is_active?: boolean;
    password?: string;
  }) {
    await updateUser.mutateAsync(userData);
    toast("User updated", "success");
  }

  async function handleToggleActive(user: User) {
    await updateUser.mutateAsync({
      id: user.id,
      is_active: !user.is_active,
    });
    toast(
      user.is_active ? "User deactivated" : "User activated",
      "success",
    );
  }

  const users = data?.users ?? [];

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-xl font-semibold tracking-tight">Users</h1>
          <p className="text-sm text-muted-foreground">
            Manage user accounts and permissions
          </p>
        </div>
        <Button onClick={() => setAddOpen(true)}>
          <UserPlus className="h-4 w-4" />
          Add User
        </Button>
      </div>

      {isLoading ? (
        <div className="flex items-center justify-center py-12">
          <Loader2 className="h-6 w-6 animate-spin text-muted-foreground" />
        </div>
      ) : users.length === 0 ? (
        <div className="flex flex-col items-center justify-center py-16 text-center">
          <p className="text-sm text-muted-foreground">No users found</p>
        </div>
      ) : (
        <UserTable
          users={users}
          onEdit={setEditTarget}
          onToggleActive={handleToggleActive}
        />
      )}

      <AddUserModal
        open={addOpen}
        onClose={() => setAddOpen(false)}
        onSubmit={handleCreate}
      />

      <EditUserModal
        open={editTarget !== null}
        onClose={() => setEditTarget(null)}
        user={editTarget}
        onSubmit={handleEdit}
      />
    </div>
  );
}
