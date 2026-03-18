export interface User {
  id: string;
  username: string;
  email: string;
  role: "admin" | "user";
  is_active: boolean;
  created_at: string;
}

export interface MediaMeta {
  taken_at?: string;
  camera_make?: string;
  camera_model?: string;
  width?: number;
  height?: number;
  latitude?: number;
  longitude?: number;
}

export interface FileItem {
  id: string;
  virtual_path: string;
  original_name: string;
  size: number;
  mime_type: string;
  checksum: string;
  created_at: string;
  media?: MediaMeta;
}

export interface Directory {
  id: string;
  parent_path: string;
  name: string;
  created_at: string;
}

export interface FilesResponse {
  files: FileItem[];
  directories: Directory[];
}

export interface SetupCheckResponse {
  needs_setup: boolean;
}

export interface AuthResponse {
  user: User;
}

export interface LogoutResponse {
  message: string;
}

export type ViewMode = "table" | "grid" | "gallery";
export type FileCategory = "all" | "media" | "documents";

export interface DuplicateGroup {
  checksum: string;
  size: number;
  files: FileItem[];
}

export interface DuplicatesResponse {
  duplicates: DuplicateGroup[];
}
