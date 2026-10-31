// Typed client for the Go JSON API (internal/web/api.go).
// Session rides the HttpOnly cookie; mutating requests carry the
// double-submit token in the X-CSRF-Token header.

export interface ApiError {
  status: number
  message: string
  setupRequired?: string
}

export function isApiError(e: unknown): e is ApiError {
  return (
    typeof e === 'object' &&
    e !== null &&
    'status' in e &&
    'message' in e
  )
}

function toApiError(status: number, message: string, setupRequired?: string): ApiError {
  return setupRequired ? { status, message, setupRequired } : { status, message }
}

interface ErrorBody {
  error?: string
  setup_required?: string
}

interface CsrfResponse {
  csrf_token: string
}

let csrfToken = ''

async function ensureCsrf(): Promise<string> {
  if (csrfToken) return csrfToken
  const res = await fetch('/api/v1/csrf', { credentials: 'include' })
  if (!res.ok) {
    throw toApiError(res.status, 'Could not reach the server.')
  }
  const data = (await res.json()) as CsrfResponse
  csrfToken = data.csrf_token
  return csrfToken
}

/** Warm the CSRF token at boot so the first form feels instant. */
export function bootCsrf(): void {
  void ensureCsrf().catch(() => undefined)
}

export interface ApiInit {
  method?: string
  body?: unknown
}

export async function api<T>(path: string, init?: ApiInit): Promise<T> {
  const method = init?.method ?? 'GET'
  const mutating = method !== 'GET' && method !== 'HEAD'
  const headers: Record<string, string> = {}
  let payload: string | undefined
  if (init?.body !== undefined) {
    headers['Content-Type'] = 'application/json'
    payload = JSON.stringify(init.body)
  }
  if (mutating) {
    headers['X-CSRF-Token'] = await ensureCsrf()
  }
  const run = () =>
    fetch(path, { method, headers, body: payload, credentials: 'include' })
  let res = await run()
  if (res.status === 403 && mutating) {
    // Token rotated or the server restarted: fetch once and retry.
    csrfToken = ''
    headers['X-CSRF-Token'] = await ensureCsrf()
    res = await run()
  }
  if (!res.ok) {
    let message = 'Something went wrong. Try again.'
    let setup: string | undefined
    try {
      const data = (await res.json()) as ErrorBody
      if (data.error) message = data.error
      setup = data.setup_required
    } catch {
      // Keep the default message when the body is not JSON.
    }
    throw toApiError(res.status, message, setup)
  }
  return (await res.json()) as T
}

// --- Payload shapes (mirror internal/web/api.go) ---

export interface Session {
  authed: boolean
  user_id?: number
}

export interface Credentials {
  email: string
  password: string
}

export interface Profile {
  first_name: string
  mode: string
  area_text: string
  lat: number | null
  lng: number | null
  relocating: boolean
  budget_min: number
  budget_max: number
  currency: string
  move_earliest: string
  move_latest: string
  duration_months: number
  gender: string
  gender_pref: string
  occupation: string
  bio: string
  contact_handle: string
  searching: boolean
}

export interface ProfileResponse {
  found: boolean
  profile: Profile
}

export interface QDim {
  dim: string
  label: string
  hint: string
  question: string
  options: string[]
  value: string
  hard: boolean
  weight: number
}

export interface QuestionnaireResponse {
  dims: QDim[]
  description: string
  notice?: string
}

export interface Answer {
  dimension: string
  value: string
  hard: boolean
  weight: number
}

export interface MatchItem {
  id: number
  first_name: string
  area_text: string
  gender: string
  occupation: string
  bio: string
  total: number
  top_align: string
  saved: boolean
  mutual: boolean
}

export interface MatchListResponse {
  matches: MatchItem[]
  empty: boolean
  searching: boolean
  area_filter: string
  nudge: string
}

export interface DimResult {
  name: string
  score: number
  weight: number
  aligned: boolean
  discuss: boolean
  note: string
}

export interface Breakdown {
  total: number
  practical: number
  lifestyle: number
  interpersonal: number
  dimensions: DimResult[]
  alignments: string[]
  differences: string[]
  discuss: string[]
}

export interface Bridge {
  dimension: string
  suggestion: string
}

export interface DetailPerson {
  id: number
  first_name: string
  area_text: string
  gender: string
  occupation: string
  bio: string
  mode_label: string
  budget_str: string
  move_window: string
  contact: string
  mutual: boolean
  saved: boolean
  my_interest: boolean
  ineligible: boolean
}

export interface MatchDetailResponse {
  m: DetailPerson
  ineligible?: boolean
  bd?: Breakdown
  top?: string[]
  bullets?: string[]
  enhanced?: boolean
  discuss_q?: string[]
  bridges?: Bridge[]
}

export interface Turn {
  role: string
  text: string
}

export interface OnboardingInfo {
  total: number
  active: boolean
  turns: Turn[]
  drafts_done: number
  text: string
  drafts?: Answer[]
}

export interface InterviewResponse {
  done?: boolean
  turns?: Turn[]
  drafts_done?: number
  total?: number
  seed?: Record<string, Answer>
  transcript?: string
  message?: string
  fallback?: boolean
  error?: string
}
