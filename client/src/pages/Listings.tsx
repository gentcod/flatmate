import { useState } from 'react'
import type { FormEvent } from 'react'
import { Link } from 'react-router-dom'
import {
  Card,
  ErrorNote,
  Field,
  Icon,
  Loading,
  Notice,
  PageTitle,
  Toast,
  errorMessage,
  ghostBtnCls,
  inputCls,
  primaryBtnCls,
} from '../components'
import type { Listing, ListingInput } from '../api'
import type { ToastData } from '../components'
import {
  useCreateListing,
  useDeleteSearch,
  useListings,
  useSaveSearch,
  useSearches,
} from '../query_social'

function ListingCard({ l }: { l: Listing }) {
  const initial = (l.owner_name || '?').slice(0, 1).toUpperCase()
  return (
    <article className="flex flex-col overflow-hidden rounded-2xl border border-sand bg-white shadow-card transition-shadow hover:shadow-lift">
      {l.photo_url ? (
        <img
          src={l.photo_url}
          alt=""
          loading="lazy"
          className="aspect-[16/9] w-full object-cover"
        />
      ) : (
        <div
          aria-hidden="true"
          className="flex aspect-[16/9] w-full items-center justify-center bg-parchment"
        >
          <Icon name="apartment" cls="text-[40px] text-fog" />
        </div>
      )}
      <div className="flex items-end justify-between gap-3 bg-eucalyptus px-5 py-5">
        <div>
          <p className="font-display text-[26px] font-medium tracking-tight text-white">
            {l.currency} {l.monthly_rent.toLocaleString()}
            <span className="font-sans text-sm font-normal text-[#aacec0]">/mo</span>
          </p>
          <p className="mt-0.5 flex items-center gap-1 text-[13px] text-[#aacec0]">
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
      <div className="flex flex-1 flex-col gap-3 p-5">
        <h2 className="font-display text-2xl tracking-tight text-ink">
          <Link to={`/listing/${l.id}`} className="hover:underline">
            {l.title}
          </Link>
        </h2>
        {l.description && (
          <p className="line-clamp-2 text-sm leading-relaxed text-sage">{l.description}</p>
        )}
        <div className="mt-auto flex items-center justify-between pt-1">
          <span className="flex items-center gap-2 text-[13px] text-sage">
            <span
              aria-hidden="true"
              className="flex h-8 w-8 items-center justify-center rounded-full bg-parchment font-display text-[15px] text-eucalyptus"
            >
              {initial}
            </span>
            {l.owner_name}
          </span>
          <Link
            to={`/listing/${l.id}`}
            className="text-sm font-semibold text-eucalyptus hover:underline"
          >
            View room →
          </Link>
        </div>
      </div>
    </article>
  )
}

function NewListingForm({ onDone }: { onDone: () => void }) {
  const [title, setTitle] = useState('')
  const [description, setDescription] = useState('')
  const [area, setArea] = useState('')
  const [rent, setRent] = useState('')
  const [currency, setCurrency] = useState('USD')
  const [photo, setPhoto] = useState('')
  const [formError, setFormError] = useState('')
  const create = useCreateListing()

  const onSubmit = (e: FormEvent) => {
    e.preventDefault()
    setFormError('')
    const rentN = Number(rent)
    if (rent.trim() === '' || !Number.isInteger(rentN) || rentN < 0) {
      setFormError('Rent must be a plain number, 0 or more.')
      return
    }
    const input: ListingInput = {
      title: title.trim(),
      description: description.trim(),
      area_text: area.trim(),
      monthly_rent: rentN,
      currency: currency.trim().toUpperCase() || 'USD',
      photo_url: photo.trim(),
    }
    create.mutate(input, {
      onSuccess: () => {
        setTitle('')
        setDescription('')
        setArea('')
        setRent('')
        setPhoto('')
        onDone()
      },
    })
  }

  const error = formError || (create.isError ? errorMessage(create.error) : '')

  return (
    <Card>
      <form onSubmit={onSubmit} className="flex flex-col gap-4">
        <Field label="Title">
          <input
            required
            maxLength={80}
            value={title}
            onChange={(e) => setTitle(e.target.value)}
            placeholder="e.g. Sunny room in Yaba"
            className={inputCls}
          />
        </Field>
        <Field label="Area">
          <input
            required
            maxLength={120}
            value={area}
            onChange={(e) => setArea(e.target.value)}
            placeholder="e.g. Yaba, Lagos"
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
              placeholder="150000"
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
        <Field label="Description" hint="The room, the flat, the household — up to 1000 characters.">
          <textarea
            rows={3}
            maxLength={1000}
            value={description}
            onChange={(e) => setDescription(e.target.value)}
            className={inputCls}
          />
        </Field>
        {error && <ErrorNote message={error} />}
        <div>
          <button type="submit" disabled={create.isPending} className={primaryBtnCls}>
            {create.isPending ? 'Posting…' : 'Post listing'}
          </button>
        </div>
      </form>
    </Card>
  )
}

