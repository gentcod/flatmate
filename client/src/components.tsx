/* eslint-disable react-refresh/only-export-components -- shared UI kit: components + style helpers by design */
import { useEffect, useState } from 'react'
import { Link, NavLink, Navigate, Outlet, useLocation, useNavigate } from 'react-router-dom'
import type { ReactNode } from 'react'
import { isApiError } from './api'
import { useLogout, useProfile, useSession } from './query'
import { useConversations, useMarkNotificationsRead, useNotifications } from './query_social'
import { OfflineBanner, UpdateBanner } from './pwa'

export function errorMessage(e: unknown, fallback = 'Something went wrong. Try again.'): string {
  if (isApiError(e)) return e.message
  if (e instanceof Error) return e.message
  return fallback
}

export function Loading({ label }: { label: string }) {
  return <p className="py-10 text-center text-sm text-fog">{label}</p>
}

export function ErrorNote({ message }: { message: string }) {
  return (
    <p className="rounded-xl border border-errline bg-errtint px-3 py-2 text-sm font-medium text-errink">
      {message}
    </p>
  )
}

export function Notice({ message, tone = 'ok' }: { message: string; tone?: 'ok' | 'warn' }) {
  return (
    <p
      className={`rounded-xl border px-3 py-2 text-sm font-medium ${tone === 'warn' ? 'border-errline bg-errtint text-errink' : 'border-shieldline bg-mist text-eucalyptus'}`}
    >
      {message}
    </p>
  )
}

export function Card({ children }: { children: ReactNode }) {
  return <div className="rounded-2xl border border-sand bg-white p-5 shadow-card">{children}</div>
}

export function PageTitle({ title, sub }: { title: string; sub?: string }) {
  return (
    <div className="mb-6">
      <h1 className="font-display text-3xl tracking-tight text-ink">{title}</h1>
      {sub && <p className="mt-1 text-[15px] text-sage">{sub}</p>}
    </div>
  )
}

export function Field({
  label,
  hint,
  children,
}: {
  label: string
  hint?: string
  children: ReactNode
}) {
  return (
    <label className="block">
      <span className="mb-1 block text-sm font-semibold text-ink">{label}</span>
      {children}
      {hint && <span className="mt-1 block text-xs text-fog">{hint}</span>}
    </label>
  )
}

export const inputCls =
  'w-full rounded-xl border border-sand bg-white px-4 py-3 text-ink shadow-sm placeholder:text-fog focus:border-eucalyptus focus:outline-none focus:ring-[1.5px] focus:ring-eucalyptus'

export const primaryBtnCls =
  'rounded-xl bg-eucalyptus px-4 py-2 text-sm font-semibold text-[#fbf9f5] shadow-sm hover:bg-canopy disabled:opacity-50 cursor-pointer disabled:cursor-not-allowed'

export const ghostBtnCls =
  'rounded-xl border border-sand bg-transparent px-4 py-2 text-sm font-semibold text-ink hover:bg-parchment disabled:opacity-50 cursor-pointer disabled:cursor-not-allowed'

export const dangerBtnCls =
  'rounded-xl bg-err px-4 py-2 text-sm font-semibold text-white hover:bg-errink disabled:opacity-50 cursor-pointer disabled:cursor-not-allowed'

/** Inline spinner for buttons; inherits the button's text color. */
export function Spinner({ cls = 'h-4 w-4' }: { cls?: string }) {
  return (
    <span
      aria-hidden="true"
      className={`inline-block animate-spin rounded-full border-2 border-current border-t-transparent align-[-2px] ${cls}`}
    />
  )
}

/** Blocking overlay for slow processes (AI drafts, auth). Render while the mutation is pending. */
export function LoadingModal({ label }: { label: string }) {
  return (
    <div
      className="fixed inset-0 z-50 flex items-center justify-center bg-ink/40 p-4"
      role="dialog"
      aria-modal="true"
      aria-label={label}
    >
      <div className="flex items-center gap-3 rounded-2xl bg-white px-6 py-5 shadow-lift">
        <span className="text-eucalyptus">
          <Spinner cls="h-6 w-6" />
        </span>
        <p className="text-[15px] font-semibold text-ink">{label}</p>
      </div>
    </div>
  )
}

