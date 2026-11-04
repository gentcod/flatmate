import { useState } from 'react'
import type { FormEvent } from 'react'
import { Link, useNavigate, useParams } from 'react-router-dom'
import {
  Card,
  Chip,
  ErrorNote,
  Field,
  Notice,
  Icon,
  Loading,
  LoadingModal,
  Spinner,
  Toast,
  errorMessage,
  dangerBtnCls,
  ghostBtnCls,
  inputCls,
  primaryBtnCls,
} from '../components'
import type { Listing } from '../api'
import type { ToastData } from '../components'
import { useMatchAction, useMatchDetail } from '../query'
import {
  useDeleteListing,
  useDraftAgreement,
  useListing,
  useListingReviews,
  useListingStatus,
  useProposeStay,
  useSaveListingReview,
  useStays,
  useUpdateListing,
} from '../query_social'

function EditForm({ l }: { l: Listing }) {
  const [title, setTitle] = useState(l.title)
  const [description, setDescription] = useState(l.description)
  const [area, setArea] = useState(l.area_text)
  const [rent, setRent] = useState(String(l.monthly_rent))
  const [currency, setCurrency] = useState(l.currency)
  const [photo, setPhoto] = useState(l.photo_url)
  const [formError, setFormError] = useState('')
  const [saved, setSaved] = useState(false)
  const update = useUpdateListing(l.id)

  const onSubmit = (e: FormEvent) => {
    e.preventDefault()
    setFormError('')
    setSaved(false)
    const rentN = Number(rent)
    if (rent.trim() === '' || !Number.isInteger(rentN) || rentN < 0) {
      setFormError('Rent must be a plain number, 0 or more.')
      return
    }
    update.mutate(
      {
        title: title.trim(),
        description: description.trim(),
        area_text: area.trim(),
        monthly_rent: rentN,
        currency: currency.trim().toUpperCase() || 'USD',
        photo_url: photo.trim(),
      },
      { onSuccess: () => setSaved(true) },
    )
  }

  const error = formError || (update.isError ? errorMessage(update.error) : '')

  return (
    <form onSubmit={onSubmit} className="flex flex-col gap-4">
      <Field label="Title">
        <input
          required
          maxLength={80}
          value={title}
          onChange={(e) => setTitle(e.target.value)}
          className={inputCls}
        />
      </Field>
      <Field label="Area">
        <input
          required
          maxLength={120}
          value={area}
          onChange={(e) => setArea(e.target.value)}
          className={inputCls}
        />
      </Field>
      <div className="grid grid-cols-2 gap-4">
        <Field label="Monthly rent">
          <input
            required
            inputMode="numeric"
            value={rent}
            onChange={(e) => setRent(e.target.value)}
            className={inputCls}
          />
        </Field>
        <Field label="Currency">
          <input
            required
            maxLength={8}
            value={currency}
            onChange={(e) => setCurrency(e.target.value)}
            className={inputCls}
          />
        </Field>
      </div>
      <Field
        label="Photo URL (optional)"
        hint="A /placeholders/… path or https:// link. Uploads are coming later."
      >
        <input
          maxLength={500}
          value={photo}
          onChange={(e) => setPhoto(e.target.value)}
          placeholder="/placeholders/room-1.svg"
          className={inputCls}
        />
      </Field>
      <Field label="Description">
        <textarea
          rows={3}
          maxLength={1000}
          value={description}
          onChange={(e) => setDescription(e.target.value)}
          className={inputCls}
        />
      </Field>
      {error && <ErrorNote message={error} />}
      {saved && <p className="text-sm font-semibold text-canopy">Saved.</p>}
      <div>
        <button type="submit" disabled={update.isPending} className={primaryBtnCls}>
          {update.isPending ? 'Saving…' : 'Save changes'}
        </button>
      </div>
    </form>
  )
}

