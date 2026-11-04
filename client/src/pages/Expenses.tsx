import { useState } from 'react'
import type { FormEvent } from 'react'
import { Link, useParams } from 'react-router-dom'
import {
  Card,
  ErrorNote,
  Field,
  Loading,
  PageTitle,
  Toast,
  errorMessage,
  dangerBtnCls,
  ghostBtnCls,
  inputCls,
  primaryBtnCls,
} from '../components'
import { useAddExpense, useDeleteExpense, useLedger } from '../query_social'
import type { ToastData } from '../components'

const SPLITS = [
  { value: 'equal', label: 'Split equally' },
  { value: 'mine', label: 'Payer covers all' },
  { value: 'theirs', label: 'Other covers all' },
  { value: 'custom', label: 'Custom shares' },
]

function AddForm({ peer, onAdded }: { peer: number; onAdded: () => void }) {
  const [amount, setAmount] = useState('')
  const [currency, setCurrency] = useState('USD')
  const [note, setNote] = useState('')
  const [split, setSplit] = useState('equal')
  const [payer, setPayer] = useState<'me' | 'them'>('me')
  const [payerShare, setPayerShare] = useState('')
  const [peerShare, setPeerShare] = useState('')
  const [formError, setFormError] = useState('')
  const add = useAddExpense(peer)

  const onSubmit = (e: FormEvent) => {
    e.preventDefault()
    setFormError('')
    const amountN = Number(amount)
    if (amount.trim() === '' || !Number.isInteger(amountN) || amountN <= 0) {
      setFormError('Amount must be a plain number above 0.')
      return
    }
    const ps = payerShare.trim() === '' ? 0 : Number(payerShare)
    const qs = peerShare.trim() === '' ? 0 : Number(peerShare)
    if (split === 'custom' && (!Number.isInteger(ps) || !Number.isInteger(qs) || ps < 0 || qs < 0 || ps + qs !== amountN)) {
      setFormError('Custom shares must be plain numbers that sum to the amount.')
      return
    }
    add.mutate(
      {
        amount: amountN,
        currency: currency.trim().toUpperCase() || 'USD',
        note: note.trim(),
        split,
        payer,
        payer_share: ps,
        peer_share: qs,
      },
      {
        onSuccess: () => {
          setAmount('')
          setNote('')
          setPayerShare('')
          setPeerShare('')
          onAdded()
        },
      },
    )
  }

  const error = formError || (add.isError ? errorMessage(add.error) : '')

  return (
    <form onSubmit={onSubmit} className="flex flex-col gap-4">
      <div className="grid grid-cols-2 gap-4">
        <Field label="Amount">
          <input
            required
            inputMode="numeric"
            value={amount}
            onChange={(e) => setAmount(e.target.value)}
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
      <Field label="Note" hint="e.g. electricity bill. Up to 200 characters.">
        <input
          maxLength={200}
          value={note}
          onChange={(e) => setNote(e.target.value)}
          className={inputCls}
        />
      </Field>
      <div className="grid grid-cols-2 gap-4">
        <Field label="Who paid?">
          <select value={payer} onChange={(e) => setPayer(e.target.value as 'me' | 'them')} className={inputCls}>
            <option value="me">I paid</option>
            <option value="them">They paid</option>
          </select>
        </Field>
        <Field label="Split">
          <select value={split} onChange={(e) => setSplit(e.target.value)} className={inputCls}>
            {SPLITS.map((s) => (
              <option key={s.value} value={s.value}>
                {s.label}
              </option>
            ))}
          </select>
        </Field>
      </div>
      {split === 'custom' && (
        <div className="grid grid-cols-2 gap-4">
          <Field label="Payer share" hint="Must sum to the amount.">
            <input
              inputMode="numeric"
              value={payerShare}
              onChange={(e) => setPayerShare(e.target.value)}
              className={inputCls}
            />
          </Field>
          <Field label="Other share">
            <input
              inputMode="numeric"
              value={peerShare}
              onChange={(e) => setPeerShare(e.target.value)}
              className={inputCls}
            />
          </Field>
        </div>
      )}
      {error && <ErrorNote message={error} />}
      <div>
        <button type="submit" disabled={add.isPending} className={primaryBtnCls}>
          {add.isPending ? 'Adding…' : 'Add expense'}
        </button>
      </div>
    </form>
  )
}

