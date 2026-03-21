import { useState } from "react";
import { useToast } from "@/lib/toast";
import { useCreateDirectory } from "@/features/files/hooks/use-create-directory";
import {
  useDeleteFile,
  useMoveFile,
  useDeleteDirectory,
} from "@/features/files/hooks/use-file-actions";
import type { FileItem, Directory } from "@/lib/types";

export function useFileModals(path: string) {
  const { toast } = useToast();

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

  return {
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
    handleCreateFolder,
    handleDeleteFile,
    handleMoveFile,
    handleDeleteDir,
    deleteFile,
  };
}
