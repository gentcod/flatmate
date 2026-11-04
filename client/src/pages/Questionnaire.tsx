import { useState } from 'react'
import { Link, useLocation } from 'react-router-dom'
import {
  Card,
  ErrorNote,
  Icon,
  Loading,
  Notice,
  PageTitle,
  errorMessage,
  primaryBtnCls,
} from '../components'
import type { Answer, QDim } from '../api'
import { useQuestionnaire, useSaveAnswers } from '../query'

interface SeedState {
  seed?: Record<string, Answer>
  message?: string
}

interface Row {
  value: string
  hard: boolean
  weight: number
}

type Rows = Record<string, Row>

const WEIGHTS = [
  { value: 1, label: 'Matters little' },
  { value: 2, label: 'Matters' },
  { value: 3, label: 'Matters a lot' },
]

function DimRow({
  dim,
  row,
  onChange,
}: {
  dim: QDim
  row: Row
  onChange: (patch: Partial<Row>) => void
}) {
  return (
    <fieldset className="border-t border-sand pt-4 first:border-t-0 first:pt-0">
      <legend className="px-1 text-[17px] font-semibold text-ink">{dim.label}</legend>
      {dim.question ? (
        <p className="mt-1 rounded-xl bg-parchment px-4 py-3 font-display text-[17px] italic leading-snug text-ink">
          {dim.question}
        </p>
      ) : (
        <p className="mt-1 text-xs text-fog">{dim.hint}</p>
      )}
      {dim.question && <p className="mt-1 text-xs text-fog">{dim.hint}</p>}
      <div className="mt-3 flex flex-wrap gap-2">
        {dim.options.map((opt) => {
          const selected = row.value === opt
          return (
            <button
              key={opt}
              type="button"
              aria-pressed={selected}
              onClick={() => onChange({ value: opt })}
              className={`cursor-pointer rounded-full border px-3 py-1.5 text-sm font-medium ${
                selected
                  ? 'border-eucalyptus bg-eucalyptus text-white'
                  : 'border-sand bg-white text-ink hover:bg-parchment'
              }`}
            >
              {opt}
            </button>
          )
        })}
      </div>
      <div className="mt-3 flex flex-wrap items-center gap-x-6 gap-y-2">
        <span className="text-xs font-semibold text-ink">
          How much does this matter?
          <span className="ml-2 inline-flex gap-1">
            {WEIGHTS.map((w) => (
              <button
                key={w.value}
                type="button"
                title={w.label}
                aria-pressed={row.weight === w.value}
                onClick={() => onChange({ weight: w.value })}
                className={`cursor-pointer rounded-md border px-2 py-1 text-xs font-semibold ${
                  row.weight === w.value
                    ? 'border-eucalyptus bg-eucalyptus text-white'
                    : 'border-sand bg-white text-sage hover:bg-parchment'
                }`}
              >
                {w.value}
              </button>
            ))}
          </span>
        </span>
        <label className="flex items-center gap-2 text-xs font-medium text-sage">
          <input
            type="checkbox"
            checked={row.hard}
            onChange={(e) => onChange({ hard: e.target.checked })}
            className="h-5 w-5 accent-eucalyptus"
          />
          Must match exactly (dealbreaker)
        </label>
      </div>
    </fieldset>
  )
}

function CalibrationRing({ done, total }: { done: number; total: number }) {
  const pct = total === 0 ? 0 : Math.round((done / total) * 100)
  return (
    <span className="relative flex h-10 w-10 items-center justify-center">
      <svg className="h-10 w-10 -rotate-90" viewBox="0 0 36 36">
        <path
          d="M18 2.0845 a 15.9155 15.9155 0 0 1 0 31.831 a 15.9155 15.9155 0 0 1 0 -31.831"
          fill="none"
          stroke="#e5dfd5"
          strokeWidth="3"
        />
        <path
          d="M18 2.0845 a 15.9155 15.9155 0 0 1 0 31.831 a 15.9155 15.9155 0 0 1 0 -31.831"
          fill="none"
          stroke="#1e3f35"
          strokeWidth="3"
          strokeLinecap="round"
          strokeDasharray={`${pct}, 100`}
        />
      </svg>
      <span className="absolute text-[11px] font-bold text-eucalyptus">{pct}%</span>
    </span>
  )
}

