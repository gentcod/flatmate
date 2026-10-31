import { Link, NavLink, Navigate, Outlet, useNavigate } from 'react-router-dom'
import type { ReactNode } from 'react'
import { isApiError } from './api'
import { useLogout, useSession } from './query'

export function errorMessage(e: unknown, fallback = 'Something went wrong. Try again.'): string {
  if (isApiError(e)) return e.message
  if (e instanceof Error) return e.message
  return fallback
}

export function Loading({ label }: { label: string }) {
  return <p className="py-10 text-center text-sm text-stone-600">{label}</p>
}

export function ErrorNote({ message }: { message: string }) {
  return (
    <p className="rounded-lg border border-rose-300 bg-rose-50 px-3 py-2 text-sm font-medium text-rose-900">
      {message}
    </p>
  )
}

export function Notice({ message }: { message: string }) {
  return (
    <p className="rounded-lg border border-emerald-300 bg-emerald-50 px-3 py-2 text-sm font-medium text-emerald-900">
      {message}
    </p>
  )
}

export function Card({ children }: { children: ReactNode }) {
  return <div className="rounded-xl border border-stone-200 bg-white p-5 shadow-sm">{children}</div>
}

export function PageTitle({ title, sub }: { title: string; sub?: string }) {
  return (
    <div className="mb-6">
      <h1 className="text-2xl font-bold tracking-tight text-stone-900">{title}</h1>
      {sub && <p className="mt-1 text-sm text-stone-600">{sub}</p>}
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
      <span className="mb-1 block text-sm font-semibold text-stone-800">{label}</span>
      {children}
      {hint && <span className="mt-1 block text-xs text-stone-600">{hint}</span>}
    </label>
  )
}

export const inputCls =
  'w-full rounded-lg border border-stone-300 bg-white px-3 py-2 text-stone-900 shadow-sm focus:border-emerald-700 focus:outline-none focus:ring-2 focus:ring-emerald-700/20'

export const primaryBtnCls =
  'rounded-lg bg-emerald-800 px-4 py-2 text-sm font-semibold text-white hover:bg-emerald-900 disabled:opacity-50'

export const ghostBtnCls =
  'rounded-lg border border-stone-300 bg-white px-4 py-2 text-sm font-semibold text-stone-800 hover:bg-stone-100 disabled:opacity-50'

export const dangerBtnCls =
  'rounded-lg bg-rose-700 px-4 py-2 text-sm font-semibold text-white hover:bg-rose-800 disabled:opacity-50'

/** Horizontal percentage bar, green/amber/rose by value. */
export function ScoreBar({ value, max = 100 }: { value: number; max?: number }) {
  const pct = Math.max(0, Math.min(100, Math.round((value / max) * 100)))
  const color = pct >= 70 ? 'bg-emerald-600' : pct >= 40 ? 'bg-amber-500' : 'bg-rose-500'
  return (
    <div className="h-2.5 w-full rounded-full bg-stone-200" role="img" aria-label={`${pct} percent`}>
      <div className={`h-2.5 rounded-full ${color}`} style={{ width: `${pct}%` }} />
    </div>
  )
}

export function RequireAuth({ children }: { children: ReactNode }) {
  const session = useSession()
  if (session.isPending) return <Loading label="Checking your session…" />
  if (session.isError || !session.data?.authed) return <Navigate to="/login" replace />
  return <>{children}</>
}

function navCls({ isActive }: { isActive: boolean }): string {
  return `rounded-md px-3 py-2 text-sm font-medium ${
    isActive ? 'bg-emerald-900 text-white' : 'text-stone-700 hover:bg-stone-200'
  }`
}

export function Layout() {
  const logout = useLogout()
  const navigate = useNavigate()
  const onLogout = () => {
    logout.mutate(undefined, { onSuccess: () => navigate('/login') })
  }
  return (
    <div className="min-h-screen">
      <header className="border-b border-stone-200 bg-white">
        <div className="mx-auto flex max-w-5xl flex-wrap items-center justify-between gap-3 px-4 py-3">
          <Link to="/matches" className="text-lg font-bold tracking-tight text-emerald-950">
            Flatmate
          </Link>
          <nav className="flex flex-wrap items-center gap-1">
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
            <NavLink to="/settings" className={navCls}>
              Settings
            </NavLink>
            <button
              type="button"
              onClick={onLogout}
              disabled={logout.isPending}
              className="rounded-md px-3 py-2 text-sm font-medium text-stone-700 hover:bg-stone-200 disabled:opacity-50"
            >
              Log out
            </button>
          </nav>
        </div>
      </header>
      <main className="mx-auto max-w-5xl px-4 py-8">
        <Outlet />
      </main>
    </div>
  )
}
