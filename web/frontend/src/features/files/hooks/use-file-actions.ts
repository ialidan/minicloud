import { useMutation, useQueryClient } from "@tanstack/react-query";
import { api } from "@/lib/api";

function useFilesMutation<TArgs>(
  mutationFn: (args: TArgs) => Promise<unknown>,
) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["files"] });
    },
  });
}

export function useDeleteFile() {
  return useFilesMutation((id: string) => api.del(`/files/${id}`));
}

export function useMoveFile() {
  return useFilesMutation(
    ({ id, destination }: { id: string; destination: string }) =>
      api.put(`/files/${id}/move`, { destination }),
  );
}

export function useDeleteDirectory() {
  return useFilesMutation((id: string) => api.del(`/directories/${id}`));
}
