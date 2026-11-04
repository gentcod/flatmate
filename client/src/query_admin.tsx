import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { api } from './api'
import type { MetricsResponse, ReportListResponse } from './api'

export function useReports(enabled = true) {
  return useQuery({
    queryKey: ['admin', 'reports'],
    queryFn: () => api<ReportListResponse>('/api/v1/admin/reports'),
    enabled,
  })
}

export function useMetrics(enabled = true) {
  return useQuery({
    queryKey: ['admin', 'metrics'],
    queryFn: () => api<MetricsResponse>('/api/v1/admin/metrics'),
    enabled,
  })
}

export function useResolveReport() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ id, action }: { id: number; action: 'dismiss' | 'deactivate' }) =>
      api<{ ok: boolean }>(`/api/v1/admin/reports/${id}/resolve`, {
        method: 'POST',
        body: { action },
      }),
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: ['admin'] })
    },
  })
}