export function ListingsPage() {
  const [input, setInput] = useState('')
  const [area, setArea] = useState('')
  const [rentInput, setRentInput] = useState('')
  const [maxRent, setMaxRent] = useState('')
  const [mine, setMine] = useState(false)
  const [showForm, setShowForm] = useState(false)
  const [posted, setPosted] = useState(false)
  const [saveName, setSaveName] = useState('')
  const [filterError, setFilterError] = useState('')
  const [toast, setToast] = useState<ToastData | null>(null)
  const [confirmDelSearch, setConfirmDelSearch] = useState<number | null>(null)
  const query = useListings(area, mine, maxRent)
  const searches = useSearches()
  const saveSearch = useSaveSearch()
  const delSearch = useDeleteSearch()

  const applyFilters = (e: FormEvent) => {
    e.preventDefault()
    if (rentInput.trim() !== '' && !/^\d+$/.test(rentInput.trim())) {
      setFilterError('Max rent must be a plain number.')
      return
    }
    setFilterError('')
    setPosted(false)
    setArea(input.trim())
    setMaxRent(rentInput.trim())
  }

  const applySearch = (areaValue: string, maxRentValue: number) => {
    setInput(areaValue)
    setPosted(false)
    setArea(areaValue)
    const rent = maxRentValue > 0 ? String(maxRentValue) : ''
    setRentInput(rent)
    setMaxRent(rent)
  }

  const onSaveSearch = (e: FormEvent) => {
    e.preventDefault()
    if (saveName.trim() === '' || saveSearch.isPending) return
    saveSearch.mutate(
      { name: saveName.trim(), area, max_rent: maxRent === '' ? 0 : Number(maxRent) },
      { onSuccess: () => setSaveName('') },
    )
  }

  return (
    <>
      <div className="flex flex-wrap items-end justify-between gap-3">
        <PageTitle title="Room listings" sub="Rooms on offer from people with profiles." />
        <button
          type="button"
          onClick={() => setShowForm((v) => !v)}
          className={`${primaryBtnCls} mb-6 flex items-center gap-2`}
        >
          <Icon name="add_home" cls="text-[18px]" />
          {showForm ? 'Close' : 'Post a room'}
        </button>
      </div>
      {showForm && (
        <div className="mb-4">
          <NewListingForm
            onDone={() => {
              setShowForm(false)
              setPosted(true)
            }}
          />
        </div>
      )}
      {posted && !showForm && (
        <div className="mb-4">
          <Notice message="Posted. It is visible to everyone searching." />
        </div>
      )}
      <Card>
        <h2 className="font-display text-2xl tracking-tight text-ink">Saved searches</h2>
        {searches.isPending ? (
          <p className="mt-1 text-sm text-sage">Loading…</p>
        ) : searches.isError ? (
          <p className="mt-1 text-sm text-sage">Couldn&rsquo;t load saved searches.</p>
        ) : searches.data.searches.length === 0 ? (
          <p className="mt-1 text-sm text-sage">
            None yet. Filter below, then save it to watch for new rooms.
          </p>
        ) : (
          <ul className="mt-2 flex flex-col gap-2">
            {searches.data.searches.map((s) => (
              <li key={s.id} className="flex flex-wrap items-center gap-2 text-sm">
                <button
                  type="button"
                  onClick={() => applySearch(s.area, s.max_rent)}
                  className="cursor-pointer font-semibold text-eucalyptus hover:underline"
                >
                  {s.name}
                </button>
                <span className="text-sage">
                  {s.area || 'anywhere'}
                  {s.max_rent > 0 && ` · under ${s.max_rent.toLocaleString()}`} ·{' '}
                  {s.new_listings} new
                </span>
                <button
                  type="button"
                  disabled={delSearch.isPending}
                  onClick={() => {
                    if (confirmDelSearch !== s.id) {
                      setConfirmDelSearch(s.id)
                      return
                    }
                    setConfirmDelSearch(null)
                    delSearch.mutate(s.id, {
                      onSuccess: () =>
                        setToast({ id: Date.now(), message: 'Search deleted.', tone: 'ok' }),
                      onError: (e) =>
                        setToast({ id: Date.now(), message: errorMessage(e), tone: 'err' }),
                    })
                  }}
                  className="cursor-pointer text-xs font-semibold text-fog hover:underline disabled:cursor-not-allowed"
                >
                  {confirmDelSearch === s.id ? 'Confirm?' : 'Delete'}
                </button>
              </li>
            ))}
          </ul>
        )}
        <form onSubmit={onSaveSearch} className="mt-3 flex gap-2">
          <input
            value={saveName}
            onChange={(e) => setSaveName(e.target.value)}
            placeholder="Name this filter, e.g. Yaba under 200k"
            aria-label="Name for this saved search"
            maxLength={40}
            className={inputCls}
          />
          <button type="submit" disabled={saveSearch.isPending} className={ghostBtnCls}>
            Save
          </button>
        </form>
        {saveSearch.isError && (
          <div className="mt-2">
            <ErrorNote message={errorMessage(saveSearch.error)} />
          </div>
        )}
      </Card>
      <div className="my-6">
        <Card>
          <form onSubmit={applyFilters} className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-4">
            <div className="flex flex-col gap-1">
              <label htmlFor="listing-area" className="text-[13px] font-medium text-sage">
                City & district
              </label>
              <input
                id="listing-area"
                value={input}
                onChange={(e) => setInput(e.target.value)}
                placeholder="e.g. Lagos"
                maxLength={120}
                className={inputCls}
              />
            </div>
            <div className="flex flex-col gap-1">
              <label htmlFor="listing-rent" className="text-[13px] font-medium text-sage">
                Max target rent{filterError ? ` — ${filterError}` : ''}
              </label>
              <input
                id="listing-rent"
                value={rentInput}
                onChange={(e) => setRentInput(e.target.value)}
                placeholder="No cap"
                inputMode="numeric"
                className={inputCls}
              />
            </div>
            <div className="flex items-end gap-2">
              <button type="submit" className={primaryBtnCls}>
                Filter
              </button>
              {(area || maxRent) && (
                <button
                  type="button"
                  className={ghostBtnCls}
                  onClick={() => {
                    setInput('')
                    setPosted(false)
                    setFilterError('')
                    setArea('')
                    setRentInput('')
                    setMaxRent('')
                  }}
                >
                  Clear
                </button>
              )}
            </div>
            <label className="flex items-end gap-2 pb-2 text-sm font-medium text-sage">
              <input
                type="checkbox"
                checked={mine}
                onChange={(e) => setMine(e.target.checked)}
                className="h-5 w-5 accent-eucalyptus"
              />
              Mine only
            </label>
          </form>
        </Card>
      </div>
      {query.isPending ? (
        <Loading label="Loading listings…" />
      ) : query.isError ? (
        <ErrorNote message={errorMessage(query.error)} />
      ) : query.data.listings.length === 0 ? (
        <Card>
          <p className="text-[15px] text-sage">
            No listings yet. Post the first room above.
          </p>
        </Card>
      ) : (
        <div className="grid grid-cols-1 gap-6 lg:grid-cols-2">
          {query.data.listings.map((l) => (
            <ListingCard key={l.id} l={l} />
          ))}
        </div>
      )}
      {toast && <Toast toast={toast} onClose={() => setToast(null)} />}
    </>
  )
}
