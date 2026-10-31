import { useState } from 'react'
import { Link, useNavigate } from 'react-router-dom'
import {
  Card,
  ErrorNote,
  Loading,
  PageTitle,
  dangerBtnCls,
  errorMessage,
  ghostBtnCls,
} from '../components'
import { useDeactivate, useOnboarding, useProfile, useSaveProfile } from '../query'

export function SettingsPage() {
  const profile = useProfile()
  const onboarding = useOnboarding()
  const deactivate = useDeactivate()
  const save = useSaveProfile()
  const navigate = useNavigate()
  const [confirming, setConfirming] = useState(false)

  if (profile.isPending || onboarding.isPending) return <Loading label="Loading settings…" />
  if (profile.isError) return <ErrorNote message={errorMessage(profile.error)} />
  if (onboarding.isError) return <ErrorNote message={errorMessage(onboarding.error)} />

  const p = profile.data.profile

  return (
    <>
      <PageTitle title="Settings" sub="Your details, your saved words, and your account." />
      <div className="flex flex-col gap-4">
        <Card>
          <h2 className="text-lg font-bold tracking-tight text-stone-900">Your details</h2>
          <p className="mt-1 text-sm text-stone-700">
            {p.first_name || 'No name yet'}
            {p.area_text && ` · ${p.area_text}`}
            {p.occupation && ` · ${p.occupation}`}
          </p>
          <div className="mt-3 flex flex-wrap gap-4 text-sm font-semibold text-emerald-800">
            <Link to="/profile" className="hover:underline">
              Edit profile →
            </Link>
            <Link to="/questionnaire" className="hover:underline">
              Edit answers →
            </Link>
            <Link to="/onboarding" className="hover:underline">
              Redo chat setup →
            </Link>
          </div>
        </Card>

        <Card>
          <h2 className="text-lg font-bold tracking-tight text-stone-900">Visibility</h2>
          <label className="mt-2 flex items-center gap-2 text-sm font-medium text-stone-800">
            <input
              type="checkbox"
              checked={p.searching}
              disabled={save.isPending}
              onChange={() => save.mutate({ ...p, searching: !p.searching })}
              className="h-4 w-4 accent-emerald-800"
            />
            Visible to matches
          </label>
          <p className="mt-1 text-sm text-stone-600">
            {p.searching
              ? 'People can match with you and get in touch.'
              : "You're hidden — found a roommate? Stay hidden so nobody new matches you."}
          </p>
          {save.isError && (
            <div className="mt-2">
              <ErrorNote message={errorMessage(save.error)} />
            </div>
          )}
        </Card>

        <Card>
          <h2 className="text-lg font-bold tracking-tight text-stone-900">Saved description</h2>
          {onboarding.data.text ? (
            <p className="mt-2 whitespace-pre-line text-sm text-stone-700">
              {onboarding.data.text}
            </p>
          ) : (
            <p className="mt-2 text-sm text-stone-600">
              Nothing saved yet — it appears here after you use chat setup.
            </p>
          )}
          {onboarding.data.drafts && onboarding.data.drafts.length > 0 && (
            <div className="mt-3">
              <h3 className="text-sm font-bold text-stone-900">Drafted answers</h3>
              <ul className="mt-1 flex flex-col gap-1 text-sm text-stone-700">
                {onboarding.data.drafts.map((d) => (
                  <li key={d.dimension}>
                    <span className="font-semibold">{d.dimension}:</span> {d.value}
                    {d.hard && <span className="ml-1 text-xs text-stone-600">(must match)</span>}
                  </li>
                ))}
              </ul>
            </div>
          )}
        </Card>

        <Card>
          <h2 className="text-lg font-bold tracking-tight text-rose-900">Danger zone</h2>
          {!confirming ? (
            <button type="button" onClick={() => setConfirming(true)} className={dangerBtnCls}>
              Deactivate account
            </button>
          ) : (
            <div className="mt-2">
              <p className="text-sm text-stone-700">
                Deactivating hides your profile and matches immediately and logs you out. This
                cannot be undone from the app.
              </p>
              <div className="mt-3 flex gap-2">
                <button
                  type="button"
                  disabled={deactivate.isPending}
                  onClick={() =>
                    deactivate.mutate(undefined, { onSuccess: () => navigate('/login') })
                  }
                  className={dangerBtnCls}
                >
                  {deactivate.isPending ? 'Deactivating…' : 'Yes, deactivate'}
                </button>
                <button
                  type="button"
                  disabled={deactivate.isPending}
                  onClick={() => setConfirming(false)}
                  className={ghostBtnCls}
                >
                  Keep my account
                </button>
              </div>
            </div>
          )}
          {deactivate.isError && (
            <div className="mt-3">
              <ErrorNote message={errorMessage(deactivate.error)} />
            </div>
          )}
        </Card>
      </div>
    </>
  )
}
