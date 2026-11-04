import { useState } from 'react'
import type { FormEvent } from 'react'
import { Link, useNavigate } from 'react-router-dom'
import {
  Card,
  ErrorNote,
  Loading,
  LoadingModal,
  Notice,
  PageTitle,
  errorMessage,
  inputCls,
  primaryBtnCls,
} from '../components'
import { useInterview, useOnboarding } from '../query'

export function InterviewPage() {
  const info = useOnboarding()
  const post = useInterview()
  const navigate = useNavigate()
  const [draft, setDraft] = useState('')
  const [note, setNote] = useState('')

  if (info.isPending) return <Loading label="Loading chat setup…" />
  if (info.isError) return <ErrorNote message={errorMessage(info.error)} />

  const send = (e: FormEvent) => {
    e.preventDefault()
    const message = draft.trim()
    if (post.isPending) return
    if (message.length < 3) {
      setNote('Write a little more — a sentence or two helps.')
      return
    }
    setNote('')
    post.mutate(message, {
      onSuccess: (res) => {
        navigate('/questionnaire', {
          state: { seed: res.seed ?? {}, message: res.message ?? '' },
        })
      },
    })
  }

  return (
    <>
      <PageTitle
        title="Chat setup"
        sub="Describe how you live in your own words — we draft what we can from one message, and you answer the rest in the form."
      />
      <Card>
        <p className="text-[15px] text-sage">
          Mention whatever comes to mind: tidiness, noise, sleep hours, guests, privacy, how you
          split bills, smoking, pets, working from home. A few sentences is plenty — you review
          everything before it counts.
        </p>
        {note && (
          <div className="mt-4">
            <Notice message={note} />
          </div>
        )}
        {info.data.text && (
          <details className="mt-3 rounded-xl border border-sand bg-parchment px-4 py-2 text-sm">
            <summary className="cursor-pointer font-semibold text-ink">
              Your previous description
            </summary>
            <p className="mt-2 whitespace-pre-line text-sage">{info.data.text}</p>
          </details>
        )}
        <form onSubmit={send} className="mt-4 flex flex-col gap-3">
          <textarea
            rows={4}
            value={draft}
            onChange={(e) => setDraft(e.target.value)}
            placeholder="e.g. I keep the kitchen spotless, sleep around 11pm, and work from home twice a week…"
            disabled={post.isPending}
            minLength={3}
            maxLength={2000}
            aria-label="Describe your habits"
            className={inputCls}
          />
          <div className="flex flex-wrap items-center gap-3">
            <button type="submit" disabled={post.isPending} className={primaryBtnCls}>
              {post.isPending ? 'Thinking…' : 'Draft my answers'}
            </button>
            <Link
              to="/questionnaire"
              className="text-sm font-semibold text-eucalyptus hover:underline"
            >
              Skip to the plain form
            </Link>
          </div>
        </form>
      </Card>
      {post.isPending && <LoadingModal label="Thinking…" />}
      {post.isError && (
        <div className="mt-4">
          <ErrorNote message={errorMessage(post.error)} />
        </div>
      )}
    </>
  )
}
