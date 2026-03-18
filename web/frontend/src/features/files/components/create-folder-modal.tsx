import { useState, type FormEvent } from "react";
import { Modal } from "@/components/ui/modal";
import { Input } from "@/components/ui/input";
import { Button } from "@/components/ui/button";
import { Loader2 } from "lucide-react";

interface CreateFolderModalProps {
  open: boolean;
  onClose: () => void;
  onSubmit: (name: string) => Promise<void>;
}

export function CreateFolderModal({
  open,
  onClose,
  onSubmit,
}: CreateFolderModalProps) {
  const [name, setName] = useState("");
  const [error, setError] = useState("");
  const [submitting, setSubmitting] = useState(false);

  async function handleSubmit(e: FormEvent) {
    e.preventDefault();
    setError("");

    const trimmed = name.trim();
    if (!trimmed) {
      setError("Folder name is required.");
      return;
    }

    if (trimmed.includes("/")) {
      setError("Folder name cannot contain slashes.");
      return;
    }

    setSubmitting(true);
    try {
      await onSubmit(trimmed);
      setName("");
      onClose();
    } catch {
      setError("Failed to create folder.");
    } finally {
      setSubmitting(false);
    }
  }

  function handleClose() {
    setName("");
    setError("");
    onClose();
  }

  return (
    <Modal open={open} onClose={handleClose} title="New Folder">
      <form onSubmit={handleSubmit} className="space-y-4">
        {error && (
          <div className="rounded-lg bg-danger/10 px-3 py-2 text-sm text-danger">
            {error}
          </div>
        )}
        <Input
          label="Folder name"
          placeholder="my-folder"
          value={name}
          onChange={(e) => setName(e.target.value)}
          disabled={submitting}
          autoFocus
        />
        <div className="flex justify-end gap-2">
          <Button
            type="button"
            variant="secondary"
            onClick={handleClose}
            disabled={submitting}
          >
            Cancel
          </Button>
          <Button type="submit" disabled={submitting}>
            {submitting && <Loader2 className="h-4 w-4 animate-spin" />}
            Create
          </Button>
        </div>
      </form>
    </Modal>
  );
}
