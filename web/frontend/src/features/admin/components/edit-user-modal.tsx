import { useState, useEffect, type FormEvent } from "react";
import { Loader2 } from "lucide-react";
import { Modal } from "@/components/ui/modal";
import { Input } from "@/components/ui/input";
import { Button } from "@/components/ui/button";
import { ApiError } from "@/lib/api";
import type { User } from "@/lib/types";

interface EditUserModalProps {
  open: boolean;
  onClose: () => void;
  user: User | null;
  onSubmit: (data: {
    id: string;
    email?: string;
    role?: string;
    is_active?: boolean;
    password?: string;
  }) => Promise<void>;
}

export function EditUserModal({
  open,
  onClose,
  user,
  onSubmit,
}: EditUserModalProps) {
  const [email, setEmail] = useState("");
  const [role, setRole] = useState("user");
  const [isActive, setIsActive] = useState(true);
  const [password, setPassword] = useState("");
  const [error, setError] = useState("");
  const [submitting, setSubmitting] = useState(false);

  useEffect(() => {
    if (user) {
      setEmail(user.email);
      setRole(user.role);
      setIsActive(user.is_active);
      setPassword("");
      setError("");
    }
  }, [user]);

  async function handleSubmit(e: FormEvent) {
    e.preventDefault();
    setError("");
    if (!user) return;

    if (password && password.length < 8) {
      setError("Password must be at least 8 characters.");
      return;
    }

    setSubmitting(true);
    try {
      const data: {
        id: string;
        email?: string;
        role?: string;
        is_active?: boolean;
        password?: string;
      } = {
        id: user.id,
        email: email.trim(),
        role,
        is_active: isActive,
      };
      if (password) data.password = password;

      await onSubmit(data);
      onClose();
    } catch (err) {
      if (err instanceof ApiError) {
        setError(err.message);
      } else {
        setError("Failed to update user.");
      }
    } finally {
      setSubmitting(false);
    }
  }

  if (!user) return null;

  return (
    <Modal open={open} onClose={onClose} title={`Edit ${user.username}`}>
      <form onSubmit={handleSubmit} className="space-y-4">
        {error && (
          <div className="rounded-lg bg-danger/10 px-3 py-2 text-sm text-danger">
            {error}
          </div>
        )}
        <Input
          label="Email"
          type="email"
          placeholder="john@example.com"
          value={email}
          onChange={(e) => setEmail(e.target.value)}
          disabled={submitting}
        />
        <div>
          <label className="mb-1.5 block text-sm font-medium">Role</label>
          <select
            value={role}
            onChange={(e) => setRole(e.target.value)}
            disabled={submitting}
            className="h-9 w-full rounded-lg border border-border bg-surface px-3 text-sm text-foreground outline-none transition-colors focus:border-ring focus:ring-1 focus:ring-ring"
          >
            <option value="user">User</option>
            <option value="admin">Admin</option>
          </select>
        </div>
        <label className="flex items-center gap-2 text-sm">
          <input
            type="checkbox"
            checked={isActive}
            onChange={(e) => setIsActive(e.target.checked)}
            disabled={submitting}
            className="h-4 w-4 rounded border-border"
          />
          Account is active
        </label>
        <Input
          label="New Password (leave blank to keep current)"
          type="password"
          placeholder="••••••••"
          value={password}
          onChange={(e) => setPassword(e.target.value)}
          disabled={submitting}
        />
        <div className="flex justify-end gap-2">
          <Button
            type="button"
            variant="secondary"
            onClick={onClose}
            disabled={submitting}
          >
            Cancel
          </Button>
          <Button type="submit" disabled={submitting}>
            {submitting && <Loader2 className="h-4 w-4 animate-spin" />}
            Save Changes
          </Button>
        </div>
      </form>
    </Modal>
  );
}
