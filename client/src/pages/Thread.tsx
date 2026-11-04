import { useEffect, useRef, useState } from 'react'
import type { FormEvent } from 'react'
import { Link, useParams } from 'react-router-dom'
import {
  Card,
  ErrorNote,
  Loading,
  Spinner,
  errorMessage,
  inputCls,
  primaryBtnCls,
} from '../components'
import { isApiError } from '../api'
import { useQueryClient } from '@tanstack/react-query'
import { useSendMessage, useThread } from '../query_social'

export function ThreadPage() {
  const params = useParams()
  const peer = Number(params.id)
  const valid = Number.isInteger(peer) && peer > 0
  const thread = useThread(valid ? peer : 0, valid)
  const send = useSendMessage(valid ? peer : 0)
  const [draft, setDraft] = useState('')
  const qc = useQueryClient()
  const bottomRef = useRef<HTMLDivElement>(null)
  const count = thread.data?.messages.length ?? 0
  // Server marks the thread read on fetch; refresh the inbox so badges clear,
  // and keep the newest message in view when the thread grows.
  useEffect(() => {
    void qc.invalidateQueries({ queryKey: ['conversations'] })
    bottomRef.current?.scrollIntoView()
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [count])

  if (!valid) return <ErrorNote message="Not found." />
  if (thread.isPending) return <Loading label="Loading this chat…" />
  if (thread.isError) {
    if (isApiError(thread.error) && thread.error.status === 403) {
      return (
        <Card>
          <p className="text-[15px] text-sage">
            {thread.error.message}{' '}
            <Link
              to={`/match/${peer}`}
              className="font-semibold text-eucalyptus hover:underline"
            >
              View their profile →
            </Link>
          </p>
        </Card>
      )
    }
    return <ErrorNote message={errorMessage(thread.error)} />
  }

  const onSend = (e: FormEvent) => {
    e.preventDefault()
    const body = draft.trim()
    if (body === '' || send.isPending) return
    send.mutate(body, { onSuccess: () => setDraft('') })
  }

  return (
    <>
      <div className="mb-4 flex gap-4">
        <Link
          to="/chat"
          className="inline-block text-sm font-semibold text-sage hover:text-ink hover:underline"
        >
          ← All chats
        </Link>
        <Link
          to={`/expenses/${peer}`}
          className="inline-block text-sm font-semibold text-sage hover:text-ink hover:underline"
        >
          Split expenses →
        </Link>
      </div>
      <h1 className="mb-4 font-display text-3xl tracking-tight text-ink">
        {thread.data.peer.first_name}
      </h1>
      <Card>
        {thread.data.messages.length === 0 ? (
          <p className="text-sm text-sage">
            No messages yet. Say hello — keep it friendly, arrange to meet in public first.
          </p>
        ) : (
          <div className="flex flex-col gap-3">
            {thread.data.messages.map((m) => (
              <div key={m.id} className={m.from_me ? 'flex justify-end' : 'flex justify-start'}>
                <div
                  className={`flex max-w-[80%] flex-col ${m.from_me ? 'items-end' : 'items-start'}`}
                >
                  <p
                    className={`rounded-xl px-4 py-2 text-sm ${
                      m.from_me
                        ? 'bg-eucalyptus text-white'
                        : 'border border-sand bg-white text-ink shadow-card'
                    }`}
                  >
                    {m.body}
                  </p>
                  <span className="mt-1 text-[11px] tabular-nums text-fog">
                    {m.created_at.slice(0, 16).replace('T', ' ')}
                  </span>
                </div>
              </div>
            ))}
            <div ref={bottomRef} />
          </div>
        )}
        <form onSubmit={onSend} className="mt-4 flex gap-2">
          <input
            value={draft}
            onChange={(e) => setDraft(e.target.value)}
            maxLength={2000}
            placeholder="Write a message…"
            disabled={send.isPending}
            className={inputCls}
          />
          <button type="submit" disabled={send.isPending} className={primaryBtnCls}>
            {send.isPending ? (
              <>
                <Spinner /> Sending…
              </>
            ) : (
              'Send'
            )}
          </button>
        </form>
        {send.isError && (
          <div className="mt-3">
            <ErrorNote message={errorMessage(send.error)} />
          </div>
        )}
      </Card>
    </>
  )
}
