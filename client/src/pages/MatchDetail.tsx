import { useState } from 'react'
import type { FormEvent, ReactNode } from 'react'
import { Link, useNavigate, useParams } from 'react-router-dom'
import {
  Card,
  Chip,
  ErrorNote,
  Icon,
  Loading,
  LoadingModal,
  Notice,
  ScoreBar,
  Spinner,
  Toast,
  TrustBadge,
  errorMessage,
  dangerBtnCls,
  ghostBtnCls,
  inputCls,
  primaryBtnCls,
} from '../components'
import type { DetailPerson, MatchDetailResponse } from '../api'
import type { ToastData, ToastTone } from '../components'
import { useMatchAction, useMatchDetail } from '../query'
import {
  useDraftAgreement,
  useProposeStay,
  useReviews,
  useSaveReview,
  useStayTransition,
} from '../query_social'

function StayCard({
  m,
  notify,
}: {
  m: DetailPerson
  notify: (message: string, tone: ToastTone) => void
}) {
  const propose = useProposeStay()
  const transition = useStayTransition(m.stay_id)
  const busy = propose.isPending || transition.isPending
  const done = (message: string) => ({
    onSuccess: () => notify(message, 'ok' as const),
    onError: (e: unknown) => notify(errorMessage(e), 'err' as const),
  })

  const proposeBtn = (label: string) => (
    <button
      type="button"
      disabled={busy}
      onClick={() => propose.mutate({ peer_id: m.id }, done('Stay proposal sent.'))}
      className={primaryBtnCls}
    >
      {propose.isPending ? (
        <>
          <Spinner /> Sending…
        </>
      ) : (
        label
      )}
    </button>
  )

  let body: ReactNode = null
  if (m.stay_status === 'active') {
    body = (
      <>
        <Notice message="Living together — verified stay." />
        <button
          type="button"
          disabled={busy}
          onClick={() => transition.mutate('end', done('Stay ended.'))}
          className={`${ghostBtnCls} mt-2`}
        >
          {transition.isPending ? 'Ending…' : 'End stay'}
        </button>
      </>
    )
  } else if (m.stay_status === 'ended') {
    body = <Notice message="Lived together before — verified stay." />
  } else if (m.stay_status === 'proposed') {
    body = m.stay_proposed_by_me ? (
      <p className="text-sm text-sage">Stay proposal sent — waiting for them to confirm.</p>
    ) : (
      <>
        <p className="text-sm text-sage">{m.first_name} says you live together.</p>
        <div className="mt-2 flex flex-wrap gap-2">
          <button
            type="button"
            disabled={busy}
            onClick={() => transition.mutate('accept', done('Stay confirmed.'))}
            className={primaryBtnCls}
          >
            {transition.isPending ? 'Confirming…' : 'Confirm stay'}
          </button>
          <button
            type="button"
            disabled={busy}
            onClick={() => transition.mutate('decline', done('Proposal declined.'))}
            className={ghostBtnCls}
          >
            Decline
          </button>
        </div>
      </>
    )
  } else if (m.stay_status === 'declined') {
    body = (
      <>
        <p className="text-sm text-sage">Stay proposal declined.</p>
        {m.mutual && <div className="mt-2">{proposeBtn('Propose stay again')}</div>}
      </>
    )
  } else if (m.mutual) {
    body = proposeBtn('We live together — propose stay')
  } else {
    return null
  }
  return (
    <div className="rounded-xl bg-mist p-4">
      <p className="mb-2 text-[12px] font-semibold uppercase tracking-wider text-fog">
        Shared home
      </p>
      {body}
    </div>
  )
}

