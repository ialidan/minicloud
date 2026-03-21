import { useState, useRef, type ReactNode, type DragEvent } from "react";
import { Upload } from "lucide-react";

interface UploadZoneProps {
  onFileDrop: (file: File) => void;
  multiple?: boolean;
  children: ReactNode;
}

export function UploadZone({ onFileDrop, multiple, children }: UploadZoneProps) {
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

    const files = e.dataTransfer.files;
    if (multiple) {
      for (let i = 0; i < files.length; i++) {
        onFileDrop(files[i]!);
      }
    } else if (files[0]) {
      onFileDrop(files[0]);
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
            {multiple ? "Drop files to upload" : "Drop file to upload"}
          </p>
        </div>
      )}
    </div>
  );
}
