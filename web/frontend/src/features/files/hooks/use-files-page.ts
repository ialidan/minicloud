import { useState, useRef, useCallback } from "react";
import { useDebounce } from "@/lib/hooks/use-debounce";
import { useLocalStorage } from "@/lib/hooks/use-local-storage";
import { getUniqueName } from "@/lib/format";
import { useToast } from "@/lib/toast";
import { useCurrentPath } from "@/features/files/hooks/use-current-path";
import { useFiles } from "@/features/files/hooks/use-files";
import { useUpload } from "@/features/files/hooks/use-upload";
import { useCreateDirectory } from "@/features/files/hooks/use-create-directory";
import {
  useDeleteFile,
  useMoveFile,
  useDeleteDirectory,
} from "@/features/files/hooks/use-file-actions";
import type { FileItem, Directory, ViewMode } from "@/lib/types";

export function useFilesPage() {
  const { path, category, navigateToDir, setCategory } = useCurrentPath();
  const [search, setSearch] = useState("");
  const debouncedSearch = useDebounce(search, 300);
  const { data, isLoading } = useFiles(path, debouncedSearch, category);
  const { toast } = useToast();
  const [viewMode, setViewMode] = useLocalStorage<ViewMode>("minicloud-view-mode", "table");

  // Upload
  const fileInputRef = useRef<HTMLInputElement>(null);
  const { upload, progress, isUploading } = useUpload(
    useCallback(() => {
      toast("File uploaded successfully", "success");
    }, [toast]),
    useCallback((msg: string) => {
      toast(msg, "error");
    }, [toast]),
  );

  // Duplicate upload handling
  const [pendingUpload, setPendingUpload] = useState<{
    file: File;
    existingFileId: string;
  } | null>(null);

  // Create folder
  const [folderModalOpen, setFolderModalOpen] = useState(false);
  const createDir = useCreateDirectory();

  // Delete file
  const [deleteTarget, setDeleteTarget] = useState<FileItem | null>(null);
  const deleteFile = useDeleteFile();

  // Move file
  const [moveTarget, setMoveTarget] = useState<FileItem | null>(null);
  const moveFile = useMoveFile();

  // Delete directory
  const [deleteDirTarget, setDeleteDirTarget] = useState<Directory | null>(null);
  const deleteDir = useDeleteDirectory();

  // File preview
  const [previewFile, setPreviewFile] = useState<FileItem | null>(null);

  // Find duplicates
  const [duplicatesOpen, setDuplicatesOpen] = useState(false);

  const files = data?.files ?? [];
  const directories = data?.directories ?? [];
  const isEmpty = files.length === 0 && directories.length === 0;

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
    const file = e.target.files?.[0];
    if (file) {
      tryUpload(file);
      e.target.value = "";
    }
  }

  async function handleDuplicateReplace() {
    if (!pendingUpload) return;
    try {
      await deleteFile.mutateAsync(pendingUpload.existingFileId);
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

  async function handleCreateFolder(name: string) {
    await createDir.mutateAsync({ path: path, name });
    toast("Folder created", "success");
  }

  async function handleDeleteFile() {
    if (!deleteTarget) return;
    await deleteFile.mutateAsync(deleteTarget.id);
    toast("File deleted", "success");
  }

  async function handleMoveFile(destination: string) {
    if (!moveTarget) return;
    await moveFile.mutateAsync({ id: moveTarget.id, destination });
    toast("File moved", "success");
  }

  async function handleDeleteDir() {
    if (!deleteDirTarget) return;
    await deleteDir.mutateAsync(deleteDirTarget.id);
    toast("Folder deleted", "success");
  }

  const showPath = debouncedSearch.length > 0;

  const viewProps = {
    files,
    directories,
    currentPath: path,
    onNavigateDir: navigateToDir,
    onDeleteFile: setDeleteTarget,
    onMoveFile: setMoveTarget,
    onDeleteDir: setDeleteDirTarget,
    onPreviewFile: setPreviewFile,
    showPath,
  };

  return {
    path,
    category,
    navigateToDir,
    setCategory,
    search,
    setSearch,
    debouncedSearch,
    isLoading,
    viewMode,
    setViewMode,
    fileInputRef,
    progress,
    isUploading,
    pendingUpload,
    setPendingUpload,
    folderModalOpen,
    setFolderModalOpen,
    deleteTarget,
    setDeleteTarget,
    moveTarget,
    setMoveTarget,
    deleteDirTarget,
    setDeleteDirTarget,
    previewFile,
    setPreviewFile,
    duplicatesOpen,
    setDuplicatesOpen,
    files,
    directories,
    isEmpty,
    handleFileDrop,
    handleUploadClick,
    handleFileInputChange,
    handleDuplicateReplace,
    handleDuplicateKeepBoth,
    handleCreateFolder,
    handleDeleteFile,
    handleMoveFile,
    handleDeleteDir,
    showPath,
    viewProps,
  };
}