function DetailBody({ l }: { l: Listing }) {
  const navigate = useNavigate()
  const [editing, setEditing] = useState(false)
  const [reportOpen, setReportOpen] = useState(false)
  const [reason, setReason] = useState('')
  const [income, setIncome] = useState('')
  const [pct, setPct] = useState('30')
  const del = useDeleteListing()
  const [confirmDelete, setConfirmDelete] = useState(false)
  const report = useMatchAction(l.user_id)
  const status = useListingStatus(l.id)

  return (
    <>
      <Link
        to="/listings"
        className="mb-6 inline-flex items-center gap-1 text-sm font-semibold text-sage hover:text-ink"
      >
        <Icon name="arrow_back" cls="text-[18px]" />
        All listings
      </Link>
      <div className="overflow-hidden rounded-2xl border border-sand bg-white shadow-card">
        {l.photo_url && (
          <img
            src={l.photo_url}
            alt=""
            loading="lazy"
            className="aspect-[16/9] w-full object-cover"
          />
        )}
        <div className="flex items-end justify-between gap-3 bg-eucalyptus px-6 py-6">
          <div>
            <p className="font-display text-3xl font-medium tracking-tight text-white">
              {l.currency} {l.monthly_rent.toLocaleString()}
              <span className="font-sans text-[15px] font-normal text-[#aacec0]">/mo</span>
            </p>
            <p className="mt-1 flex items-center gap-1 text-sm text-[#aacec0]">
              <Icon name="location_on" cls="text-[16px]" />
              {l.area_text}
            </p>
          </div>
          <div className="flex shrink-0 flex-col items-end gap-1.5">
            {l.mine && (
              <span className="rounded-full bg-canopy px-2.5 py-1 text-[11px] font-semibold uppercase tracking-wide text-white">
                Yours
              </span>
            )}
            {l.status === 'rented' && (
              <span className="rounded-full bg-terracotta px-2.5 py-1 text-[11px] font-semibold uppercase tracking-wide text-white">
                Rented
              </span>
            )}
          </div>
        </div>
        <div className="flex flex-col gap-3 p-6">
          <h1 className="font-display text-3xl tracking-tight text-ink">{l.title}</h1>
          <p className="text-sm text-sage">Posted by {l.owner_name}</p>
          {l.description && (
            <p className="whitespace-pre-line text-[15px] leading-relaxed text-sage">
              {l.description}
            </p>
          )}
          {!l.mine && (
            <p className="text-sm">
              <Link
                to={`/match/${l.user_id}`}
                className="font-semibold text-eucalyptus hover:underline"
              >
                View {l.owner_name}&rsquo;s profile →
              </Link>
            </p>
          )}
        </div>
      </div>

      <div className="mt-6">
        <Card>
          <h2 className="font-display text-2xl tracking-tight text-ink">
            Can you afford this?
          </h2>
          <div className="mt-2 grid grid-cols-2 gap-3">
            <Field label="Monthly income">
              <input
                value={income}
                onChange={(e) => setIncome(e.target.value)}
                inputMode="numeric"
                className={inputCls}
                placeholder="e.g. 500000"
              />
            </Field>
            <Field label="Max % on rent">
              <input
                value={pct}
                onChange={(e) => setPct(e.target.value)}
                inputMode="numeric"
                className={inputCls}
              />
            </Field>
          </div>
          {(() => {
            const inc = parseInt(income, 10)
            const p = parseInt(pct, 10)
            if (!income || Number.isNaN(inc) || Number.isNaN(p) || inc <= 0 || p <= 0)
              return (
                <p className="mt-2 text-sm text-sage">
                  Enter your income and a percentage — nothing is saved.
                </p>
              )
            const max = Math.floor((inc * p) / 100)
            const ok = max >= l.monthly_rent
            return (
              <div className="mt-2">
                <Notice tone={ok ? 'ok' : 'warn'}
                  message={
                    ok
                      ? `Yes — your ceiling is ${l.currency} ${max}.`
                      : `Tight — your ceiling is ${l.currency} ${max}, rent is ${l.currency} ${l.monthly_rent}.`
                  }
                />
              </div>
            )
          })()}
        </Card>
      </div>
      <div className="mt-6">
        <ListingReviews listingId={l.id} mine={l.mine} />
      </div>
      {!l.mine && (
        <ConnectPanel ownerId={l.user_id} ownerName={l.owner_name} listingId={l.id} />
      )}
      {l.mine ? (
        <div className="mt-4">
          <Card>
            <div className="flex flex-wrap gap-2">
              <button type="button" onClick={() => setEditing((v) => !v)} className={ghostBtnCls}>
                {editing ? 'Close editor' : 'Edit'}
              </button>
              <button
                type="button"
                disabled={status.isPending}
                onClick={() => status.mutate(l.status === 'rented' ? 'available' : 'rented')}
                className={ghostBtnCls}
              >
                {status.isPending
                  ? 'Saving…'
                  : l.status === 'rented'
                    ? 'Mark available'
                    : 'Mark rented'}
              </button>
              <button
                type="button"
                disabled={del.isPending}
                onClick={() => {
                  if (confirmDelete) {
                    del.mutate(l.id, { onSuccess: () => navigate('/listings') })
                  } else {
                    setConfirmDelete(true)
                  }
                }}
                className={dangerBtnCls}
              >
                {confirmDelete ? (del.isPending ? 'Deleting…' : 'Confirm delete?') : 'Delete'}
              </button>
            </div>
            {editing && (
              <div className="mt-4">
                <EditForm l={l} />
              </div>
            )}
            {del.isError && (
              <div className="mt-3">
                <ErrorNote message={errorMessage(del.error)} />
              </div>
            )}
            {status.isError && (
              <div className="mt-3">
                <ErrorNote message={errorMessage(status.error)} />
              </div>
            )}
          </Card>
        </div>
      ) : (
        <div className="mt-4">
          <Card>
            <h2 className="font-display text-2xl tracking-tight text-ink">Safety</h2>
            <p className="mt-1 text-sm text-sage">
              Listings are unverified. Reporting flags the owner for review.
            </p>
            <button
              type="button"
              onClick={() => setReportOpen((v) => !v)}
              className={`${ghostBtnCls} mt-2`}
            >
              Report this listing
            </button>
            {reportOpen && (
              <form
                className="mt-3 flex flex-col gap-2"
                onSubmit={(e) => {
                  e.preventDefault()
                  report.mutate({ kind: 'report', reason })
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
                <div>
                  <button type="submit" disabled={report.isPending} className={dangerBtnCls}>
                    Submit report
                  </button>
                </div>
              </form>
            )}
            {report.isError && (
              <div className="mt-3">
                <ErrorNote message={errorMessage(report.error)} />
              </div>
            )}
            {report.isSuccess && (
              <p className="mt-2 text-sm font-semibold text-canopy">Reported. Thanks.</p>
            )}
          </Card>
        </div>
      )}
    </>
  )
}

function ListingReviews({ listingId, mine }: { listingId: number; mine: boolean }) {
  const query = useListingReviews(listingId)
  const save = useSaveListingReview(listingId)
  const stays = useStays()
  const [rating, setRating] = useState(5)
  const [body, setBody] = useState('')
  const [kind, setKind] = useState('visit')
  const [saved, setSaved] = useState(false)
  const canStay = (stays.data?.stays ?? []).some(
    (s) => s.listing_id === listingId && (s.status === 'active' || s.status === 'ended'),
  )

  const onSubmit = (e: FormEvent) => {
    e.preventDefault()
    if (body.trim() === '' || save.isPending) return
    setSaved(false)
    save.mutate(
      { rating, body: body.trim(), kind },
      {
        onSuccess: () => {
          setBody('')
          setSaved(true)
        },
      },
    )
  }

  const list = (reviews: { author_name: string; rating: number; body: string }[], tenant: boolean) =>
    reviews.length === 0 ? (
      <p className="text-sm text-sage">{tenant ? 'No tenant reviews yet.' : 'No visitor reviews yet.'}</p>
    ) : (
      <div className="grid grid-cols-1 gap-4 md:grid-cols-2">
        {reviews.map((rv, i) => (
          <div key={i} className="flex flex-col justify-between rounded-xl bg-parchment p-4">
            <p className="mb-3 font-display text-[17px] italic leading-relaxed text-ink">
              “{rv.body}”
            </p>
            <div className="flex items-center justify-between border-t border-sand pt-2">
              <span className="text-sm font-semibold text-ink">{rv.author_name}</span>
              <span className="flex items-center gap-2">
                {tenant && <Chip>Verified stay</Chip>}
                <span className="text-[13px] font-medium tabular-nums text-eucalyptus">
                  {rv.rating}/5
                </span>
              </span>
            </div>
          </div>
        ))}
      </div>
    )

  return (
    <Card>
      <div className="flex flex-wrap items-baseline justify-between gap-2">
        <h2 className="font-display text-2xl tracking-tight text-ink">Reviews</h2>
        {query.data && (query.data.stay_count > 0 || query.data.visit_count > 0) && (
          <span className="rounded bg-mist px-2 py-0.5 text-[13px] font-semibold text-eucalyptus">
            Trust {query.data.trust_score.toFixed(1)}/5 · {query.data.stay_count} tenant,{' '}
            {query.data.visit_count} visitor
          </span>
        )}
      </div>
      {query.isPending ? (
        <p className="mt-2 text-sm text-sage">Loading reviews…</p>
      ) : query.isError ? (
        <div className="mt-2">
          <ErrorNote message={errorMessage(query.error)} />
        </div>
      ) : (
        <>
          <h3 className="mb-2 mt-4 text-sm font-bold text-ink">From tenants</h3>
          {list(query.data.stay_reviews, true)}
          <h3 className="mb-2 mt-4 text-sm font-bold text-ink">From visitors</h3>
          {list(query.data.visit_reviews, false)}
        </>
      )}
      {!mine && (
        <form className="mt-4 flex flex-col gap-2" onSubmit={onSubmit}>
          <div className="grid grid-cols-2 gap-2">
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
            <select
              value={kind}
              onChange={(e) => setKind(e.target.value)}
              aria-label="Visit or stay"
              className={inputCls}
            >
              <option value="visit">I visited</option>
              {canStay && <option value="stay">I lived here (verified)</option>}
            </select>
          </div>
          <textarea
            value={body}
            onChange={(e) => setBody(e.target.value)}
            rows={2}
            maxLength={500}
            placeholder="What was the place like?"
            className={inputCls}
          />
          <div>
            <button type="submit" disabled={save.isPending} className={ghostBtnCls}>
              {save.isPending ? 'Saving…' : 'Save review'}
            </button>
          </div>
        </form>
      )}
      {saved && <p className="mt-2 text-sm font-semibold text-canopy">Saved.</p>}
      {save.isError && (
        <div className="mt-2">
          <ErrorNote message={errorMessage(save.error)} />
        </div>
      )}
    </Card>
  )
}

function ConnectPanel({
  ownerId,
  ownerName,
  listingId,
}: {
  ownerId: number
  ownerName: string
  listingId: number
}) {
  const match = useMatchDetail(ownerId)
  const action = useMatchAction(ownerId)
  const agreement = useDraftAgreement(ownerId)
  const propose = useProposeStay()
  const [toast, setToast] = useState<ToastData | null>(null)
  const m = match.data?.m
  if (match.isPending) return <Loading label="Checking connection status…" />
  if (match.isError || !m) return null
  return (
    <div className="mt-6">
      <Card>
        <h2 className="font-display text-2xl tracking-tight text-ink">
          Connect with {ownerName}
        </h2>
        {m.mutual ? (
          <>
            {m.contact && (
              <p className="mt-2 text-sm text-sage">
                Mutual interest — reach them at{' '}
                <span className="font-semibold text-ink">{m.contact}</span> (
                {m.contact_self_confirmed ? 'self-confirmed' : 'unverified'})
              </p>
            )}
            <div className="mt-3 flex flex-wrap gap-2">
              <Link to={`/chat/${ownerId}`} className={primaryBtnCls}>
                Open chat
              </Link>
              <button
                type="button"
                disabled={agreement.isPending}
                onClick={() => agreement.mutate()}
                className="cursor-pointer rounded-xl bg-terracotta px-4 py-2 text-sm font-semibold text-white shadow-sm hover:bg-clay disabled:cursor-not-allowed disabled:opacity-50"
              >
                {agreement.isPending ? 'Drafting…' : 'Draft agreement'}
              </button>
              {agreement.isPending && <LoadingModal label="Drafting your agreement…" />}
              {m.stay_status === 'active' || m.stay_status === 'ended' ? (
                <span className="inline-flex items-center rounded-full bg-mist px-3 py-2 text-[13px] font-semibold text-eucalyptus">
                  Verified stay ✓
                </span>
              ) : m.stay_status === 'proposed' ? (
                <Link to={`/match/${ownerId}`} className={ghostBtnCls}>
                  Stay proposal pending — respond
                </Link>
              ) : (
                <button
                  type="button"
                  disabled={propose.isPending}
                  onClick={() =>
                    propose.mutate(
                      { peer_id: ownerId, listing_id: listingId },
                      {
                        onSuccess: () =>
                          setToast({
                            id: Date.now(),
                            message: 'Stay proposal sent.',
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
                  className={ghostBtnCls}
                >
                  {propose.isPending ? 'Sending…' : 'We live here together'}
                </button>
              )}
            </div>
            {agreement.isError && (
              <div className="mt-2">
                <ErrorNote message={errorMessage(agreement.error)} />
              </div>
            )}
            {agreement.data && (
              <div className="mt-4 border-t border-sand pt-4">
                <h3 className="font-display text-xl tracking-tight text-ink">
                  Roommate agreement
                </h3>
                <p className="mt-1 text-xs font-semibold text-fog">
                  {agreement.data.enhanced
                    ? 'Drafted with AI.'
                    : 'Drafted from a template (AI unavailable).'}{' '}
                  Not legal advice.
                </p>
                <div className="mt-2 flex flex-col gap-3">
                  {agreement.data.sections.map((s) => (
                    <div key={s.heading}>
                      <h4 className="text-sm font-bold text-ink">{s.heading}</h4>
                      <ul className="list-disc pl-5 text-sm text-sage">
                        {s.points.map((p) => (
                          <li key={p}>{p}</li>
                        ))}
                      </ul>
                    </div>
                  ))}
                </div>
              </div>
            )}
          </>
        ) : (
          <>
            <p className="mt-2 text-sm text-sage">
              Interested in this room? Show interest in {ownerName} to unlock chat.
            </p>
            <div className="mt-3">
              <button
                type="button"
                disabled={action.isPending || m.my_interest}
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
                className={primaryBtnCls}
              >
                {action.isPending ? (
                  <>
                    <Spinner /> Sending…
                  </>
                ) : m.my_interest ? (
                  'Interest sent ✓'
                ) : (
                  `Show interest in ${ownerName}`
                )}
              </button>
            </div>
            {action.isError && (
              <div className="mt-2">
                <ErrorNote message={errorMessage(action.error)} />
              </div>
            )}
          </>
        )}
        {toast && <Toast toast={toast} onClose={() => setToast(null)} />}
      </Card>
    </div>
  )
}

export function ListingDetailPage() {
  const params = useParams()
  const id = Number(params.id)
  const valid = Number.isInteger(id) && id > 0
  const detail = useListing(valid ? id : 0, valid)

  if (!valid) return <ErrorNote message="Not found." />
  if (detail.isPending) return <Loading label="Loading this listing…" />
  if (detail.isError) return <ErrorNote message={errorMessage(detail.error)} />
  return <DetailBody l={detail.data.listing} />
}
