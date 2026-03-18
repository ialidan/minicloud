import { useSearchParams } from "react-router-dom";
import { useCallback } from "react";
import type { FileCategory } from "@/lib/types";

export function useCurrentPath() {
  const [searchParams, setSearchParams] = useSearchParams();
  const path = searchParams.get("path") ?? "";
  const category = (searchParams.get("category") as FileCategory) ?? "all";

  const navigateToDir = useCallback(
    (dir: string) => {
      const next = new URLSearchParams(searchParams);
      if (dir) {
        next.set("path", dir);
      } else {
        next.delete("path");
      }
      // Clear search when navigating directories
      next.delete("search");
      setSearchParams(next, { replace: true });
    },
    [searchParams, setSearchParams],
  );

  const setCategory = useCallback(
    (cat: FileCategory) => {
      const next = new URLSearchParams(searchParams);
      if (cat && cat !== "all") {
        next.set("category", cat);
      } else {
        next.delete("category");
      }
      setSearchParams(next, { replace: true });
    },
    [searchParams, setSearchParams],
  );

  return { path, category, navigateToDir, setCategory };
}
