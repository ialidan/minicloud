import { useState, useRef, useCallback } from "react";
import { useToast } from "@/lib/toast";
import { getUniqueName } from "@/lib/format";
import { useUpload } from "@/features/files/hooks/use-upload";
import type { FileItem } from "@/lib/types";

interface UseFileUploadOptions {
  files: FileItem[];
  path: string;
  deleteFileMutateAsync: (id: string) => Promise<unknown>;
}

export function useFileUpload({ files, path, deleteFileMutateAsync }: UseFileUploadOptions) {
  const { toast } = useToast();
  const fileInputRef = useRef<HTMLInputElement>(null);
  const { upload, progress, isUploading } = useUpload(
    useCallback(() => {
      toast("File uploaded successfully", "success");
    }, [toast]),
    useCallback((msg: string) => {
      toast(msg, "error");
    }, [toast]),
  );

  const [pendingUpload, setPendingUpload] = useState<{
    file: File;
    existingFileId: string;
  } | null>(null);

  function tryUpload(file: File) {
    const existing = files.find(
      (f) => f.original_name.toLowerCase() === file.name.toLowerCase(),
    );
    if (existing) {
      setPendingUpload({ file, existingFileId: existing.id });
    } else {
      upload(file, path);
    }
  }

  function handleFileDrop(file: File) {
    tryUpload(file);
  }

  function handleUploadClick() {
    fileInputRef.current?.click();
  }

  function handleFileInputChange(e: React.ChangeEvent<HTMLInputElement>) {
    const selected = e.target.files;
    if (selected) {
      for (let i = 0; i < selected.length; i++) {
        tryUpload(selected[i]!);
      }
      e.target.value = "";
    }
  }

  async function handleDuplicateReplace() {
    if (!pendingUpload) return;
    try {
      await deleteFileMutateAsync(pendingUpload.existingFileId);
      upload(pendingUpload.file, path);
    } catch {
      toast("Failed to replace file", "error");
    }
    setPendingUpload(null);
  }

  function handleDuplicateKeepBoth() {
    if (!pendingUpload) return;
    const existingNames = files.map((f) => f.original_name);
    const newName = getUniqueName(pendingUpload.file.name, existingNames);
    upload(pendingUpload.file, path, newName);
    setPendingUpload(null);
  }

  return {
    fileInputRef,
    progress,
    isUploading,
    pendingUpload,
    setPendingUpload,
    handleFileDrop,
    handleUploadClick,
    handleFileInputChange,
    handleDuplicateReplace,
    handleDuplicateKeepBoth,
  };
}
