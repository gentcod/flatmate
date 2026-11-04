import { useState } from 'react'
import { Link, Navigate } from 'react-router-dom'
import {
  Card,
  Chip,
  ErrorNote,
  Icon,
  Loading,
  Notice,
  PageTitle,
  ScoreBar,
  Spinner,
  Toast,
  errorMessage,
  ghostBtnCls,
  inputCls,
  primaryBtnCls,
} from '../components'
import { isApiError } from '../api'
import type { MatchItem, Stay } from '../api'
import type { ToastData } from '../components'
import { useMatchAction, useMatches } from '../query'
import { useNLS, useStayTransition, useStays } from '../query_social'

function ProposalRow({ s }: { s: Stay }) {
  const transition = useStayTransition(s.id)
  const [toast, setToast] = useState<ToastData | null>(null)
  const busy = transition.isPending
  const done = (message: string) => ({
    onSuccess: () => setToast({ id: Date.now(), message, tone: 'ok' as const }),
    onError: (e: unknown) => setToast({ id: Date.now(), message: errorMessage(e), tone: 'err' as const }),
  })
  return (
    <div className="mt-2 flex flex-wrap items-center gap-2">
      <p className="text-sm text-sage">
        <Link
          to={`/match/${s.peer_id}`}
          className="font-semibold text-eucalyptus hover:underline"
        >
          {s.peer_name}
        </Link>{' '}
        says you live together{s.listing_title ? ` at ${s.listing_title}` : ''}.
      </p>
      <button
        type="button"
        disabled={busy}
        onClick={() => transition.mutate('accept', done('Stay confirmed.'))}
        className={primaryBtnCls}
      >
        {busy ? 'Confirming…' : 'Confirm'}
      </button>
      <button
        type="button"
        disabled={busy}
        onClick={() => transition.mutate('decline', done('Proposal declined.'))}
        className={ghostBtnCls}
      >
        Decline
      </button>
      {toast && <Toast toast={toast} onClose={() => setToast(null)} />}
    </div>
  )
}

function StayProposals() {
  const stays = useStays()
  const incoming = (stays.data?.stays ?? []).filter(
    (s) => s.status === 'proposed' && !s.proposed_by_me,
  )
  if (stays.isPending || stays.isError || incoming.length === 0) return null
  return (
    <div className="mb-6">
      <Card>
        <h2 className="font-display text-2xl tracking-tight text-ink">Stay proposals</h2>
        {incoming.map((s) => (
          <ProposalRow key={s.id} s={s} />
        ))}
      </Card>
    </div>
  )
}

