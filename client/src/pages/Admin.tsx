import { useState } from 'react'
import { Link } from 'react-router-dom'
import {
  Card,
  ErrorNote,
  Loading,
  PageTitle,
  Spinner,
  Toast,
  dangerBtnCls,
  errorMessage,
  ghostBtnCls,
} from '../components'
import type { ToastData } from '../components'
import { useSession } from '../query'
import { useMetrics, useReports, useResolveReport } from '../query_admin'

export function AdminPage() {
  const session = useSession()
  const isAdmin = session.data?.is_admin ?? false
  const reportsQuery = useReports(isAdmin)
  const metricsQuery = useMetrics(isAdmin)
  const resolve = useResolveReport()
  const [toast, setToast] = useState<ToastData | null>(null)
  const [pendingId, setPendingId] = useState<number | null>(null)
  const [confirmId, setConfirmId] = useState<number | null>(null)

  if (session.isPending) return <Loading label="Checking your session…" />
  if (!isAdmin) {
    return (
      <Card>
        <h1 className="font-display text-3xl tracking-tight text-ink">Admin</h1>
        <p className="mt-2 text-sm text-sage">
          Admins only.{' '}
          <Link to="/matches" className="font-semibold text-eucalyptus hover:underline">
            Back to matches
          </Link>
        </p>
      </Card>
    )
  }

  const run = (id: number, action: 'dismiss' | 'deactivate', okMessage: string) => {
    setPendingId(id)
    resolve.mutate(
      { id, action },
      {
        onSuccess: () => setToast({ id: Date.now(), message: okMessage, tone: 'ok' }),
        onError: (e) => setToast({ id: Date.now(), message: errorMessage(e), tone: 'err' }),
        onSettled: () => {
          setPendingId(null)
          setConfirmId(null)
        },
      },
    )
  }

  const stats = metricsQuery.data
    ? [
        { label: 'Users', value: metricsQuery.data.users },
        { label: 'Active users', value: metricsQuery.data.active_users },
        { label: 'Open reports', value: metricsQuery.data.open_reports },
        { label: 'Listings', value: metricsQuery.data.listings },
        { label: 'Reviews', value: metricsQuery.data.reviews },
        { label: 'Expenses', value: metricsQuery.data.expenses },
      ]
    : []

  return (
    <div>
      <PageTitle title="Admin" sub="Reports and platform metrics." />

      <div className="mt-4">
        <Card>
          <h2 className="font-display text-lg">Metrics</h2>
          {metricsQuery.isPending && <Loading label="Loading metrics…" />}
          {metricsQuery.isError && <ErrorNote message={errorMessage(metricsQuery.error)} />}
          {stats.length > 0 && (
            <dl className="mt-2 grid grid-cols-2 gap-3 sm:grid-cols-3">
              {stats.map((s) => (
                <div key={s.label} className="rounded-xl bg-parchment px-3 py-2">
                  <dt className="text-xs font-medium text-fog">{s.label}</dt>
                  <dd className="text-xl font-bold tabular-nums text-ink">{s.value}</dd>
                </div>
              ))}
            </dl>
          )}
        </Card>
      </div>

      <div className="mt-6">
        <Card>
          <h2 className="font-display text-lg">Reports</h2>
          {reportsQuery.isPending && <Loading label="Loading reports…" />}
          {reportsQuery.isError && <ErrorNote message={errorMessage(reportsQuery.error)} />}
          {resolve.isError && <ErrorNote message={errorMessage(resolve.error)} />}
          {reportsQuery.data && reportsQuery.data.reports.length === 0 && (
            <p className="mt-2 text-sm text-sage">No reports.</p>
          )}
          <ul className="mt-2 divide-y divide-sand">
            {(reportsQuery.data?.reports ?? []).map((r) => (
              <li key={r.id} className="py-3">
                <p className="text-sm text-ink">
                  <Link
                    to={`/match/${r.reporter}`}
                    className="font-semibold text-eucalyptus hover:underline"
                  >
                    {r.reporter_name}
                  </Link>{' '}
                  reported{' '}
                  <Link
                    to={`/match/${r.target}`}
                    className="font-semibold text-eucalyptus hover:underline"
                  >
                    {r.target_name}
                  </Link>
                </p>
                <p className="mt-1 text-sm text-sage">{r.reason}</p>
                <p className="mt-1 text-xs text-fog">
                  {r.created_at.slice(0, 16).replace('T', ' ')} ·{' '}
                  {r.resolved_at ? `Resolved (${r.resolution})` : 'Open'}
                </p>
                {!r.resolved_at && (
                  <div className="mt-2 flex flex-wrap gap-2">
                    <button
                      type="button"
                      disabled={resolve.isPending}
                      onClick={() => run(r.id, 'dismiss', 'Report dismissed.')}
                      className={ghostBtnCls}
                    >
                      {pendingId === r.id ? (
                        <>
                          <Spinner /> Dismissing…
                        </>
                      ) : (
                        'Dismiss'
                      )}
                    </button>
                    {confirmId === r.id ? (
                      <>
                        <button
                          type="button"
                          disabled={resolve.isPending}
                          onClick={() => run(r.id, 'deactivate', 'User deactivated.')}
                          className={dangerBtnCls}
                        >
                          {pendingId === r.id
                            ? 'Deactivating…'
                            : `Yes, deactivate ${r.target_name}`}
                        </button>
                        <button
                          type="button"
                          disabled={resolve.isPending}
                          onClick={() => setConfirmId(null)}
                          className={ghostBtnCls}
                        >
                          Keep
                        </button>
                      </>
                    ) : (
                      <button
                        type="button"
                        disabled={resolve.isPending}
                        onClick={() => setConfirmId(r.id)}
                        className={dangerBtnCls}
                      >
                        Deactivate user
                      </button>
                    )}
                  </div>
                )}
              </li>
            ))}
          </ul>
        </Card>
      </div>
      {toast && <Toast toast={toast} onClose={() => setToast(null)} />}
    </div>
  )
}
