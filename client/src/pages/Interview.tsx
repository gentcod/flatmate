import { useState } from 'react'
import type { FormEvent } from 'react'
import { Link, useNavigate } from 'react-router-dom'
import {
  Card,
  ErrorNote,
  Loading,
  Notice,
  PageTitle,
  ScoreBar,
  errorMessage,
  inputCls,
  primaryBtnCls,
} from '../components'
import type { Turn } from '../api'
import { useInterview, useOnboarding } from '../query'

function Turns({ turns }: { turns: Turn[] }) {
  return (
    <div className="flex flex-col gap-3">
      {turns.map((t, i) => (
        <div key={i} className={t.role === 'user' ? 'flex justify-end' : 'flex justify-start'}>
          <p
            className={`max-w-[80%] rounded-xl px-4 py-2 text-sm ${
              t.role === 'user'
                ? 'bg-emerald-800 text-white'
                : 'border border-stone-200 bg-white text-stone-900'
            }`}
          >
            {t.text}
          </p>
        </div>
      ))}
    </div>
  )
}

export function InterviewPage() {
  const info = useOnboarding()
  const post = useInterview()
  const navigate = useNavigate()
  const [draft, setDraft] = useState('')
  const [override, setOverride] = useState<Turn[] | null>(null)
  const [progress, setProgress] = useState<{ done: number; total: number } | null>(null)
  const [note, setNote] = useState('')

  if (info.isPending) return <Loading label="Loading chat setup…" />
  if (info.isError) return <ErrorNote message={errorMessage(info.error)} />

  const serverTurns = info.data.active ? info.data.turns : []
  const turns = override ?? serverTurns
  const prog = progress ?? { done: info.data.drafts_done, total: info.data.total }

  const send = (e: FormEvent) => {
    e.preventDefault()
    const message = draft.trim()
    if (message.length < 3 || post.isPending) return
    setNote('')
    post.mutate(message, {
      onSuccess: (res) => {
        if (res.done || res.fallback) {
          navigate('/questionnaire', {
            state: { seed: res.seed ?? {}, message: res.message ?? '' },
          })
          return
        }
        setDraft('')
        setOverride(res.turns ?? [])
        setProgress({ done: res.drafts_done ?? 0, total: res.total ?? info.data.total })
        if (res.error) setNote(res.error)
      },
    })
  }

  return (
    <>
      <PageTitle
        title="Chat setup"
        sub="Describe how you live in your own words — we draft your questionnaire answers from the conversation."
      />
      {turns.length === 0 ? (
        <Card>
          <p className="text-sm text-stone-700">
            Mention whatever comes to mind: tidiness, noise, sleep hours, guests, privacy, how you
            split bills, smoking, pets, working from home. A few sentences is plenty — you review
            everything before it counts.
          </p>
          {info.data.text && (
            <details className="mt-3 rounded-lg border border-stone-200 bg-stone-50 px-4 py-2 text-sm">
              <summary className="cursor-pointer font-semibold text-stone-800">
                Your previous description
              </summary>
              <p className="mt-2 whitespace-pre-line text-stone-700">{info.data.text}</p>
            </details>
          )}
          <form onSubmit={send} className="mt-4 flex flex-col gap-3">
            <textarea
              rows={4}
              value={draft}
              onChange={(e) => setDraft(e.target.value)}
              placeholder="e.g. I keep the kitchen spotless, sleep around 11pm, and work from home twice a week…"
              className={inputCls}
            />
            <div className="flex flex-wrap items-center gap-3">
              <button type="submit" disabled={post.isPending} className={primaryBtnCls}>
                {post.isPending ? 'Thinking…' : 'Start chatting'}
              </button>
              <Link
                to="/questionnaire"
                className="text-sm font-semibold text-emerald-800 hover:underline"
              >
                Skip to the plain form
              </Link>
            </div>
          </form>
        </Card>
      ) : (
        <>
          <Card>
            <div className="mb-4 flex items-center gap-3">
              <div className="flex-1">
                <ScoreBar value={prog.done} max={Math.max(prog.total, 1)} />
              </div>
              <span className="text-xs font-semibold tabular-nums text-stone-600">
                {prog.done}/{prog.total} drafted
              </span>
            </div>
            <Turns turns={turns} />
            <form onSubmit={send} className="mt-4 flex flex-col gap-3">
              <textarea
                rows={2}
                value={draft}
                onChange={(e) => setDraft(e.target.value)}
                placeholder="Reply…"
                disabled={post.isPending}
                className={inputCls}
              />
              <div className="flex flex-wrap items-center gap-3">
                <button type="submit" disabled={post.isPending} className={primaryBtnCls}>
                  {post.isPending ? 'Thinking…' : 'Send'}
                </button>
                <Link
                  to="/questionnaire"
                  className="text-sm font-semibold text-emerald-800 hover:underline"
                >
                  Skip to the plain form
                </Link>
              </div>
            </form>
          </Card>
          {note && (
            <div className="mt-4">
              <Notice message={note} />
            </div>
          )}
        </>
      )}
      {post.isError && (
        <div className="mt-4">
          <ErrorNote message={errorMessage(post.error)} />
        </div>
      )}
    </>
  )
}
