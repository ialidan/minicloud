import { useState } from "react";
import { useDebounce } from "@/lib/hooks/use-debounce";
import { useLocalStorage } from "@/lib/hooks/use-local-storage";
import { useCurrentPath } from "@/features/files/hooks/use-current-path";
import { useFiles } from "@/features/files/hooks/use-files";
import { useFileModals } from "@/features/files/hooks/use-file-modals";
import { useFileUpload } from "@/features/files/hooks/use-file-upload";
import type { ViewMode } from "@/lib/types";

export function useFilesPage() {
  const { path, category, navigateToDir, setCategory } = useCurrentPath();
  const [search, setSearch] = useState("");
  const debouncedSearch = useDebounce(search, 300);
  const { data, isLoading } = useFiles(path, debouncedSearch, category);
  const [viewMode, setViewMode] = useLocalStorage<ViewMode>("minicloud-view-mode", "table");

  const files = data?.files ?? [];
  const directories = data?.directories ?? [];
  const isEmpty = files.length === 0 && directories.length === 0;

  const modals = useFileModals(path);

  const uploadState = useFileUpload({
    files,
    path,
    deleteFileMutateAsync: modals.deleteFile.mutateAsync,
  });

  const showPath = debouncedSearch.length > 0;

  const viewProps = {
    files,
    directories,
    currentPath: path,
    onNavigateDir: navigateToDir,
    onDeleteFile: modals.setDeleteTarget,
    onMoveFile: modals.setMoveTarget,
    onDeleteDir: modals.setDeleteDirTarget,
    onPreviewFile: modals.setPreviewFile,
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
    files,
    directories,
    isEmpty,
    showPath,
    viewProps,
    // Modal state & handlers
    folderModalOpen: modals.folderModalOpen,
    setFolderModalOpen: modals.setFolderModalOpen,
    deleteTarget: modals.deleteTarget,
    setDeleteTarget: modals.setDeleteTarget,
    moveTarget: modals.moveTarget,
    setMoveTarget: modals.setMoveTarget,
    deleteDirTarget: modals.deleteDirTarget,
    setDeleteDirTarget: modals.setDeleteDirTarget,
    previewFile: modals.previewFile,
    setPreviewFile: modals.setPreviewFile,
    duplicatesOpen: modals.duplicatesOpen,
    setDuplicatesOpen: modals.setDuplicatesOpen,
    handleCreateFolder: modals.handleCreateFolder,
    handleDeleteFile: modals.handleDeleteFile,
    handleMoveFile: modals.handleMoveFile,
    handleDeleteDir: modals.handleDeleteDir,
    // Upload state & handlers
    fileInputRef: uploadState.fileInputRef,
    progress: uploadState.progress,
    isUploading: uploadState.isUploading,
    pendingUpload: uploadState.pendingUpload,
    setPendingUpload: uploadState.setPendingUpload,
    handleFileDrop: uploadState.handleFileDrop,
    handleUploadClick: uploadState.handleUploadClick,
    handleFileInputChange: uploadState.handleFileInputChange,
    handleDuplicateReplace: uploadState.handleDuplicateReplace,
    handleDuplicateKeepBoth: uploadState.handleDuplicateKeepBoth,
  };
}