export function ExpensesPage() {
  const params = useParams()
  const peer = Number(params.id)
  const valid = Number.isInteger(peer) && peer > 0
  const ledger = useLedger(valid ? peer : 0, valid)
  const del = useDeleteExpense(valid ? peer : 0)
  const [showForm, setShowForm] = useState(false)
  const [toast, setToast] = useState<ToastData | null>(null)
  const [confirmDel, setConfirmDel] = useState<number | null>(null)
  const [delId, setDelId] = useState<number | null>(null)

  if (!valid) return <ErrorNote message="Not found." />
  if (ledger.isPending) return <Loading label="Loading shared expenses…" />
  if (ledger.isError) return <ErrorNote message={errorMessage(ledger.error)} />

  const { peer: peerInfo, expenses, balances } = ledger.data

  return (
    <>
      <Link
        to={`/chat/${peerInfo.id}`}
        className="mb-4 inline-block text-sm font-semibold text-sage hover:text-ink hover:underline"
      >
        ← Back to chat
      </Link>
      <div className="flex flex-wrap items-end justify-between gap-3">
        <PageTitle
          title={`Shared expenses with ${peerInfo.first_name}`}
          sub="Tracking only — settle up however suits you both."
        />
        <button
          type="button"
          onClick={() => setShowForm((v) => !v)}
          className={`${ghostBtnCls} mb-6`}
        >
          {showForm ? 'Close' : 'Add expense'}
        </button>
      </div>
      {showForm && (
        <div className="mb-4">
          <Card>
            <AddForm
              peer={peerInfo.id}
              onAdded={() => {
                setShowForm(false)
                setToast({ id: Date.now(), message: 'Expense added.', tone: 'ok' })
              }}
            />
          </Card>
        </div>
      )}
      <Card>
        <h2 className="font-display text-2xl tracking-tight text-ink">Balance</h2>
        {balances.length === 0 ? (
          <p className="mt-1 text-sm text-sage">Nothing shared yet.</p>
        ) : (
          <ul className="mt-1 flex flex-col gap-1 text-sm text-sage">
            {balances.map((b) => (
              <li key={b.currency}>
                <span className="font-semibold">{b.currency}:</span>{' '}
                {b.amount > 0 ? (
                  <span className="tabular-nums">
                    they owe you {b.amount.toLocaleString()}
                  </span>
                ) : b.amount < 0 ? (
                  <span className="tabular-nums">you owe {Math.abs(b.amount).toLocaleString()}</span>
                ) : (
                  'settled'
                )}
              </li>
            ))}
          </ul>
        )}
      </Card>
      <div className="mt-4 flex flex-col gap-3">
        {expenses.length === 0 ? (
          <Card>
            <p className="text-[15px] text-sage">No shared expenses yet.</p>
          </Card>
        ) : (
          expenses.map((e) => (
            <Card key={e.id}>
              <div className="flex items-start justify-between gap-3">
                <div>
                  <p className="text-sm text-ink">
                    <span className="font-bold tabular-nums">
                      {e.currency} {e.amount.toLocaleString()}
                    </span>{' '}
                    <span className="text-sage">
                      — {e.paid_by_me ? 'you paid' : `${peerInfo.first_name} paid`} (
                      {SPLITS.find((s) => s.value === e.split)?.label ?? e.split})
                    </span>
                  </p>
                  {e.note && <p className="mt-1 text-sm text-sage">{e.note}</p>}
                  <p className="mt-1 text-xs text-fog">
                    {e.owed_to_me > 0
                      ? `owes you ${e.owed_to_me.toLocaleString()}`
                      : e.owed_by_me > 0
                        ? `you owe ${e.owed_by_me.toLocaleString()}`
                        : 'settled'}
                  </p>
                </div>
                <button
                  type="button"
                  disabled={del.isPending}
                  onClick={() => {
                    if (confirmDel !== e.id) {
                      setConfirmDel(e.id)
                      return
                    }
                    setConfirmDel(null)
                    setDelId(e.id)
                    del.mutate(e.id, {
                      onSuccess: () =>
                        setToast({ id: Date.now(), message: 'Expense deleted.', tone: 'ok' }),
                      onError: (er) =>
                        setToast({ id: Date.now(), message: errorMessage(er), tone: 'err' }),
                      onSettled: () => setDelId(null),
                    })
                  }}
                  className={`${dangerBtnCls} px-3 py-1 text-xs`}
                >
                  {confirmDel === e.id ? (delId === e.id ? 'Deleting…' : 'Confirm?') : 'Delete'}
                </button>
              </div>
            </Card>
          ))
        )}
      </div>
      {del.isError && (
        <div className="mt-3">
          <ErrorNote message={errorMessage(del.error)} />
        </div>
      )}
      {toast && <Toast toast={toast} onClose={() => setToast(null)} />}
    </>
  )
}
