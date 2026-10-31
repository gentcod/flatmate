import { useState } from 'react'
import { Link, Navigate } from 'react-router-dom'
import {
  Card,
  ErrorNote,
  Loading,
  Notice,
  PageTitle,
  ScoreBar,
  errorMessage,
  ghostBtnCls,
  inputCls,
} from '../components'
import { isApiError } from '../api'
import type { MatchItem } from '../api'
import { useMatchAction, useMatches } from '../query'

function MatchCard({ m }: { m: MatchItem }) {
  const action = useMatchAction(m.id)
  const meta = [m.gender, m.occupation, m.area_text].filter((s) => s !== '').join(' · ')
  return (
    <Card>
      <div className="flex items-start justify-between gap-4">
        <div>
          <Link
            to={`/match/${m.id}`}
            className="text-lg font-bold tracking-tight text-emerald-950 hover:underline"
          >
            {m.first_name}
          </Link>
          {meta && <p className="mt-0.5 text-xs text-stone-600">{meta}</p>}
        </div>
        <span className="text-2xl font-bold tabular-nums text-stone-900">{m.total}</span>
      </div>
      <div className="mt-3">
        <ScoreBar value={m.total} />
      </div>
      {m.bio && <p className="mt-3 line-clamp-2 text-sm text-stone-700">{m.bio}</p>}
      {m.top_align && <p className="mt-2 text-xs font-medium text-stone-600">⊕ {m.top_align}</p>}
      <div className="mt-3 flex flex-wrap items-center gap-2">
        {m.saved && (
          <span className="rounded-full bg-amber-100 px-2.5 py-0.5 text-xs font-semibold text-amber-900">
            Saved
          </span>
        )}
        {m.mutual && (
          <span className="rounded-full bg-emerald-100 px-2.5 py-0.5 text-xs font-semibold text-emerald-900">
            Mutual interest
          </span>
        )}
        <span className="flex-1" />
        <Link
          to={`/match/${m.id}`}
          className="text-sm font-semibold text-emerald-800 hover:underline"
        >
          Why this score?
        </Link>
        <button
          type="button"
          disabled={action.isPending}
          onClick={() => action.mutate({ kind: 'dismiss' })}
          className={`${ghostBtnCls} px-3 py-1 text-xs`}
        >
          Dismiss
        </button>
      </div>
    </Card>
  )
}

export function MatchesPage() {
  const [input, setInput] = useState('')
  const [area, setArea] = useState('')
  const query = useMatches(area)
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
        <PageTitle title="Your matches" sub="Ranked by compatibility. Nothing here is AI-ranked — the formula is fixed and every score is explained." />
        <label className="mb-6 flex items-center gap-2 text-sm font-medium text-stone-700">
          <input
            type="checkbox"
            checked={savedOnly}
            onChange={(e) => setSavedOnly(e.target.checked)}
            className="h-4 w-4 accent-emerald-800"
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
        <p className="mb-4 text-sm text-stone-600">
          {query.data.nudge}{' '}
          <Link to="/profile" className="font-semibold text-emerald-800 hover:underline">
            Edit profile
          </Link>
        </p>
      )}
      <form
        className="mb-4 flex gap-2"
        onSubmit={(e) => {
          e.preventDefault()
          setArea(input.trim())
        }}
      >
        <input
          value={input}
          onChange={(e) => setInput(e.target.value)}
          placeholder="Filter by area, e.g. Lagos"
          maxLength={120}
          className={inputCls}
        />
        <button type="submit" className={ghostBtnCls}>
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
      </form>
      {query.data.empty || matches.length === 0 ? (
        <Card>
          <p className="text-sm text-stone-700">
            {query.data.empty
              ? 'No matches yet. Complete your profile and questionnaire, then check back as more people join.'
              : 'Nothing saved yet. Tap into a match and hit Save to pin them here.'}
          </p>
        </Card>
      ) : (
        <div className="grid gap-4 md:grid-cols-2">
          {matches.map((m) => (
            <MatchCard key={m.id} m={m} />
          ))}
        </div>
      )}
    </>
  )
}