function MatchCard({ m }: { m: MatchItem }) {
  const action = useMatchAction(m.id)
  const [toast, setToast] = useState<ToastData | null>(null)
  const meta = [m.gender, m.occupation, m.area_text].filter((s) => s !== '').join(' · ')
  const initial = (m.first_name || '?').slice(0, 1).toUpperCase()
  return (
    <article className="flex flex-col justify-between overflow-hidden rounded-2xl border border-sand bg-white shadow-card transition-shadow hover:shadow-lift">
      <div>
        <div className="flex items-center gap-4 bg-eucalyptus px-5 py-5">
          <span
            aria-hidden="true"
            className="flex h-14 w-14 shrink-0 items-center justify-center rounded-full bg-canopy font-display text-2xl text-canvas"
          >
            {initial}
          </span>
          <div className="min-w-0">
            <h2 className="truncate font-display text-2xl tracking-tight text-white">
              <Link to={`/match/${m.id}`} className="hover:underline">
                {m.first_name}
              </Link>
            </h2>
            {meta && <p className="mt-0.5 truncate text-[13px] text-[#aacec0]">{meta}</p>}
          </div>
          <span className="ml-auto inline-flex shrink-0 items-center rounded-full bg-canopy px-3 py-1.5 text-[13px] font-semibold text-white">
            {m.total}% match
          </span>
          <button
            type="button"
            aria-label={m.saved ? 'Unsave match' : 'Save match'}
            title={m.saved ? 'Unsave match' : 'Save match'}
            disabled={action.isPending}
            onClick={() =>
              action.mutate(
                { kind: m.saved ? 'unsave' : 'save' },
                {
                  onSuccess: () =>
                    setToast({
                      id: Date.now(),
                      message: m.saved ? 'Removed from saved.' : 'Saved.',
                      tone: 'ok',
                    }),
                  onError: (e) =>
                    setToast({ id: Date.now(), message: errorMessage(e), tone: 'err' }),
                },
              )
            }
            className={`flex h-9 w-9 shrink-0 cursor-pointer items-center justify-center rounded-full text-white hover:bg-white/25 disabled:cursor-not-allowed disabled:opacity-50 ${m.saved ? 'bg-white/30' : 'bg-white/15'}`}
          >
            {action.isPending ? (
              <Spinner />
            ) : (
              <Icon name="bookmark" filled={m.saved} />
            )}
          </button>
        </div>
        <div className="flex flex-col gap-4 p-5">
          {(m.mutual || m.saved) && (
            <div className="flex flex-wrap gap-1.5">
              {m.mutual && (
                <span className="inline-flex items-center rounded-full bg-terracotta px-2.5 py-0.5 text-[11px] font-semibold uppercase tracking-wide text-white">
                  Mutual interest
                </span>
              )}
              {m.saved && <Chip>Saved</Chip>}
            </div>
          )}
          {m.bio && (
            <p className="line-clamp-3 font-display text-[19px] italic leading-snug text-sage">
              “{m.bio}”
            </p>
          )}
          {m.top_align && (
            <div className="flex flex-col gap-1.5 rounded-xl bg-parchment/70 p-3">
              <div className="flex items-center justify-between text-[13px] text-ink">
                <span className="font-medium">Top alignment</span>
                <span className="font-semibold text-eucalyptus">{m.total}%</span>
              </div>
              <ScoreBar value={m.total} />
              <p className="text-[13px] text-sage">{m.top_align}</p>
            </div>
          )}
        </div>
      </div>
      <div className="flex items-center gap-2 p-5 pt-0">
        <button
          type="button"
          aria-label="Dismiss match"
          title="Dismiss"
          disabled={action.isPending}
          onClick={() =>
            action.mutate(
              { kind: 'dismiss' },
              {
                onSuccess: () =>
                  setToast({ id: Date.now(), message: 'Dismissed.', tone: 'ok' }),
                onError: (e) =>
                  setToast({ id: Date.now(), message: errorMessage(e), tone: 'err' }),
              },
            )
          }
          className="flex h-10 w-10 cursor-pointer items-center justify-center rounded-xl text-fog hover:bg-errtint hover:text-err disabled:cursor-not-allowed disabled:opacity-50"
        >
          {action.isPending ? <Spinner /> : <Icon name="close" />}
        </button>
        <Link to={`/match/${m.id}`} className={`${primaryBtnCls} flex-1 py-2.5 text-center text-[15px]`}>
          View Full Compatibility
        </Link>
      </div>
      {toast && <Toast toast={toast} onClose={() => setToast(null)} />}
    </article>
  )
}

