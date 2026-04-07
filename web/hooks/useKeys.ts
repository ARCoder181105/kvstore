"use client";

import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { api } from "@/lib/api";
import type { SetKeyPayload } from "@/lib/types";

export function useKeys(pattern?: string) {
  return useQuery({
    queryKey: ["keys", pattern],
    queryFn: () => api.getKeys(pattern),
    refetchInterval: 2000,
    staleTime: 0,
  });
}

export function useSetKey() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({ key, payload }: { key: string; payload: SetKeyPayload }) =>
      api.setKey(key, payload),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["keys"] });
      qc.invalidateQueries({ queryKey: ["stats"] });
    },
    onError: (err: Error) => {
      console.error("[useSetKey] failed:", err.message);
    },
  });
}

export function useDeleteKey() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (key: string) => api.deleteKey(key),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["keys"] });
      qc.invalidateQueries({ queryKey: ["stats"] });
    },
    onError: (err: Error) => {
      console.error("[useDeleteKey] failed:", err.message);
    },
  });
}

export function useSetExpire() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({ key, seconds }: { key: string; seconds: number }) =>
      api.setExpire(key, seconds),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["keys"] }),
    onError: (err: Error) => {
      console.error("[useSetExpire] failed:", err.message);
    },
  });
}
