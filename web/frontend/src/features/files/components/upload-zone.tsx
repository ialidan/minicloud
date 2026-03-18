import { useState, useRef, type ReactNode, type DragEvent } from "react";
import { Upload } from "lucide-react";

interface UploadZoneProps {
  onFileDrop: (file: File) => void;
  children: ReactNode;
}

export function UploadZone({ onFileDrop, children }: UploadZoneProps) {
  const [dragOver, setDragOver] = useState(false);
  const dragCounter = useRef(0);

  function handleDragEnter(e: DragEvent) {
    e.preventDefault();
    dragCounter.current++;
    if (e.dataTransfer.types.includes("Files")) {
      setDragOver(true);
    }
  }

  function handleDragLeave(e: DragEvent) {
    e.preventDefault();
    dragCounter.current--;
    if (dragCounter.current === 0) {
      setDragOver(false);
    }
  }

  function handleDragOver(e: DragEvent) {
    e.preventDefault();
  }

  function handleDrop(e: DragEvent) {
    e.preventDefault();
    dragCounter.current = 0;
    setDragOver(false);

    const file = e.dataTransfer.files[0];
    if (file) {
      onFileDrop(file);
    }
  }

  return (
    <div
      className="relative"
      onDragEnter={handleDragEnter}
      onDragLeave={handleDragLeave}
      onDragOver={handleDragOver}
      onDrop={handleDrop}
    >
      {children}
      {dragOver && (
        <div className="absolute inset-0 z-10 flex flex-col items-center justify-center rounded-lg border-2 border-dashed border-primary bg-primary/5 backdrop-blur-sm">
          <Upload className="h-8 w-8 text-primary" />
          <p className="mt-2 text-sm font-medium text-primary">
            Drop file to upload
          </p>
        </div>
      )}
    </div>
  );
}
