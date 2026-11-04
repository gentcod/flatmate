import {
  QueryClient,
  QueryClientProvider,
  useMutation,
  useQuery,
  useQueryClient,
} from '@tanstack/react-query'
/* eslint-disable react-refresh/only-export-components -- hooks + provider live here by design */
import type { ReactNode } from 'react'
import { api } from './api'
import type {
  Answer,
  Credentials,
  InterviewResponse,
  MatchDetailResponse,
  MatchListResponse,
  OnboardingInfo,
  Profile,
  ProfileResponse,
  QuestionnaireResponse,
  Session,
} from './api'

export const queryClient = new QueryClient({
  defaultOptions: { queries: { retry: 1, staleTime: 30_000 } },
})

export function AppProviders({ children }: { children: ReactNode }) {
  return <QueryClientProvider client={queryClient}>{children}</QueryClientProvider>
}

export function useSession() {
  return useQuery({
    queryKey: ['session'],
    queryFn: () => api<Session>('/api/v1/session'),
  })
}

export function useSignup() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (creds: Credentials) =>
      api<{ user_id: number }>('/api/v1/signup', { method: 'POST', body: creds }),
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: ['session'] })
    },
  })
}

export function useLogin() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (creds: Credentials) =>
      api<{ user_id: number }>('/api/v1/login', { method: 'POST', body: creds }),
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: ['session'] })
    },
  })
}

export function useLogout() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: () => api<{ ok: boolean }>('/api/v1/logout', { method: 'POST', body: {} }),
    onSuccess: () => {
      qc.clear()
      qc.setQueryData(['session'], { authed: false })
    },
  })
}

export function useProfile() {
  return useQuery({
    queryKey: ['profile'],
    queryFn: () => api<ProfileResponse>('/api/v1/profile'),
  })
}

export function useSaveProfile() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (profile: Profile) =>
      api<ProfileResponse>('/api/v1/profile', { method: 'PUT', body: profile }),
    onSuccess: (data) => {
      qc.setQueryData(['profile'], data)
      void qc.invalidateQueries({ queryKey: ['matches'] })
    },
  })
}

export function useQuestionnaire() {
  return useQuery({
    queryKey: ['questionnaire'],
    queryFn: () => api<QuestionnaireResponse>('/api/v1/questionnaire'),
  })
}

export function useSaveAnswers() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (answers: Answer[]) =>
      api<{ ok: boolean }>('/api/v1/questionnaire', {
        method: 'PUT',
        body: { answers },
      }),
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: ['questionnaire'] })
      void qc.invalidateQueries({ queryKey: ['matches'] })
      void qc.invalidateQueries({ queryKey: ['onboarding'] })
    },
  })
}

export function useMatches(area: string) {
  return useQuery({
    queryKey: ['matches', area],
    queryFn: () =>
      api<MatchListResponse>(
        `/api/v1/matches${area ? `?area=${encodeURIComponent(area)}` : ''}`,
      ),
  })
}

export function useMatchDetail(id: number, enabled = true) {
  return useQuery({
    queryKey: ['match', id],
    queryFn: () => api<MatchDetailResponse>(`/api/v1/matches/${id}`),
    enabled,
  })
}

export type MatchAction = 'save' | 'unsave' | 'dismiss' | 'interest' | 'block' | 'report'

export function useMatchAction(id: number) {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (input: { kind: MatchAction; reason?: string }) =>
      api<{ ok: boolean }>(`/api/v1/matches/${id}/${input.kind}`, {
        method: 'POST',
        body: input.kind === 'report' ? { reason: input.reason ?? '' } : {},
      }),
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: ['matches'] })
      void qc.invalidateQueries({ queryKey: ['match', id] })
    },
  })
}

export function useOnboarding() {
  return useQuery({
    queryKey: ['onboarding'],
    queryFn: () => api<OnboardingInfo>('/api/v1/onboarding'),
  })
}

export function useInterview() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (message: string) =>
      api<InterviewResponse>('/api/v1/onboarding', {
        method: 'POST',
        body: { message },
      }),
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: ['onboarding'] })
    },
  })
}

export function useDeactivate() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: () =>
      api<{ ok: boolean }>('/api/v1/account/deactivate', {
        method: 'POST',
        body: { confirm: true },
      }),
    onSuccess: () => {
      qc.clear()
      qc.setQueryData(['session'], { authed: false })
    },
  })
}
