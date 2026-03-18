import { useState, type FormEvent } from "react";
import { Modal } from "@/components/ui/modal";
import { Input } from "@/components/ui/input";
import { Button } from "@/components/ui/button";
import { Loader2 } from "lucide-react";

interface MoveModalProps {
  open: boolean;
  onClose: () => void;
  fileName: string;
  onMove: (destination: string) => Promise<void>;
}

export function MoveModal({
  open,
  onClose,
  fileName,
  onMove,
}: MoveModalProps) {
  const [destination, setDestination] = useState("/");
  const [error, setError] = useState("");
  const [submitting, setSubmitting] = useState(false);

  async function handleSubmit(e: FormEvent) {
    e.preventDefault();
    setError("");

    const trimmed = destination.trim();
    if (!trimmed) {
      setError("Destination path is required.");
      return;
    }

    setSubmitting(true);
    try {
      await onMove(trimmed);
      onClose();
    } catch {
      setError("Failed to move file.");
    } finally {
      setSubmitting(false);
    }
  }

  function handleClose() {
    setDestination("/");
    setError("");
    onClose();
  }

  return (
    <Modal open={open} onClose={handleClose} title="Move File">
      <p className="text-sm text-muted-foreground mb-4">
        Move <strong>{fileName}</strong> to a new location.
      </p>
      <form onSubmit={handleSubmit} className="space-y-4">
        {error && (
          <div className="rounded-lg bg-danger/10 px-3 py-2 text-sm text-danger">
            {error}
          </div>
        )}
        <Input
          label="Destination path"
          placeholder="/documents"
          value={destination}
          onChange={(e) => setDestination(e.target.value)}
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
            Move
          </Button>
        </div>
      </form>
    </Modal>
  );
}
