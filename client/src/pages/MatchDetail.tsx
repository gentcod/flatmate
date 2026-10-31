import { useState } from 'react'
import { Link, useNavigate, useParams } from 'react-router-dom'
import {
  Card,
  ErrorNote,
  Loading,
  Notice,
  ScoreBar,
  errorMessage,
  dangerBtnCls,
  ghostBtnCls,
  inputCls,
  primaryBtnCls,
} from '../components'
import type { MatchDetailResponse } from '../api'
import { useMatchAction, useMatchDetail } from '../query'

function DetailBody({ data }: { data: MatchDetailResponse }) {
  const { m } = data
  const navigate = useNavigate()
  const action = useMatchAction(m.id)
  const [reportOpen, setReportOpen] = useState(false)
  const [reason, setReason] = useState('')
  const busy = action.isPending

  const goBack = () => navigate('/matches')
  const meta = [m.gender, m.occupation, m.area_text].filter((s) => s !== '').join(' · ')

  return (
    <>
      <Link
        to="/matches"
        className="mb-4 inline-block text-sm font-semibold text-emerald-800 hover:underline"
      >
        ← All matches
      </Link>
      <Card>
        <div className="flex flex-wrap items-start justify-between gap-4">
          <div>
            <h1 className="text-2xl font-bold tracking-tight text-stone-900">{m.first_name}</h1>
            {meta && <p className="mt-1 text-sm text-stone-600">{meta}</p>}
            <p className="mt-1 text-sm text-stone-600">{m.mode_label}</p>
          </div>
          {data.bd && (
            <div className="text-right">
              <p className="text-4xl font-bold tabular-nums text-stone-900">{data.bd.total}</p>
              <p className="text-xs font-medium text-stone-600">compatibility</p>
            </div>
          )}
        </div>
        {m.bio && <p className="mt-3 text-sm text-stone-700">{m.bio}</p>}
        <dl className="mt-4 grid gap-3 text-sm sm:grid-cols-2">
          <div>
            <dt className="font-semibold text-stone-800">Budget</dt>
            <dd className="tabular-nums text-stone-700">{m.budget_str}</dd>
          </div>
          <div>
            <dt className="font-semibold text-stone-800">Move window</dt>
            <dd className="text-stone-700">{m.move_window}</dd>
          </div>
        </dl>
        {m.contact && (
          <div className="mt-4">
            <Notice message={`Mutual interest — reach them at ${m.contact}`} />
          </div>
        )}
      </Card>

      {data.ineligible || !data.bd ? (
        <div className="mt-4">
          <Notice message="You are not eligible to match — gender preferences or a hard requirement rules this one out." />
        </div>
      ) : (
        <>
          <div className="mt-4">
            <Card>
              <h2 className="text-lg font-bold tracking-tight text-stone-900">
                Why this score
              </h2>
              <div className="mt-3">
                <ScoreBar value={data.bd.total} />
              </div>
              <div className="mt-4 grid gap-4 sm:grid-cols-3">
                {[
                  { label: 'Practical', value: data.bd.practical },
                  { label: 'Lifestyle', value: data.bd.lifestyle },
                  { label: 'Interpersonal', value: data.bd.interpersonal },
                ].map((g) => (
                  <div key={g.label}>
                    <p className="mb-1 flex justify-between text-xs font-semibold text-stone-700">
                      <span>{g.label}</span>
                      <span className="tabular-nums">{g.value}</span>
                    </p>
                    <ScoreBar value={g.value} />
                  </div>
                ))}
              </div>
              {data.top && data.top.length > 0 && (
                <div className="mt-4">
                  <h3 className="text-sm font-bold text-stone-900">Top strengths</h3>
                  <ul className="mt-1 list-disc pl-5 text-sm text-stone-700">
                    {data.top.map((t, i) => (
                      <li key={i}>{t}</li>
                    ))}
                  </ul>
                </div>
              )}
            </Card>
          </div>

          <div className="mt-4">
            <Card>
              <h2 className="text-lg font-bold tracking-tight text-stone-900">
                Dimension by dimension
              </h2>
              <div className="mt-3 overflow-x-auto">
                <table className="w-full text-left text-sm">
                  <thead>
                    <tr className="border-b border-stone-200 text-xs uppercase tracking-wide text-stone-600">
                      <th className="py-2 pr-4 font-semibold">Dimension</th>
                      <th className="w-40 py-2 pr-4 font-semibold">Score</th>
                      <th className="py-2 font-semibold">Note</th>
                    </tr>
                  </thead>
                  <tbody>
                    {data.bd.dimensions.map((d) => (
                      <tr key={d.name} className="border-b border-stone-100 last:border-b-0">
                        <td className="py-2 pr-4 font-medium text-stone-800">
                          {d.name}{' '}
                          {d.aligned && <span className="text-emerald-700">✓</span>}
                          {d.discuss && (
                            <span className="ml-1 rounded bg-amber-100 px-1.5 py-0.5 text-[11px] font-semibold text-amber-900">
                              discuss
                            </span>
                          )}
                        </td>
                        <td className="py-2 pr-4">
                          <div className="flex items-center gap-2">
                            <div className="flex-1">
                              <ScoreBar value={d.score} />
                            </div>
                            <span className="w-8 text-right tabular-nums text-stone-700">
                              {d.score}
                            </span>
                          </div>
                        </td>
                        <td className="py-2 text-stone-600">{d.note}</td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            </Card>
          </div>

          {data.enhanced && data.bullets && data.bullets.length > 0 && (
            <div className="mt-4">
              <Card>
                <h2 className="text-lg font-bold tracking-tight text-stone-900">
                  In plain words{' '}
                  <span className="ml-1 rounded bg-emerald-100 px-1.5 py-0.5 align-middle text-[11px] font-semibold text-emerald-900">
                    AI-narrated
                  </span>
                </h2>
                <ul className="mt-2 list-disc pl-5 text-sm text-stone-700">
                  {data.bullets.map((b, i) => (
                    <li key={i}>{b}</li>
                  ))}
                </ul>
                {data.discuss_q && data.discuss_q.length > 0 && (
                  <>
                    <h3 className="mt-4 text-sm font-bold text-stone-900">
                      Worth asking each other
                    </h3>
                    <ul className="mt-1 list-disc pl-5 text-sm text-stone-700">
                      {data.discuss_q.map((q, i) => (
                        <li key={i}>{q}</li>
                      ))}
                    </ul>
                  </>
                )}
                {data.bridges && data.bridges.length > 0 && (
                  <>
                    <h3 className="mt-4 text-sm font-bold text-stone-900">
                      Possible compromises
                    </h3>
                    <ul className="mt-1 flex flex-col gap-1 text-sm text-stone-700">
                      {data.bridges.map((b, i) => (
                        <li key={i}>
                          <span className="font-semibold">{b.dimension}:</span> {b.suggestion}
                        </li>
                      ))}
                    </ul>
                  </>
                )}
              </Card>
            </div>
          )}
        </>
      )}

      <div className="mt-4">
        <Card>
          <h2 className="text-lg font-bold tracking-tight text-stone-900">Actions</h2>
          <div className="mt-3 flex flex-wrap gap-2">
            {m.saved ? (
              <button
                type="button"
                disabled={busy}
                onClick={() => action.mutate({ kind: 'unsave' })}
                className={ghostBtnCls}
              >
                Unsave
              </button>
            ) : (
              <button
                type="button"
                disabled={busy}
                onClick={() => action.mutate({ kind: 'save' })}
                className={ghostBtnCls}
              >
                Save
              </button>
            )}
            <button
              type="button"
              disabled={busy || m.my_interest}
              onClick={() => action.mutate({ kind: 'interest' })}
              className={primaryBtnCls}
            >
              {m.my_interest ? 'Interest sent ✓' : 'Show interest'}
            </button>
            <button
              type="button"
              disabled={busy}
              onClick={() => action.mutate({ kind: 'dismiss' }, { onSuccess: goBack })}
              className={ghostBtnCls}
            >
              Dismiss
            </button>
            <button
              type="button"
              disabled={busy}
              onClick={() => {
                if (window.confirm(`Block ${m.first_name}? This hides you from each other.`)) {
                  action.mutate({ kind: 'block' }, { onSuccess: goBack })
                }
              }}
              className={dangerBtnCls}
            >
              Block
            </button>
            <button
              type="button"
              disabled={busy}
              onClick={() => setReportOpen((v) => !v)}
              className={ghostBtnCls}
            >
              Report
            </button>
          </div>
          {reportOpen && (
            <form
              className="mt-3 flex flex-col gap-2"
              onSubmit={(e) => {
                e.preventDefault()
                action.mutate({ kind: 'report', reason }, { onSuccess: goBack })
              }}
            >
              <input
                value={reason}
                onChange={(e) => setReason(e.target.value)}
                maxLength={500}
                placeholder="Reason (optional)"
                className={inputCls}
              />
              <div>
                <button type="submit" disabled={busy} className={dangerBtnCls}>
                  Submit report
                </button>
              </div>
            </form>
          )}
          {action.isError && (
            <div className="mt-3">
              <ErrorNote message={errorMessage(action.error)} />
            </div>
          )}
        </Card>
      </div>
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