function QuestionnaireForm({
  dims,
  description,
  notice,
  seed,
  seedMessage,
}: {
  dims: QDim[]
  description: string
  notice?: string
  seed?: Record<string, Answer>
  seedMessage?: string
}) {
  const [rows, setRows] = useState<Rows>(() => {
    const init: Rows = {}
    for (const d of dims) {
      const s = seed?.[d.dim]
      init[d.dim] = {
        value: s?.value ?? d.value,
        hard: s?.hard ?? d.hard,
        weight: s?.weight ?? d.weight,
      }
    }
    return init
  })
  const [formError, setFormError] = useState('')
  const [saved, setSaved] = useState(false)
  const save = useSaveAnswers()

  const setRow = (dim: string, patch: Partial<Row>) =>
    setRows((prev) => ({ ...prev, [dim]: { ...prev[dim], ...patch } }))

  const onSave = () => {
    setFormError('')
    setSaved(false)
    const missing = dims.find((d) => rows[d.dim].value === '')
    if (missing) {
      setFormError(`Answer every question before saving — "${missing.label}" is blank.`)
      return
    }
    const answers: Answer[] = dims.map((d) => ({
      dimension: d.dim,
      value: rows[d.dim].value,
      hard: rows[d.dim].hard,
      weight: rows[d.dim].weight,
    }))
    save.mutate(answers, { onSuccess: () => setSaved(true) })
  }

  const error = formError || (save.isError ? errorMessage(save.error) : '')

  return (
    <>
      <div className="mb-8 grid grid-cols-1 gap-4 md:grid-cols-2">
        <div className="relative flex flex-col justify-between gap-4 rounded-2xl border border-sand bg-white p-5 shadow-card">
          <div className="flex items-start justify-between">
            <span className="flex h-9 w-9 items-center justify-center rounded-xl bg-eucalyptus text-white">
              <Icon name="checklist" />
            </span>
            <span className="rounded-full bg-eucalyptus px-2.5 py-0.5 text-[11px] font-medium tracking-wide text-white">
              Active method
            </span>
          </div>
          <div>
            <h2 className="text-[17px] font-semibold text-ink">Plain 12-dimension form</h2>
            <p className="mt-1 text-sm text-sage">
              Direct options with AI-drafted scenarios inline on 6 dimensions. Zero pressure, edit
              anytime.
            </p>
          </div>
        </div>
        <Link
          to="/onboarding"
          className="group flex flex-col justify-between gap-4 rounded-2xl border border-sand bg-white p-5 shadow-card transition-shadow hover:shadow-lift"
        >
          <div className="flex items-start justify-between">
            <span className="flex h-9 w-9 items-center justify-center rounded-xl bg-parchment text-sage transition-colors group-hover:text-eucalyptus">
              <Icon name="forum" />
            </span>
            <span className="rounded-full bg-parchment px-2 py-0.5 text-[11px] text-sage">
              5-min chat
            </span>
          </div>
          <div>
            <h2 className="text-[17px] font-semibold text-ink">Conversational interview</h2>
            <p className="mt-1 text-sm text-sage">
              Describe how you live in your own words — answers are drafted for your review.
            </p>
          </div>
          <span className="flex items-center text-sm text-sage group-hover:text-eucalyptus">
            Switch to chat
            <Icon name="arrow_forward" cls="ml-1 text-[16px]" />
          </span>
        </Link>
      </div>
      {(seedMessage || notice) && (
        <div className="mb-4">
          <Notice message={seedMessage || notice || ''} />
        </div>
      )}
      {description && (
        <details className="mb-4 rounded-xl border border-sand bg-white px-4 py-2 text-sm shadow-card">
          <summary className="cursor-pointer font-semibold text-ink">
            Your saved description (from chat setup)
          </summary>
          <p className="mt-2 whitespace-pre-line text-sage">{description}</p>
        </details>
      )}
      <div className="sticky top-24 z-30 mb-6 flex justify-end">
        <div className="flex items-center gap-3 rounded-2xl border border-sand bg-white px-4 py-3 shadow-card">
          <div className="flex flex-col">
            <span className="text-[13px] text-sage">Calibrated</span>
            <span className="text-[17px] font-semibold text-eucalyptus">
              {dims.filter((d) => rows[d.dim].value !== '').length} of {dims.length} set
            </span>
          </div>
          <CalibrationRing
            done={dims.filter((d) => rows[d.dim].value !== '').length}
            total={dims.length}
          />
        </div>
      </div>
      <Card>
        <div className="flex flex-col gap-6">
          {dims.map((d) => (
            <DimRow key={d.dim} dim={d} row={rows[d.dim]} onChange={(p) => setRow(d.dim, p)} />
          ))}
        </div>
      </Card>
      {error && (
        <div className="mt-4">
          <ErrorNote message={error} />
        </div>
      )}
      {saved && (
        <div className="mt-4">
          <Notice message="Saved. Your matches are up to date." />
        </div>
      )}
      <div className="sticky bottom-4 z-30 mt-6 flex flex-wrap items-center justify-between gap-3 rounded-2xl border border-sand bg-white/95 p-5 shadow-modal backdrop-blur">
        <button type="button" onClick={onSave} disabled={save.isPending} className={`${primaryBtnCls} px-6 py-3`}>
          {save.isPending ? 'Saving…' : 'Save answers'}
        </button>
        <Link to="/matches" className="text-sm font-semibold text-eucalyptus hover:underline">
          View matches
        </Link>
      </div>
    </>
  )
}

export function QuestionnairePage() {
  const query = useQuestionnaire()
  const location = useLocation()
  const navState = (location.state ?? {}) as SeedState
  if (navState.seed) {
    try {
      sessionStorage.setItem(
        'qseed',
        JSON.stringify({ seed: navState.seed, message: navState.message ?? '' }),
      )
    } catch {
      /* private mode: seeds last until refresh */
    }
  }
  const seedKey = navState.seed ? JSON.stringify(navState.seed) : 'server'
  let state = navState
  if (!state.seed) {
    try {
      const raw = sessionStorage.getItem('qseed')
      if (raw) state = JSON.parse(raw) as SeedState
    } catch {
      /* corrupted entry: fall through to server values */
    }
  }

  if (query.isPending) return <Loading label="Loading the questionnaire…" />
  if (query.isError) return <ErrorNote message={errorMessage(query.error)} />
  return (
    <>
      <PageTitle
        title="Find your household rhythm"
        sub="Choose the path that fits how you express yourself. Every path calibrates the same 12 living compatibility dimensions."
      />
      <QuestionnaireForm
        key={seedKey}
        dims={query.data.dims}
        description={query.data.description}
        notice={query.data.notice}
        seed={state.seed}
        seedMessage={state.message}
      />
    </>
  )
}