function DetailBody({ data }: { data: MatchDetailResponse }) {
  const { m } = data
  const navigate = useNavigate()
  const action = useMatchAction(m.id)
  const [reportOpen, setReportOpen] = useState(false)
  const [toast, setToast] = useState<ToastData | null>(null)
  const [showAI, setShowAI] = useState(true)
  const [reason, setReason] = useState('')
  const reviews = useReviews(m.id)
  const saveReview = useSaveReview(m.id)
  const [rating, setRating] = useState(5)
  const [reviewBody, setReviewBody] = useState('')
  const [reviewSaved, setReviewSaved] = useState(false)
  const submitReview = (e: FormEvent) => {
    e.preventDefault()
    if (reviewBody.trim() === '' || saveReview.isPending) return
    setReviewSaved(false)
    saveReview.mutate(
      { rating, body: reviewBody.trim() },
      {
        onSuccess: () => {
          setReviewBody('')
          setReviewSaved(true)
        },
      },
    )
  }
  const agreement = useDraftAgreement(m.id)
  const busy = action.isPending

  const goBack = () => navigate('/matches')
  const meta = [m.gender, m.occupation, m.area_text].filter((s) => s !== '').join(' · ')
  const initial = (m.first_name || '?').slice(0, 1).toUpperCase()
  const iconFor = (text: string): string => {
    const t = text.toLowerCase()
    if (t.includes('sleep') || t.includes('quiet') || t.includes('noise')) return 'bedtime'
    if (t.includes('clean')) return 'mop'
    if (t.includes('guest') || t.includes('social')) return 'groups'
    if (t.includes('work') || t.includes('wfh') || t.includes('home')) return 'desk'
    if (t.includes('budget') || t.includes('rent') || t.includes('money') || t.includes('expense'))
      return 'payments'
    if (t.includes('smok')) return 'smoke_free'
    if (t.includes('pet')) return 'pets'
    return 'check_circle'
  }

  return (
    <>
      <div className="mb-6 flex items-center justify-between border-b border-sand pb-4">
        <div className="flex items-center gap-2 text-sm text-sage">
          <Link to="/matches" className="flex items-center gap-1 hover:text-ink">
            <Icon name="arrow_back" cls="text-[18px]" />
            Matches
          </Link>
          <span className="text-fog">/</span>
          <span className="font-semibold text-ink">{m.first_name}</span>
        </div>
        {m.mutual && (
          <span className="inline-flex items-center gap-1.5 rounded-full bg-terracotta px-3 py-1 text-[12px] font-semibold uppercase tracking-wide text-white">
            Mutual interest
          </span>
        )}
      </div>
      <div className="grid grid-cols-1 items-start gap-8 lg:grid-cols-12">
        <div className="flex flex-col gap-10 lg:col-span-8">
      {data.bd && (
        <Card>
          <div className="flex flex-col justify-between gap-6 md:flex-row md:items-center">
            <div className="max-w-lg space-y-2">
              <span className="inline-block rounded bg-parchment px-2.5 py-0.5 text-[12px] font-semibold uppercase tracking-wider text-eucalyptus">
                Compatibility assessment
              </span>
              <h1 className="font-display text-4xl tracking-tight text-ink">
                {data.bd.total}% Overall Compatibility
              </h1>
              <p className="text-[15px] leading-relaxed text-sage">
                Advisory score from a fixed formula across 12 dimensions. AI narrates; people
                decide.
              </p>
            </div>
            <div className="flex shrink-0 flex-col items-center justify-center self-center">
              <div className="relative flex h-32 w-32 items-center justify-center">
                <svg className="h-32 w-32 -rotate-90" viewBox="0 0 120 120">
                  <circle
                    cx="60"
                    cy="60"
                    r="50"
                    fill="none"
                    stroke="#e5dfd5"
                    strokeWidth="8"
                    className="opacity-60"
                  />
                  <circle
                    cx="60"
                    cy="60"
                    r="50"
                    fill="none"
                    stroke="#1e3f35"
                    strokeWidth="8"
                    strokeLinecap="round"
                    strokeDasharray="314.159"
                    strokeDashoffset={(314.159 * (100 - data.bd.total) / 100).toFixed(2)}
                  />
                </svg>
                <span className="absolute font-display text-[26px] leading-none text-ink">
                  {data.bd.total}%
                </span>
              </div>
            </div>
          </div>
          <div className="mt-6 grid gap-4 border-t border-sand pt-5 sm:grid-cols-3">
            {[
              { label: 'Practical', value: data.bd.practical },
              { label: 'Lifestyle', value: data.bd.lifestyle },
              { label: 'Interpersonal', value: data.bd.interpersonal },
            ].map((g) => (
              <div key={g.label}>
                <p className="mb-1 flex justify-between text-[13px] font-semibold text-ink">
                  <span>{g.label}</span>
                  <span className="tabular-nums">{g.value}%</span>
                </p>
                <ScoreBar value={g.value} />
              </div>
            ))}
          </div>
        </Card>
      )}

      {data.ineligible || !data.bd ? (
        <div className="mt-4">
          <Notice message="You are not eligible to match — gender preferences or a hard requirement rules this one out." />
        </div>
      ) : (
        <>
          {data.top && data.top.length > 0 && (
            <div className="flex flex-col gap-4">
              <div className="flex items-baseline justify-between">
                <h2 className="text-[20px] font-semibold tracking-tight text-ink">
                  Primary synergy drivers
                </h2>
                <span className="text-[13px] text-fog">Highest correlated traits</span>
              </div>
              <div className="grid grid-cols-1 gap-4 md:grid-cols-3">
                {data.top.map((t, i) => (
                  <div
                    key={i}
                    className="flex flex-col justify-between gap-3 rounded-2xl border border-sand bg-white p-6 shadow-card"
                  >
                    <span className="flex h-10 w-10 items-center justify-center rounded-xl bg-parchment text-eucalyptus">
                      <Icon name={iconFor(t)} />
                    </span>
                    <p className="text-[15px] font-medium leading-snug text-ink">{t}</p>
                  </div>
                ))}
              </div>
            </div>
          )}

          <Card>
            <div className="border-b border-sand pb-4">
              <h2 className="text-[20px] font-semibold tracking-tight text-ink">
                Detailed habit comparison
              </h2>
              <p className="mt-0.5 text-sm text-sage">Computed dimension by dimension</p>
            </div>
            <div className="flex flex-col gap-6 pt-6">
              {data.bd.dimensions.map((d) => (
                <div key={d.name} className="flex flex-col gap-2">
                  <div className="flex items-center justify-between">
                    <div className="flex items-center gap-2">
                      <span className="text-[17px] font-semibold text-ink">{d.name}</span>
                      {d.aligned && <Icon name="check_circle" cls="text-[18px] text-canopy" />}
                      {d.discuss && (
                        <span className="rounded bg-parchment px-2 py-0.5 text-[12px] font-semibold text-clay">
                          discuss
                        </span>
                      )}
                    </div>
                    <span className="text-sm font-semibold tabular-nums text-eucalyptus">
                      {d.score}% match
                    </span>
                  </div>
                  {d.note && (
                    <p className="rounded-xl bg-parchment p-3 text-sm text-sage">{d.note}</p>
                  )}
                  <ScoreBar value={d.score} tone={d.discuss ? 'pivotal' : 'standard'} />
                </div>
              ))}
            </div>
          </Card>

          {data.enhanced && data.bullets && data.bullets.length > 0 && (
            <div className="flex flex-col gap-5 rounded-2xl bg-parchment p-7">
              <div className="flex items-center justify-between gap-3">
                <div className="flex items-center gap-2.5">
                  <Icon name="auto_awesome" cls="text-[20px] text-eucalyptus" />
                  <h2 className="text-[20px] font-semibold tracking-tight text-ink">
                    Suggested discussion starters
                  </h2>
                  <span className="rounded bg-mist px-2 py-0.5 text-[11px] font-semibold text-eucalyptus">
                    AI-narrated
                  </span>
                </div>
                <button
                  type="button"
                  onClick={() => setShowAI((v) => !v)}
                  className="cursor-pointer text-[13px] text-sage underline decoration-sand underline-offset-4 hover:text-ink"
                >
                  {showAI ? 'Hide AI suggestions' : 'Show AI suggestions'}
                </button>
              </div>
              {showAI && (
                <>
                  <ul className="flex flex-col gap-2 text-sm text-sage">
                    {data.bullets.map((b, i) => (
                      <li key={i}>{b}</li>
                    ))}
                  </ul>
                  {data.discuss_q && data.discuss_q.length > 0 && (
                    <div className="grid grid-cols-1 gap-4 md:grid-cols-2">
                      {data.discuss_q.map((q, i) => (
                        <div
                          key={i}
                          className="flex items-start gap-3 rounded-xl bg-white p-4 shadow-card"
                        >
                          <Icon name="forum" cls="mt-0.5 text-[18px] text-terracotta" />
                          <p className="text-sm text-sage">{q}</p>
                        </div>
                      ))}
                    </div>
                  )}
                  {data.bridges && data.bridges.length > 0 && (
                    <div className="flex flex-col gap-2">
                      <h3 className="text-sm font-bold text-ink">Possible compromises</h3>
                      {data.bridges.map((b, i) => (
                        <p key={i} className="text-sm text-sage">
                          <span className="font-semibold text-ink">{b.dimension}:</span>{' '}
                          {b.suggestion}
                        </p>
                      ))}
                    </div>
                  )}
                </>
              )}
            </div>
          )}
        </>
      )}

      <Card>
          <div className="flex items-center justify-between border-b border-sand pb-4">
            <h2 className="text-[20px] font-semibold tracking-tight text-ink">
              Flatmate reviews
            </h2>
            {reviews.data && reviews.data.count > 0 && (
              <span className="rounded bg-mist px-2 py-0.5 text-[13px] font-semibold text-eucalyptus">
                {reviews.data.avg.toFixed(1)} / 5.0 ({reviews.data.count})
              </span>
            )}
          </div>
          {reviews.isPending ? (
            <p className="pt-4 text-sm text-sage">Loading reviews…</p>
          ) : reviews.data && reviews.data.count > 0 ? (
            <div className="grid grid-cols-1 gap-6 pt-6 md:grid-cols-2">
              {reviews.data.reviews.map((rv) => (
                <div key={rv.author_id} className="flex flex-col justify-between rounded-xl bg-parchment p-5">
                  <p className="mb-4 font-display text-[19px] italic leading-relaxed text-ink">
                    “{rv.body}”
                  </p>
                  <div className="flex items-center justify-between border-t border-sand pt-3">
                    <span className="text-sm font-semibold text-ink">{rv.author_name}</span>
                    <span className="flex items-center gap-2">
                      {rv.verified_stay && <Chip>Verified stay</Chip>}
                      <span className="text-[13px] font-medium tabular-nums text-eucalyptus">
                        {rv.rating}/5
                      </span>
                    </span>
                  </div>
                </div>
              ))}
            </div>
          ) : (
            <p className="pt-4 text-sm text-sage">No reviews yet.</p>
          )}
          {m.can_review ? (
            <form className="mt-4 flex flex-col gap-2" onSubmit={submitReview}>
              <select
                value={rating}
                onChange={(e) => setRating(Number(e.target.value))}
                aria-label="Rating"
                className={inputCls}
              >
                {[5, 4, 3, 2, 1].map((n) => (
                  <option key={n} value={n}>
                    {n} stars
                  </option>
                ))}
              </select>
              <textarea
                value={reviewBody}
                onChange={(e) => setReviewBody(e.target.value)}
                rows={2}
                maxLength={500}
                placeholder="What was it like living with them?"
                className={inputCls}
              />
              <div>
                <button type="submit" disabled={saveReview.isPending} className={ghostBtnCls}>
                  {saveReview.isPending ? 'Saving…' : 'Save review'}
                </button>
              </div>
            </form>
          ) : (
            <p className="mt-4 text-sm text-sage">
              Reviews unlock after a verified stay together.
            </p>
          )}
          {reviewSaved && <p className="mt-2 text-sm font-semibold text-canopy">Saved.</p>}
          {saveReview.isError && (
            <div className="mt-2">
              <ErrorNote message={errorMessage(saveReview.error)} />
            </div>
          )}
        </Card>
        </div>
        <aside className="flex flex-col gap-6 lg:col-span-4 lg:sticky lg:top-24">
          <div className="flex flex-col overflow-hidden rounded-2xl border border-sand bg-white shadow-card">
            <div className="flex items-center gap-4 bg-eucalyptus p-6">
              <span
                aria-hidden="true"
                className="flex h-16 w-16 shrink-0 items-center justify-center rounded-full bg-canopy font-display text-3xl text-canvas"
              >
                {initial}
              </span>
              <div>
                <h2 className="font-display text-[26px] font-medium tracking-tight text-white">
                  {m.first_name}
                </h2>
                {meta && <p className="mt-0.5 text-sm text-[#aacec0]">{meta}</p>}
              </div>
            </div>
            <div className="flex flex-col gap-5 p-6">
              <div>
                <TrustBadge selfConfirmed={m.contact !== '' && m.contact_self_confirmed} />
              </div>
              <p className="text-sm text-sage">{m.mode_label}</p>
              <div className="grid grid-cols-2 gap-3 border-y border-sand py-4">
                <div className="flex flex-col">
                  <span className="text-[13px] text-fog">Budget</span>
                  <span className="text-[17px] font-semibold tabular-nums text-eucalyptus">
                    {m.budget_str}
                  </span>
                </div>
                <div className="flex flex-col">
                  <span className="text-[13px] text-fog">Move window</span>
                  <span className="text-[17px] font-semibold text-ink">{m.move_window}</span>
                </div>
              </div>
              {m.bio && (
                <div className="space-y-1">
                  <span className="text-[12px] font-semibold uppercase tracking-wider text-fog">
                    A bit about them
                  </span>
                  <p className="text-sm leading-relaxed text-sage">“{m.bio}”</p>
                </div>
              )}
              {m.contact && (
                <div className="flex flex-col gap-2.5 rounded-xl bg-mist p-4">
                  <div className="flex items-center gap-2">
                    <Icon name="check_circle" cls="text-[18px] text-terracotta" />
                    <span className="text-sm font-semibold text-eucalyptus">
                      Mutual interest — contact unlocked
                    </span>
                  </div>
                  <p className="select-all rounded-lg bg-white p-3 text-sm font-semibold text-ink">
                    {m.contact}{' '}
                    <span className="font-normal text-sage">
                      ({m.contact_self_confirmed ? 'self-confirmed' : 'unverified'})
                    </span>
                  </p>
                </div>
              )}
              <StayCard
                m={m}
                notify={(message, tone) => setToast({ id: Date.now(), message, tone })}
              />
              <div className="flex flex-col gap-2.5 pt-2">
                {m.mutual ? (
                  <Link
                    to={`/chat/${m.id}`}
                    className={`${primaryBtnCls} flex w-full items-center justify-center gap-2 py-3 text-center text-[15px]`}
                  >
                    <Icon name="forum" />
                    Open Direct Chat
                  </Link>
                ) : (
                  <button
                    type="button"
                    disabled={busy || m.my_interest}
                    onClick={() =>
                      action.mutate(
                        { kind: 'interest' },
                        {
                          onSuccess: () =>
                            setToast({
                              id: Date.now(),
                              message: 'Interest sent.',
                              tone: 'ok',
                            }),
                          onError: (e) =>
                            setToast({
                              id: Date.now(),
                              message: errorMessage(e),
                              tone: 'err',
                            }),
                        },
                      )
                    }
                    className={`${primaryBtnCls} flex w-full items-center justify-center gap-2 py-3 text-[15px]`}
                  >
                    {busy ? (
                      <>
                        <Spinner /> Sending…
                      </>
                    ) : m.my_interest ? (
                      'Interest sent ✓'
                    ) : (
                      'Show interest'
                    )}
                  </button>
                )}
                <a
                  href="#agreement"
                  className="flex w-full items-center justify-center gap-2 rounded-xl bg-terracotta px-4 py-3 text-[15px] font-semibold text-white shadow-sm hover:bg-clay"
                >
                  <Icon name="handshake" />
                  Draft Flatmate Agreement
                </a>
                <div className="flex gap-2">
                  <button
                    type="button"
                    disabled={busy}
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
                            setToast({
                              id: Date.now(),
                              message: errorMessage(e),
                              tone: 'err',
                            }),
                        },
                      )
                    }
                    className={`${ghostBtnCls} flex-1`}
                  >
                    {m.saved ? 'Unsave' : 'Save'}
                  </button>
                  <button
                    type="button"
                    disabled={busy}
                    onClick={() =>
                      action.mutate(
                        { kind: 'dismiss' },
                        {
                          onSuccess: goBack,
                          onError: (e) =>
                            setToast({
                              id: Date.now(),
                              message: errorMessage(e),
                              tone: 'err',
                            }),
                        },
                      )
                    }
                    className={`${ghostBtnCls} flex-1`}
                  >
                    Dismiss
                  </button>
                </div>
              </div>
              <div className="pt-2 text-center">
                <button
                  type="button"
                  disabled={busy}
                  onClick={() => setReportOpen((v) => !v)}
                  className="mx-auto flex cursor-pointer items-center justify-center gap-1 text-[13px] text-fog hover:text-clay hover:underline disabled:cursor-not-allowed"
                >
                  <Icon name="flag" cls="text-[14px]" />
                  Report or Block Profile
                </button>
                {reportOpen && (
                  <form
                    className="mt-3 flex flex-col gap-2 text-left"
                    onSubmit={(e) => {
                      e.preventDefault()
                      action.mutate(
                        { kind: 'report', reason },
                        {
                          onSuccess: goBack,
                          onError: (e) =>
                            setToast({
                              id: Date.now(),
                              message: errorMessage(e),
                              tone: 'err',
                            }),
                        },
                      )
                    }}
                  >
                    <input
                      value={reason}
                      onChange={(e) => setReason(e.target.value)}
                      maxLength={500}
                      placeholder="Reason (optional)"
                      aria-label="Reason for reporting (optional)"
                      className={inputCls}
                    />
                    <div className="flex gap-2">
                      <button type="submit" disabled={busy} className={dangerBtnCls}>
                        {busy ? (
                          <>
                            <Spinner /> Submitting…
                          </>
                        ) : (
                          'Submit report'
                        )}
                      </button>
                      <button
                        type="button"
                        disabled={busy}
                        onClick={() => {
                          if (
                            window.confirm(
                              `Block ${m.first_name}? This hides you from each other.`,
                            )
                          ) {
                            action.mutate(
                              { kind: 'block' },
                              {
                                onSuccess: goBack,
                                onError: (e) =>
                                  setToast({
                                    id: Date.now(),
                                    message: errorMessage(e),
                                    tone: 'err',
                                  }),
                              },
                            )
                          }
                        }}
                        className={ghostBtnCls}
                      >
                        Block
                      </button>
                    </div>
                  </form>
                )}
                {action.isError && (
                  <div className="mt-3 text-left">
                    <ErrorNote message={errorMessage(action.error)} />
                  </div>
                )}
              </div>
            </div>
          </div>
          <Card>
            <div id="agreement">
            <h2 className="font-display text-2xl tracking-tight text-ink">Roommate agreement</h2>
          <p className="mt-1 text-sm text-sage">
            Not legal advice — a starting point for your conversation.
          </p>
          <button
            type="button"
            disabled={agreement.isPending}
            onClick={() => agreement.mutate(undefined)}
            className="mt-2 flex w-full cursor-pointer items-center justify-center gap-2 rounded-xl bg-terracotta px-4 py-3 text-[15px] font-semibold text-white shadow-sm hover:bg-clay disabled:cursor-not-allowed disabled:opacity-50"
          >
            {agreement.isPending ? 'Drafting…' : 'Draft agreement'}
          </button>
          {agreement.isPending && <LoadingModal label="Drafting your agreement…" />}
          {agreement.data && (
            <div className="mt-3">
              <p className="text-xs font-semibold text-fog">
                {agreement.data.enhanced ? 'Drafted with AI' : 'Template version (AI unavailable)'}
              </p>
              {agreement.data.sections.map((s) => (
                <div key={s.heading} className="mt-2">
                  <h3 className="text-sm font-bold text-ink">{s.heading}</h3>
                  <ul className="list-disc pl-5 text-sm text-sage">
                    {s.points.map((pt, i) => (
                      <li key={i}>{pt}</li>
                    ))}
                  </ul>
                </div>
              ))}
            </div>
          )}
          {agreement.isError && (
            <div className="mt-2">
              <ErrorNote message={errorMessage(agreement.error)} />
            </div>
          )}
          </div>
        </Card>
        </aside>
      </div>
      {toast && <Toast toast={toast} onClose={() => setToast(null)} />}
    </>
  )
}

export function MatchDetailPage() {
  const params = useParams()
  const id = Number(params.id)
  const valid = Number.isInteger(id) && id > 0
  const detail = useMatchDetail(valid ? id : 0, valid)

  if (!valid) return <ErrorNote message="Not found." />
  if (detail.isPending) return <Loading label="Loading this match…" />
  if (detail.isError) return <ErrorNote message={errorMessage(detail.error)} />
  return <DetailBody data={detail.data} />
}
