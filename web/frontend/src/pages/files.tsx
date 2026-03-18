import { useFilesPage } from "@/features/files/hooks/use-files-page";
import { Breadcrumb } from "@/features/files/components/breadcrumb";
import { CategoryTabs } from "@/features/files/components/category-tabs";
import { Toolbar } from "@/features/files/components/toolbar";
import { FileTable } from "@/features/files/components/file-table";
import { FileGrid } from "@/features/files/components/file-grid";
import { FileGallery } from "@/features/files/components/file-gallery";
import { FileTableSkeleton } from "@/features/files/components/file-table-skeleton";
import { EmptyState } from "@/features/files/components/empty-state";
import { UploadZone } from "@/features/files/components/upload-zone";
import { UploadProgress } from "@/features/files/components/upload-progress";
import { CreateFolderModal } from "@/features/files/components/create-folder-modal";
import { ConfirmModal } from "@/features/files/components/confirm-modal";
import { MoveModal } from "@/features/files/components/move-modal";
import { DuplicateFileModal } from "@/features/files/components/duplicate-file-modal";
import { FindDuplicatesModal } from "@/features/files/components/find-duplicates-modal";
import { FilePreviewModal } from "@/features/files/components/file-preview-modal";
import { MediaTimeline } from "@/features/files/components/media-timeline";

export function FilesPage() {
  const {
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
    viewProps,
  } = useFilesPage();

  return (
    <div className="space-y-4">
      <Breadcrumb path={path} onNavigate={navigateToDir} />
      <CategoryTabs active={category} onChange={setCategory} />
      <Toolbar
        search={search}
        onSearchChange={setSearch}
        onUpload={handleUploadClick}
        onNewFolder={() => setFolderModalOpen(true)}
        onFindDuplicates={() => setDuplicatesOpen(true)}
        viewMode={viewMode}
        onViewModeChange={setViewMode}
        hideViewToggle={category === "media"}
      />

      <UploadProgress progress={progress} isUploading={isUploading} />

      <UploadZone onFileDrop={handleFileDrop}>
        {isLoading ? (
          <FileTableSkeleton />
        ) : isEmpty ? (
          <EmptyState search={debouncedSearch} />
        ) : category === "media" && !debouncedSearch ? (
          <MediaTimeline
            files={files}
            onDeleteFile={setDeleteTarget}
            onMoveFile={setMoveTarget}
            onPreviewFile={setPreviewFile}
          />
        ) : viewMode === "grid" ? (
          <FileGrid {...viewProps} />
        ) : viewMode === "gallery" ? (
          <FileGallery {...viewProps} />
        ) : (
          <FileTable {...viewProps} />
        )}
      </UploadZone>

      <input
        ref={fileInputRef}
        type="file"
        className="hidden"
        onChange={handleFileInputChange}
      />

      <CreateFolderModal
        open={folderModalOpen}
        onClose={() => setFolderModalOpen(false)}
        onSubmit={handleCreateFolder}
      />

      <DuplicateFileModal
        open={pendingUpload !== null}
        onClose={() => setPendingUpload(null)}
        fileName={pendingUpload?.file.name ?? ""}
        onReplace={handleDuplicateReplace}
        onKeepBoth={handleDuplicateKeepBoth}
      />

      <ConfirmModal
        open={deleteTarget !== null}
        onClose={() => setDeleteTarget(null)}
        title="Delete File"
        description={`Are you sure you want to delete "${deleteTarget?.original_name ?? ""}"? This action cannot be undone.`}
        confirmLabel="Delete"
        variant="danger"
        onConfirm={handleDeleteFile}
      />

      {moveTarget && (
        <MoveModal
          open
          onClose={() => setMoveTarget(null)}
          fileName={moveTarget.original_name}
          onMove={handleMoveFile}
        />
      )}

      <ConfirmModal
        open={deleteDirTarget !== null}
        onClose={() => setDeleteDirTarget(null)}
        title="Delete Folder"
        description={`Are you sure you want to delete "${deleteDirTarget?.name ?? ""}"? All files inside will be deleted. This action cannot be undone.`}
        confirmLabel="Delete"
        variant="danger"
        onConfirm={handleDeleteDir}
      />

      <FindDuplicatesModal
        open={duplicatesOpen}
        onClose={() => setDuplicatesOpen(false)}
      />

      <FilePreviewModal
        open={previewFile !== null}
        onClose={() => setPreviewFile(null)}
        file={previewFile}
        files={files}
        onNavigate={setPreviewFile}
      />
    </div>
  );
}
