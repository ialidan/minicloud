import { MoreHorizontal, Pencil, UserCheck, UserX } from "lucide-react";
import type { User } from "@/lib/types";
import { formatDate } from "@/lib/format";
import { Button } from "@/components/ui/button";
import { DropdownMenu, DropdownItem } from "@/components/ui/dropdown-menu";

interface UserTableProps {
  users: User[];
  onEdit: (user: User) => void;
  onToggleActive: (user: User) => void;
}

export function UserTable({ users, onEdit, onToggleActive }: UserTableProps) {
  return (
    <div className="overflow-hidden rounded-lg border border-border bg-surface">
      <table className="w-full text-sm">
        <thead>
          <tr className="border-b border-border text-left text-xs text-muted-foreground">
            <th scope="col" className="px-4 py-3 font-medium">
              Username
            </th>
            <th scope="col" className="hidden px-4 py-3 font-medium sm:table-cell">
              Email
            </th>
            <th scope="col" className="px-4 py-3 font-medium">
              Role
            </th>
            <th scope="col" className="hidden px-4 py-3 font-medium md:table-cell">
              Status
            </th>
            <th scope="col" className="hidden px-4 py-3 font-medium md:table-cell">
              Created
            </th>
            <th scope="col" className="px-4 py-3 text-right font-medium w-16">
              <span className="sr-only">Actions</span>
            </th>
          </tr>
        </thead>
        <tbody>
          {users.map((user) => (
            <tr
              key={user.id}
              className="border-b border-border last:border-0 hover:bg-surface-hover transition-colors"
            >
              <td className="px-4 py-3 font-medium">{user.username}</td>
              <td className="hidden px-4 py-3 text-muted-foreground sm:table-cell">
                {user.email || <span className="text-muted-foreground/50">&mdash;</span>}
              </td>
              <td className="px-4 py-3">
                <span
                  className={`inline-flex items-center rounded-full px-2 py-0.5 text-xs font-medium ${
                    user.role === "admin"
                      ? "bg-primary/10 text-primary"
                      : "bg-muted text-muted-foreground"
                  }`}
                >
                  {user.role}
                </span>
              </td>
              <td className="hidden px-4 py-3 md:table-cell">
                <span
                  className={`inline-flex items-center rounded-full px-2 py-0.5 text-xs font-medium ${
                    user.is_active
                      ? "bg-green-500/10 text-green-600 dark:text-green-400"
                      : "bg-danger/10 text-danger"
                  }`}
                >
                  {user.is_active ? "Active" : "Inactive"}
                </span>
              </td>
              <td className="hidden px-4 py-3 text-muted-foreground md:table-cell">
                {formatDate(user.created_at)}
              </td>
              <td className="px-4 py-3 text-right">
                <DropdownMenu
                  trigger={
                    <Button variant="ghost" size="icon" aria-label="Actions">
                      <MoreHorizontal className="h-4 w-4" />
                    </Button>
                  }
                >
                  <DropdownItem onClick={() => onEdit(user)}>
                    <Pencil className="h-4 w-4" />
                    Edit
                  </DropdownItem>
                  <DropdownItem onClick={() => onToggleActive(user)}>
                    {user.is_active ? (
                      <>
                        <UserX className="h-4 w-4" />
                        Deactivate
                      </>
                    ) : (
                      <>
                        <UserCheck className="h-4 w-4" />
                        Activate
                      </>
                    )}
                  </DropdownItem>
                </DropdownMenu>
              </td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}
