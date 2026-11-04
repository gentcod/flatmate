import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { api } from './api'
import type {
  AgreementResponse,
  ConversationListResponse,
  ExpenseInput,
  ExpenseLedgerResponse,
  ListingInput,
  ListingListResponse,
  ListingResponse,
  ListingReviewInput,
  ListingReviewListResponse,
  NLSearchResponse,
  NotificationListResponse,
  ReviewInput,
  ReviewListResponse,
  SearchInput,
  SearchListResponse,
  StayListResponse,
  ThreadResponse,
} from './api'

export function useListings(area: string, mine: boolean, maxRent: string) {
  return useQuery({
    queryKey: ['listings', area, mine, maxRent],
    queryFn: () => {
      const params = new URLSearchParams()
      if (area) params.set('area', area)
      if (maxRent) params.set('max_rent', maxRent)
      if (mine) params.set('mine', '1')
      const q = params.toString()
      return api<ListingListResponse>(`/api/v1/listings${q ? `?${q}` : ''}`)
    },
  })
}

export function useListing(id: number, enabled = true) {
  return useQuery({
    queryKey: ['listing', id],
    queryFn: () => api<ListingResponse>(`/api/v1/listings/${id}`),
    enabled,
  })
}

export function useCreateListing() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (input: ListingInput) =>
      api<{ id: number }>('/api/v1/listings', { method: 'POST', body: input }),
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: ['listings'] })
    },
  })
}

export function useUpdateListing(id: number) {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (input: ListingInput) =>
      api<ListingResponse>(`/api/v1/listings/${id}`, { method: 'PUT', body: input }),
    onSuccess: (data) => {
      qc.setQueryData(['listing', id], data)
      void qc.invalidateQueries({ queryKey: ['listings'] })
    },
  })
}

export function useDeleteListing() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (id: number) =>
      api<{ ok: boolean }>(`/api/v1/listings/${id}`, { method: 'DELETE' }),
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: ['listings'] })
    },
  })
}

export function useConversations() {
  return useQuery({
    queryKey: ['conversations'],
    queryFn: () => api<ConversationListResponse>('/api/v1/conversations'),
    refetchInterval: 15_000,
  })
}

export function useNotifications() {
  return useQuery({
    queryKey: ['notifications'],
    queryFn: () => api<NotificationListResponse>('/api/v1/notifications'),
    refetchInterval: 30_000,
  })
}

export function useMarkNotificationsRead() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: () => api<{ ok: boolean }>('/api/v1/notifications/read', { method: 'POST' }),
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: ['notifications'] })
    },
  })
}

export function useThread(peer: number, enabled = true) {
  return useQuery({
    queryKey: ['thread', peer],
    queryFn: () => api<ThreadResponse>(`/api/v1/conversations/${peer}`),
    enabled,
    refetchInterval: 10_000,
  })
}

export function useSendMessage(peer: number) {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (body: string) =>
      api<{ ok: boolean }>(`/api/v1/conversations/${peer}`, {
        method: 'POST',
        body: { body },
      }),
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: ['thread', peer] })
      void qc.invalidateQueries({ queryKey: ['conversations'] })
    },
  })
}

export function useReviews(userId: number) {
  return useQuery({
    queryKey: ['reviews', userId],
    queryFn: () => api<ReviewListResponse>(`/api/v1/users/${userId}/reviews`),
  })
}

export function useSaveReview(userId: number) {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (input: ReviewInput) =>
      api<{ ok: boolean }>(`/api/v1/users/${userId}/reviews`, {
        method: 'PUT',
        body: input,
      }),
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: ['reviews', userId] })
      void qc.invalidateQueries({ queryKey: ['match', userId] })
    },
  })
}

export function useLedger(peer: number, enabled = true) {
  return useQuery({
    queryKey: ['ledger', peer],
    queryFn: () => api<ExpenseLedgerResponse>(`/api/v1/expenses/${peer}`),
    enabled,
  })
}

export function useAddExpense(peer: number) {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (input: ExpenseInput) =>
      api<{ ok: boolean }>(`/api/v1/expenses/${peer}`, { method: 'POST', body: input }),
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: ['ledger', peer] })
    },
  })
}

export function useDeleteExpense(peer: number) {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (id: number) =>
      api<{ ok: boolean }>(`/api/v1/expenses/item/${id}`, { method: 'DELETE' }),
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: ['ledger', peer] })
    },
  })
}

export function useSearches() {
  return useQuery({
    queryKey: ['searches'],
    queryFn: () => api<SearchListResponse>('/api/v1/searches'),
  })
}

export function useSaveSearch() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (input: SearchInput) =>
      api<{ id: number }>('/api/v1/searches', { method: 'POST', body: input }),
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: ['searches'] })
    },
  })
}

export function useDeleteSearch() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (id: number) =>
      api<{ ok: boolean }>(`/api/v1/searches/${id}`, { method: 'DELETE' }),
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: ['searches'] })
    },
  })
}

export function useDraftAgreement(id: number) {
  return useMutation({
    mutationFn: () =>
      api<AgreementResponse>(`/api/v1/matches/${id}/agreement`, {
        method: 'POST',
        body: {},
      }),
  })
}

export function useNLS(query: string) {
  return useQuery({
    queryKey: ['nlsearch', query],
    queryFn: () =>
      api<NLSearchResponse>('/api/v1/matches/search', {
        method: 'POST',
        body: { query },
      }),
    enabled: query !== '',
  })
}

export function useStays() {
  return useQuery({
    queryKey: ['stays'],
    queryFn: () => api<StayListResponse>('/api/v1/stays'),
  })
}

export function useProposeStay() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (input: { peer_id: number; listing_id?: number }) =>
      api<{ id: number; status: string }>('/api/v1/stays', { method: 'POST', body: input }),
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: ['stays'] })
      void qc.invalidateQueries({ queryKey: ['match'] })
    },
  })
}

export function useStayTransition(id: number) {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (action: 'accept' | 'decline' | 'end') =>
      api<{ id: number; status: string }>(`/api/v1/stays/${id}/${action}`, {
        method: 'POST',
        body: {},
      }),
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: ['stays'] })
      void qc.invalidateQueries({ queryKey: ['match'] })
      void qc.invalidateQueries({ queryKey: ['listing'] })
      void qc.invalidateQueries({ queryKey: ['listings'] })
    },
  })
}

export function useListingReviews(id: number) {
  return useQuery({
    queryKey: ['listing-reviews', id],
    queryFn: () => api<ListingReviewListResponse>(`/api/v1/listings/${id}/reviews`),
  })
}

export function useSaveListingReview(id: number) {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (input: ListingReviewInput) =>
      api<{ ok: boolean }>(`/api/v1/listings/${id}/reviews`, { method: 'PUT', body: input }),
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: ['listing-reviews', id] })
    },
  })
}

export function useListingStatus(id: number) {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (status: string) =>
      api<{ id: number; status: string }>(`/api/v1/listings/${id}/status`, {
        method: 'POST',
        body: { status },
      }),
    onSuccess: (data) => {
      qc.setQueryData(['listing', id], (old: ListingResponse | undefined) =>
        old ? { ...old, listing: { ...old.listing, status: data.status } } : old,
      )
      void qc.invalidateQueries({ queryKey: ['listings'] })
    },
  })
}