export function MatchesPage() {
  const [input, setInput] = useState('')
  const [area, setArea] = useState('')
  const [nlInput, setNlInput] = useState('')
  const [nlQuery, setNlQuery] = useState('')
  const query = useMatches(area)
  const nl = useNLS(nlQuery)
  const [savedOnly, setSavedOnly] = useState(false)

  if (query.isPending) return <Loading label="Finding your matches…" />
  if (query.isError) {
    if (isApiError(query.error) && query.error.setupRequired === 'profile') {
      return <Navigate to="/profile" replace />
    }
    if (isApiError(query.error) && query.error.setupRequired === 'questionnaire') {
      return <Navigate to="/questionnaire" replace />
    }
    return <ErrorNote message={errorMessage(query.error)} />
  }

  const matches = savedOnly ? query.data.matches.filter((m) => m.saved) : query.data.matches

  return (
    <>
      <div className="flex flex-wrap items-end justify-between gap-3">
        <PageTitle title="Explore matches" sub="Ranked by compatibility from a fixed formula — every score is explained, nothing is AI-ranked." />
        <label className="mb-6 flex items-center gap-2 text-sm font-medium text-sage">
          <input
            type="checkbox"
            checked={savedOnly}
            onChange={(e) => setSavedOnly(e.target.checked)}
            className="h-4 w-4 accent-eucalyptus"
          />
          Saved only
        </label>
      </div>
      {!query.data.searching && (
        <div className="mb-4">
          <Notice message="You're hidden from matches — nobody new can find you until you switch back on in Settings." />
        </div>
      )}
      {query.data.nudge && (
        <p className="mb-4 text-sm text-sage">
          {query.data.nudge}{' '}
          <Link to="/profile" className="font-semibold text-eucalyptus hover:underline">
            Edit profile
          </Link>
        </p>
      )}
      <StayProposals />
      <div className="mb-6">
        <Card>
          <form
            className="flex items-center gap-3 rounded-2xl bg-parchment px-4 py-3"
            onSubmit={(e) => {
              e.preventDefault()
              setNlQuery(nlInput.trim())
            }}
          >
            <Icon name="travel_explore" cls="shrink-0 text-[28px] text-eucalyptus" />
            <div className="flex flex-1 flex-col justify-center">
              <span className="mb-0.5 text-[12px] font-semibold uppercase tracking-wider text-fog">
                Describe your ideal flatmate
              </span>
              <input
                value={nlInput}
                onChange={(e) => setNlInput(e.target.value)}
                placeholder="quiet non-smoker under 300k in Yaba…"
                maxLength={200}
                aria-label="Natural language search criteria"
                className="w-full bg-transparent text-[17px] font-semibold text-ink placeholder:font-normal placeholder:text-fog focus:outline-none"
              />
            </div>
            <button type="submit" className={primaryBtnCls}>
              Search
            </button>
            {nlQuery && (
              <button
                type="button"
                className={ghostBtnCls}
                onClick={() => {
                  setNlInput('')
                  setNlQuery('')
                }}
              >
                Clear
              </button>
            )}
          </form>
          {nlQuery && nl.data && (
            <p className="pt-3 text-sm text-sage">
              <span className="mr-1 font-medium text-ink">Interpreted intent:</span>
              {nl.data.description ? `${nl.data.description}. ` : ''}
              {nl.data.enhanced ? 'Parsed with AI.' : 'Keyword match (AI unavailable).'}
            </p>
          )}
        </Card>
      </div>
      {nlQuery ? (
        nl.isPending ? (
          <Loading label="Searching…" />
        ) : nl.isError ? (
          <ErrorNote message={errorMessage(nl.error)} />
        ) : (
          <>
            <div className="mb-4 flex flex-col justify-between gap-2 px-1 sm:flex-row sm:items-baseline">
              <div className="flex items-center gap-2">
                <span className="h-2.5 w-2.5 rounded-full bg-eucalyptus" />
                <h2 className="text-[17px] font-semibold tracking-tight text-ink">
                  {nl.data.matches.length} matching candidates
                </h2>
              </div>
              <p className="text-[13px] text-sage">Sorted by overall compatibility</p>
            </div>
            {nl.data.empty ? (
              <Card>
                <p className="text-[15px] text-sage">
                  Nothing matches that description. Try fewer constraints.
                </p>
              </Card>
            ) : (
              <div className="grid grid-cols-1 gap-6 md:grid-cols-2 lg:grid-cols-3">
                {nl.data.matches.map((m) => (
                  <MatchCard key={m.id} m={m} />
                ))}
              </div>
            )}
          </>
        )
      ) : (
        <>
          <div className="mb-6">
            <Card>
              <form
                className="flex flex-col gap-3 sm:flex-row"
                onSubmit={(e) => {
                  e.preventDefault()
                  setArea(input.trim())
                }}
              >
                <div className="flex flex-1 flex-col gap-1">
                  <label htmlFor="area-filter" className="text-[13px] font-medium text-sage">
                    Neighborhood
                  </label>
                  <input
                    id="area-filter"
                    value={input}
                    onChange={(e) => setInput(e.target.value)}
                    placeholder="Filter by area, e.g. Lagos"
                    maxLength={120}
                    className={inputCls}
                  />
                </div>
                <div className="flex items-end gap-2">
                  <button type="submit" className={primaryBtnCls}>
                    Filter
                  </button>
                  {area && (
                    <button
                      type="button"
                      className={ghostBtnCls}
                      onClick={() => {
                        setInput('')
                        setArea('')
                      }}
                    >
                      Clear
                    </button>
                  )}
                </div>
              </form>
            </Card>
          </div>
          {matches.length > 0 && (
            <div className="mb-4 flex flex-col justify-between gap-2 px-1 sm:flex-row sm:items-baseline">
              <div className="flex items-center gap-2">
                <span className="h-2.5 w-2.5 rounded-full bg-eucalyptus" />
                <h2 className="text-[17px] font-semibold tracking-tight text-ink">
                  Showing {matches.length} compatible candidates{area ? ` in ${area}` : ''}
                </h2>
              </div>
              <p className="text-[13px] text-sage">Sorted by overall compatibility</p>
            </div>
          )}
          {query.data.empty || matches.length === 0 ? (
            <Card>
              <p className="text-[15px] text-sage">
                {query.data.empty
                  ? 'No matches yet. Complete your profile and questionnaire, then check back as more people join.'
                  : 'Nothing saved yet. Tap into a match and hit Save to pin them here.'}
              </p>
            </Card>
          ) : (
            <div className="grid grid-cols-1 gap-6 md:grid-cols-2 lg:grid-cols-3">
              {matches.map((m) => (
                <MatchCard key={m.id} m={m} />
              ))}
            </div>
          )}
        </>
      )}
    </>
  )
}
