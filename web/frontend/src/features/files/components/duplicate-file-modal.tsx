import { Modal } from "@/components/ui/modal";
import { Button } from "@/components/ui/button";

interface DuplicateFileModalProps {
  open: boolean;
  onClose: () => void;
  fileName: string;
  onReplace: () => void;
  onKeepBoth: () => void;
}

export function DuplicateFileModal({
  open,
  onClose,
  fileName,
  onReplace,
  onKeepBoth,
}: DuplicateFileModalProps) {
  return (
    <Modal open={open} onClose={onClose} title="File Already Exists">
      <p className="text-sm text-muted-foreground mb-4">
        A file named <span className="font-medium text-foreground">"{fileName}"</span> already
        exists in this folder. What would you like to do?
      </p>
      <div className="flex justify-end gap-2">
        <Button variant="secondary" onClick={onClose}>
          Cancel
        </Button>
        <Button variant="destructive" onClick={onReplace}>
          Replace
        </Button>
        <Button onClick={onKeepBoth}>Keep Both</Button>
      </div>
    </Modal>
  );
}
