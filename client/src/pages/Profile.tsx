import { useState } from 'react'
import type { FormEvent } from 'react'
import { useNavigate } from 'react-router-dom'
import {
  Card,
  ErrorNote,
  Field,
  Loading,
  PageTitle,
  errorMessage,
  ghostBtnCls,
  inputCls,
  primaryBtnCls,
} from '../components'
import type { Profile } from '../api'
import { useProfile, useSaveProfile } from '../query'

const MODES = [
  { value: 'have-room', label: 'I have a room to offer' },
  { value: 'need-room', label: 'I need a room' },
  { value: 'find-together', label: "Let's find a place together" },
]

const GENDERS = [
  { value: '', label: 'Select…' },
  { value: 'woman', label: 'Woman' },
  { value: 'man', label: 'Man' },
  { value: 'nonbinary', label: 'Non-binary' },
]

const GENDER_PREFS = [
  { value: 'anyone', label: 'Anyone' },
  { value: 'women', label: 'Women' },
  { value: 'men', label: 'Men' },
  { value: 'nonbinary', label: 'Non-binary people' },
]

function ProfileForm({ initial }: { initial: Profile }) {
  const [firstName, setFirstName] = useState(initial.first_name)
  const [mode, setMode] = useState(initial.mode || 'have-room')
  const [areaText, setAreaText] = useState(initial.area_text)
  const [lat, setLat] = useState(initial.lat?.toString() ?? '')
  const [lng, setLng] = useState(initial.lng?.toString() ?? '')
  const [relocating, setRelocating] = useState(initial.relocating)
  const [budgetMin, setBudgetMin] = useState(initial.budget_min ? String(initial.budget_min) : '')
  const [budgetMax, setBudgetMax] = useState(initial.budget_max ? String(initial.budget_max) : '')
  const [currency, setCurrency] = useState(initial.currency || 'USD')
  const [moveEarliest, setMoveEarliest] = useState(initial.move_earliest)
  const [duration, setDuration] = useState(String(initial.duration_months || 6))
  const [gender, setGender] = useState(initial.gender)
  const [genderPref, setGenderPref] = useState(initial.gender_pref || 'anyone')
  const [occupation, setOccupation] = useState(initial.occupation)
  const [bio, setBio] = useState(initial.bio)
  const [contact, setContact] = useState(initial.contact_handle)
  const [formError, setFormError] = useState('')
  const save = useSaveProfile()
  const navigate = useNavigate()

  const onSubmit = (e: FormEvent) => {
    e.preventDefault()
    setFormError('')
    if (budgetMin.trim() === '' || budgetMax.trim() === '') {
      setFormError('Budget must be plain numbers, e.g. 400000.')
      return
    }
    const budgetMinN = Number(budgetMin)
    const budgetMaxN = Number(budgetMax)
    if (!Number.isInteger(budgetMinN) || budgetMinN < 0 || budgetMaxN < budgetMinN) {
      setFormError('Budget must be plain numbers, with max at least min.')
      return
    }
    const durationN = Number(duration)
    if (!Number.isInteger(durationN) || durationN < 1 || durationN > 36) {
      setFormError('Stay duration must be 1–36 months.')
      return
    }
    const latN = lat.trim() === '' ? null : Number(lat)
    const lngN = lng.trim() === '' ? null : Number(lng)
    if ((latN !== null && Number.isNaN(latN)) || (lngN !== null && Number.isNaN(lngN))) {
      setFormError('Location coordinates are invalid.')
      return
    }
    const profile: Profile = {
      first_name: firstName.trim(),
      mode,
      area_text: areaText.trim(),
      lat: latN,
      lng: lngN,
      relocating,
      budget_min: budgetMinN,
      budget_max: budgetMaxN,
      currency: currency.trim().toUpperCase() || 'USD',
      move_earliest: moveEarliest,
      move_latest: initial.move_latest,
      duration_months: durationN,
      gender,
      gender_pref: genderPref,
      occupation: occupation.trim(),
      bio: bio.trim(),
      contact_handle: contact.trim(),
      searching: initial.searching,
    }
    save.mutate(profile, { onSuccess: () => navigate('/questionnaire') })
  }

  const error = formError || (save.isError ? errorMessage(save.error) : '')

  return (
    <form onSubmit={onSubmit}>
      <div className="grid gap-5 md:grid-cols-2">
        <Card>
          <div className="flex flex-col gap-4">
            <Field label="First name">
              <input
                required
                value={firstName}
                onChange={(e) => setFirstName(e.target.value)}
                className={inputCls}
              />
            </Field>
            <Field label="I am looking to">
              <select value={mode} onChange={(e) => setMode(e.target.value)} className={inputCls}>
                {MODES.map((m) => (
                  <option key={m.value} value={m.value}>
                    {m.label}
                  </option>
                ))}
              </select>
            </Field>
            <Field label="Area" hint="Neighbourhood and city, e.g. Yaba, Lagos.">
              <input
                required
                maxLength={120}
                value={areaText}
                onChange={(e) => setAreaText(e.target.value)}
                className={inputCls}
              />
            </Field>
            <label className="flex items-center gap-2 text-sm font-medium text-stone-800">
              <input
                type="checkbox"
                checked={relocating}
                onChange={(e) => setRelocating(e.target.checked)}
                className="h-4 w-4 accent-emerald-800"
              />
              I am relocating to this area
            </label>
            <div className="grid grid-cols-2 gap-4">
              <Field label="Latitude" hint="Optional">
                <input
                  inputMode="decimal"
                  placeholder="6.5244"
                  value={lat}
                  onChange={(e) => setLat(e.target.value)}
                  className={inputCls}
                />
              </Field>
              <Field label="Longitude" hint="Optional">
                <input
                  inputMode="decimal"
                  placeholder="3.3792"
                  value={lng}
                  onChange={(e) => setLng(e.target.value)}
                  className={inputCls}
                />
              </Field>
            </div>
          </div>
        </Card>
        <Card>
          <div className="flex flex-col gap-4">
            <div className="grid grid-cols-3 gap-4">
              <Field label="Budget min">
                <input
                  required
                  inputMode="numeric"
                  value={budgetMin}
                  onChange={(e) => setBudgetMin(e.target.value)}
                  className={inputCls}
                />
              </Field>
              <Field label="Budget max">
                <input
                  required
                  inputMode="numeric"
                  value={budgetMax}
                  onChange={(e) => setBudgetMax(e.target.value)}
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
            <div className="grid grid-cols-2 gap-4">
              <Field label="Move-in from" hint="YYYY-MM-DD">
                <input
                  type="date"
                  required
                  value={moveEarliest}
                  onChange={(e) => setMoveEarliest(e.target.value)}
                  className={inputCls}
                />
              </Field>
              <Field
                label="Stay duration"
                hint={
                  initial.move_latest
                    ? `Current window ends ${initial.move_latest}.`
                    : 'How long do you plan to stay?'
                }
              >
                <div className="flex items-center gap-2">
                  <input
                    required
                    type="number"
                    min={1}
                    max={36}
                    value={duration}
                    onChange={(e) => setDuration(e.target.value)}
                    className={inputCls}
                  />
                  <span className="text-sm text-stone-600">months</span>
                </div>
              </Field>
            </div>
            <div className="grid grid-cols-2 gap-4">
              <Field label="Gender">
                <select
                  required
                  value={gender}
                  onChange={(e) => setGender(e.target.value)}
                  className={inputCls}
                >
                  {GENDERS.map((g) => (
                    <option key={g.value} value={g.value}>
                      {g.label}
                    </option>
                  ))}
                </select>
              </Field>
              <Field label="Match me with" hint="Mutual: both sides must agree.">
                <select
                  value={genderPref}
                  onChange={(e) => setGenderPref(e.target.value)}
                  className={inputCls}
                >
                  {GENDER_PREFS.map((g) => (
                    <option key={g.value} value={g.value}>
                      {g.label}
                    </option>
                  ))}
                </select>
              </Field>
            </div>
            <Field label="Occupation" hint="Shown to your matches.">
              <input
                maxLength={80}
                value={occupation}
                onChange={(e) => setOccupation(e.target.value)}
                className={inputCls}
              />
            </Field>
            <Field label="About you" hint="A line or two for potential flatmates.">
              <textarea
                rows={3}
                maxLength={500}
                value={bio}
                onChange={(e) => setBio(e.target.value)}
                className={inputCls}
              />
            </Field>
            <Field
              label="Contact handle"
              hint="Only revealed on mutual interest, e.g. @you on WhatsApp."
            >
              <input
                maxLength={120}
                value={contact}
                onChange={(e) => setContact(e.target.value)}
                className={inputCls}
              />
            </Field>
          </div>
        </Card>
      </div>
      {error && (
        <div className="mt-4">
          <ErrorNote message={error} />
        </div>
      )}
      <div className="mt-4 flex gap-3">
        <button type="submit" disabled={save.isPending} className={primaryBtnCls}>
          {save.isPending ? 'Saving…' : 'Save and continue'}
        </button>
        <button type="button" onClick={() => navigate('/matches')} className={ghostBtnCls}>
          Back to matches
        </button>
      </div>
    </form>
  )
}

export function ProfilePage() {
  const query = useProfile()
  if (query.isPending) return <Loading label="Loading your profile…" />
  if (query.isError) return <ErrorNote message={errorMessage(query.error)} />
  return (
    <>
      <PageTitle
        title={query.data.found ? 'Your profile' : 'Tell us about yourself'}
        sub="The practical facts matching runs on."
      />
      <ProfileForm initial={query.data.profile} />
    </>
  )
}
