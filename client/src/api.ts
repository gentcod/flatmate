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
  is_admin?: boolean
}

export interface AdminReport {
  id: number
  reporter: number
  reporter_name: string
  target: number
  target_name: string
  reason: string
  created_at: string
  resolved_at: string | null
  resolution: string
}

export interface ReportListResponse {
  reports: AdminReport[]
}

export interface MetricsResponse {
  users: number
  active_users: number
  open_reports: number
  listings: number
  reviews: number
  expenses: number
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
  contact_self_confirmed: boolean
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
  contact_self_confirmed: boolean
  mutual: boolean
  saved: boolean
  my_interest: boolean
  ineligible: boolean
  stay_id: number
  stay_status: string
  stay_proposed_by_me: boolean
  can_review: boolean
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
  reviews?: Review[]
  review_avg?: number
  review_count?: number
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

export interface ExpenseItem {
  id: number
  paid_by_me: boolean
  amount: number
  currency: string
  note: string
  split: string
  owed_by_me: number
  owed_to_me: number
  created_at: string
}

export interface Balance {
  currency: string
  amount: number
}

export interface ExpenseLedgerResponse {
  peer: { id: number; first_name: string }
  expenses: ExpenseItem[]
  balances: Balance[]
}

export interface ExpenseInput {
  amount: number
  currency: string
  note: string
  split: string
  payer: 'me' | 'them'
  payer_share: number
  peer_share: number
}

export interface SavedSearch {
  id: number
  name: string
  area: string
  max_rent: number
  created_at: string
  new_listings: number
}

export interface SearchListResponse {
  searches: SavedSearch[]
}

export interface SearchInput {
  name: string
  area: string
  max_rent: number
}

export interface AgreementSection {
  heading: string
  points: string[]
}

export interface AgreementResponse {
  sections: AgreementSection[]
  enhanced: boolean
}

export interface NLFilters {
  area: string
  max_rent: number
  smoking: string
  pets: string
  wfh_max: number
}

export interface NLSearchResponse {
  matches: MatchItem[]
  empty: boolean
  description: string
  enhanced: boolean
  filters: NLFilters
}

export interface Listing {
  id: number
  user_id: number
  owner_name: string
  title: string
  description: string
  area_text: string
  monthly_rent: number
  currency: string
  photo_url: string
  created_at: string
  status: string
  mine: boolean
}

export interface ListingListResponse {
  listings: Listing[]
}

export interface ListingResponse {
  listing: Listing
}

export interface ListingInput {
  title: string
  description: string
  area_text: string
  monthly_rent: number
  currency: string
  photo_url?: string
}

export interface Conversation {
  peer_id: number
  peer_name: string
  last_body: string
  last_at: string
  last_from_me: boolean
  unread_count: number
}

export interface ConversationListResponse {
  conversations: Conversation[]
}

export interface ChatMessage {
  id: number
  from_me: boolean
  body: string
  created_at: string
}

export interface ThreadResponse {
  peer: { id: number; first_name: string }
  messages: ChatMessage[]
}

export interface Review {
  author_id: number
  author_name: string
  rating: number
  body: string
  created_at: string
  verified_stay: boolean
}

export interface ReviewListResponse {
  reviews: Review[]
  avg: number
  count: number
}

export interface ReviewInput {
  rating: number
  body: string
}

export interface Stay {
  id: number
  peer_id: number
  peer_name: string
  status: string
  proposed_by_me: boolean
  listing_id: number
  listing_title: string
}

export interface StayListResponse {
  stays: Stay[]
}

export interface ListingReview {
  author_id: number
  author_name: string
  rating: number
  body: string
  kind: string
  created_at: string
}

export interface ListingReviewListResponse {
  stay_reviews: ListingReview[]
  visit_reviews: ListingReview[]
  trust_score: number
  stay_count: number
  visit_count: number
}

export interface ListingReviewInput {
  rating: number
  body: string
  kind: string
}