export type ToastTone = 'ok' | 'err'

export interface ToastData {
  id: number
  message: string
  tone: ToastTone
}

/** Bottom toast verdict after an action completes; auto-dismisses. Parent owns the state. */
export function Toast({ toast, onClose }: { toast: ToastData; onClose: () => void }) {
  useEffect(() => {
    const t = window.setTimeout(onClose, 4000)
    return () => window.clearTimeout(t)
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [toast.id])
  return (
    <div className="fixed bottom-6 left-1/2 z-50 -translate-x-1/2" role="status">
      <div
        className={`flex items-center gap-3 rounded-xl px-4 py-3 text-sm font-semibold text-white shadow-lift ${
          toast.tone === 'ok' ? 'bg-eucalyptus' : 'bg-err'
        }`}
      >
        <span>{toast.message}</span>
        <button
          type="button"
          onClick={onClose}
          aria-label="Dismiss"
          className="cursor-pointer rounded p-0.5 hover:bg-white/20"
        >
          <Icon name="close" cls="text-[16px]" />
        </button>
      </div>
    </div>
  )
}

/** Driver bar: 6px sand rail, eucalyptus fill, terracotta for pivotal traits. */
export function ScoreBar({
  value,
  max = 100,
  tone = 'standard',
}: {
  value: number
  max?: number
  tone?: 'standard' | 'pivotal'
}) {
  const pct = Math.max(0, Math.min(100, Math.round((value / max) * 100)))
  const color = tone === 'pivotal' ? 'bg-terracotta' : 'bg-canopy'
  return (
    <div className="h-1.5 w-full rounded-full bg-sand" role="img" aria-label={`${pct} percent`}>
      <div className={`h-1.5 rounded-full ${color}`} style={{ width: `${pct}%` }} />
    </div>
  )
}

export function RequireAuth({ children }: { children: ReactNode }) {
  const session = useSession()
  if (session.isPending) return <Loading label="Checking your session…" />
  if (session.isError || !session.data?.authed) return <Navigate to="/login" replace />
  return <>{children}</>
}

/** Material Symbols icon by name, e.g. <Icon name="mop" />. */
export function Icon({
  name,
  cls = 'text-[20px]',
  filled = false,
}: {
  name: string
  cls?: string
  filled?: boolean
}) {
  return (
    <span
      aria-hidden="true"
      className={`material-symbols-outlined ${cls}`}
      style={filled ? { fontVariationSettings: "'FILL' 1" } : undefined}
    >
      {name}
    </span>
  )
}

/** Trust badge: neutral documentation chip, never alarmist. Only states the backend supports. */
export function TrustBadge({ selfConfirmed }: { selfConfirmed: boolean }) {
  return (
    <span className="inline-flex items-center gap-1.5 rounded-full border border-sand bg-parchment px-2.5 py-0.5 text-[11px] font-semibold uppercase tracking-wide text-sage">
      <span aria-hidden="true" className="h-1.5 w-1.5 rounded-full border border-fog" />
      {selfConfirmed ? 'Self-confirmed' : 'Unverified'}
    </span>
  )
}

/** Small metadata pill for habits, logistics, and filter tags. */
export function Chip({ children }: { children: ReactNode }) {
  return (
    <span className="inline-flex items-center gap-1.5 rounded-full bg-parchment px-3 py-1 text-[13px] text-ink">
      {children}
    </span>
  )
}

function navCls({ isActive }: { isActive: boolean }): string {
  return `rounded-md px-3 py-2 text-sm font-medium ${
    isActive ? 'bg-eucalyptus text-[#fbf9f5]' : 'text-sage hover:bg-parchment hover:text-ink'
  }`
}

export function Layout() {
  const logout = useLogout()
  const session = useSession()
  const navigate = useNavigate()
  const location = useLocation()
  const chats = useConversations()
  const alerts = useNotifications()
  const markRead = useMarkNotificationsRead()
  const [menuOpen, setMenuOpen] = useState(false)
  const [userOpen, setUserOpen] = useState(false)
  const [alertsOpen, setAlertsOpen] = useState(false)
  const [toast, setToast] = useState<ToastData | null>(null)
  const profile = useProfile()
  const displayName = profile.data?.profile.first_name.trim() || 'Flatmate'
  // Dead by design: settings + logout live in the account menu; the old
  // nav items below are kept for reference until the file is reformatted.
  const showLegacyNav = false
  useEffect(() => {
    setMenuOpen(false)
    setUserOpen(false)
    setAlertsOpen(false)
  }, [location.pathname])
  const unreadTotal =
    chats.data?.conversations.reduce((n, c) => n + c.unread_count, 0) ?? 0
  const onLogout = () => {
    logout.mutate(undefined, {
      onSuccess: () => navigate('/'),
      onError: (e) => setToast({ id: Date.now(), message: errorMessage(e), tone: 'err' }),
    })
  }
  return (
    <div className="min-h-screen">
      <OfflineBanner />
      <UpdateBanner />
      <header className="sticky top-0 z-40 border-b border-sand bg-canvas/90 backdrop-blur-xl">
        <div className="mx-auto flex min-h-20 max-w-7xl flex-wrap items-center justify-between gap-3 px-5 py-2 lg:px-12">
          <Link to="/matches" className="flex flex-col leading-none">
            <span className="font-display text-2xl tracking-tight text-ink">Flatmate</span>
            <span className="mt-0.5 text-[11px] font-semibold uppercase tracking-widest text-fog">
              Shared Living
            </span>
          </Link>
          <div className="relative flex items-center gap-1 md:order-last">
            <div className="relative">
              <button
                type="button"
                onClick={() => setAlertsOpen((v) => !v)}
                aria-expanded={alertsOpen}
                aria-haspopup="menu"
                aria-label="Alerts"
                className="relative flex h-10 w-10 cursor-pointer items-center justify-center rounded-full text-sage hover:bg-parchment hover:text-ink"
              >
                <Icon name="notifications" cls="text-[22px]" />
                {(alerts.data?.unread_count ?? 0) > 0 && (
                  <span className="absolute right-0.5 top-0.5 inline-flex min-w-5 items-center justify-center rounded-full bg-terracotta px-1.5 py-0.5 text-[11px] font-bold leading-none text-white">
                    {alerts.data?.unread_count}
                  </span>
                )}
              </button>
              {alertsOpen && (
                <div
                  role="menu"
                  onKeyDown={(e) => {
                    if (e.key === 'Escape') setAlertsOpen(false)
                  }}
                  className="absolute right-0 top-full z-50 mt-2 w-80 overflow-hidden rounded-xl border border-sand bg-white py-1 shadow-lift"
                >
                  <div className="flex items-center justify-between px-4 pb-1 pt-2">
                    <p className="text-[13px] font-semibold text-ink">Alerts</p>
                    <button
                      type="button"
                      disabled={markRead.isPending || (alerts.data?.unread_count ?? 0) === 0}
                      onClick={() => markRead.mutate()}
                      className="cursor-pointer text-[13px] font-medium text-eucalyptus hover:underline disabled:cursor-not-allowed disabled:opacity-50"
                    >
                      {markRead.isPending ? 'Marking…' : 'Mark all read'}
                    </button>
                  </div>
                  {(alerts.data?.notifications.length ?? 0) === 0 ? (
                    <p className="px-4 py-3 text-sm text-sage">No alerts yet.</p>
                  ) : (
                    alerts.data?.notifications.slice(0, 8).map((n) => (
                      <Link
                        key={n.id}
                        role="menuitem"
                        to={n.kind === 'listing_review' ? `/listing/${n.ref_id}` : `/match/${n.actor_id}`}
                        onClick={() => setAlertsOpen(false)}
                        className={`block px-4 py-2 text-sm hover:bg-parchment ${n.read ? 'text-sage' : 'font-medium text-ink'}`}
                      >
                        {!n.read && (
                          <span
                            aria-hidden="true"
                            className="mr-1.5 inline-block h-1.5 w-1.5 rounded-full bg-terracotta"
                          />
                        )}
                        {n.body}
                      </Link>
                    ))
                  )}
                </div>
              )}
            </div>
            <button
              type="button"
              onClick={() => setUserOpen((v) => !v)}
              aria-expanded={userOpen}
              aria-haspopup="menu"
              aria-label="Account menu"
              className="flex cursor-pointer items-center gap-2 rounded-full py-1 pl-1 pr-2 hover:bg-parchment"
            >
              <span
                aria-hidden="true"
                className="flex h-9 w-9 items-center justify-center rounded-full bg-canopy text-[15px] font-semibold text-white"
              >
                {displayName.slice(0, 1).toUpperCase()}
              </span>
              <span className="hidden max-w-28 truncate text-sm font-semibold text-ink sm:block">
                {displayName}
              </span>
              <Icon name="expand_more" cls="text-[18px] text-sage" />
            </button>
            {userOpen && (
              <div
                role="menu"
                onKeyDown={(e) => {
                  if (e.key === 'Escape') setUserOpen(false)
                }}
                className="absolute right-0 top-full z-50 mt-2 w-52 overflow-hidden rounded-xl border border-sand bg-white py-1 shadow-lift"
              >
                <p className="truncate px-4 pb-1 pt-2 text-[13px] font-semibold text-ink">
                  {displayName}
                </p>
                <Link
                  role="menuitem"
                  to="/settings"
                  onClick={() => setUserOpen(false)}
                  className="block px-4 py-2 text-sm text-sage hover:bg-parchment hover:text-ink"
                >
                  Settings
                </Link>
                <button
                  role="menuitem"
                  type="button"
                  disabled={logout.isPending}
                  onClick={() => {
                    setUserOpen(false)
                    onLogout()
                  }}
                  className="block w-full cursor-pointer px-4 py-2 text-left text-sm text-sage hover:bg-parchment hover:text-ink disabled:cursor-not-allowed disabled:opacity-50"
                >
                  {logout.isPending ? 'Logging out…' : 'Log out'}
                </button>
              </div>
            )}
          </div>
          <button
            type="button"
            aria-label={menuOpen ? 'Close menu' : 'Open menu'}
            aria-expanded={menuOpen}
            onClick={() => setMenuOpen((v) => !v)}
            className="flex h-10 w-10 cursor-pointer items-center justify-center rounded-md text-sage hover:bg-parchment hover:text-ink md:hidden"
          >
            <Icon name={menuOpen ? 'close' : 'menu'} />
          </button>
          <nav
            className={`${menuOpen ? 'flex' : 'hidden'} order-last w-full flex-col items-stretch gap-1 pb-2 md:flex md:order-none md:w-auto md:flex-row md:items-center md:pb-0`}
          >
            <NavLink to="/matches" className={navCls}>
              Matches
            </NavLink>
            <NavLink to="/profile" className={navCls}>
              Profile
            </NavLink>
            <NavLink to="/questionnaire" className={navCls}>
              Questions
            </NavLink>
            <NavLink to="/onboarding" className={navCls}>
              Chat setup
            </NavLink>
            <NavLink to="/listings" className={navCls}>
              Listings
            </NavLink>
            <NavLink to="/chat" className={navCls}>
              Chat
              {unreadTotal > 0 && (
                <span className="ml-1.5 inline-flex min-w-5 items-center justify-center rounded-full bg-terracotta px-1.5 py-0.5 text-[11px] font-bold leading-none text-white">
                  {unreadTotal}
                </span>
              )}
            </NavLink>
            {session.data?.is_admin && (
              <NavLink to="/admin" className={navCls}>
                Admin
              </NavLink>
            )}
            {/* Settings + logout moved to the account menu above. */}
            {showLegacyNav && (
              <>
            <NavLink to="/settings" className={navCls}>
              Settings
            </NavLink>
            <button
              type="button"
              onClick={onLogout}
              disabled={logout.isPending}
              className="cursor-pointer rounded-md px-3 py-2 text-sm font-medium text-sage hover:bg-parchment hover:text-ink disabled:cursor-not-allowed disabled:opacity-50"
            >
              Log out
            </button>
              </>
            )}
          </nav>
        </div>
      </header>
      <main className="mx-auto max-w-7xl px-5 py-10 lg:px-12">
        <Outlet />
      </main>
      <footer className="border-t border-sand py-6 text-center text-[13px] text-fog">
        Flatmate demo — profiles are unverified. Scores advise; you decide.
      </footer>
      {toast && <Toast toast={toast} onClose={() => setToast(null)} />}
    </div>
  )
}
