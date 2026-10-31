import { useState } from 'react'
import { Link, useLocation } from 'react-router-dom'
import {
  Card,
  ErrorNote,
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
    <fieldset className="border-t border-stone-200 pt-4 first:border-t-0 first:pt-0">
      <legend className="px-1 text-sm font-bold text-stone-900">{dim.label}</legend>
      {dim.question ? (
        <p className="mt-1 border-l-4 border-emerald-700 bg-emerald-50 px-3 py-2 text-sm text-stone-800">
          {dim.question}
        </p>
      ) : (
        <p className="mt-1 text-xs text-stone-600">{dim.hint}</p>
      )}
      {dim.question && <p className="mt-1 text-xs text-stone-600">{dim.hint}</p>}
      <div className="mt-3 flex flex-wrap gap-2">
        {dim.options.map((opt) => {
          const selected = row.value === opt
          return (
            <button
              key={opt}
              type="button"
              aria-pressed={selected}
              onClick={() => onChange({ value: opt })}
              className={`rounded-full border px-3 py-1.5 text-sm font-medium ${
                selected
                  ? 'border-emerald-800 bg-emerald-800 text-white'
                  : 'border-stone-300 bg-white text-stone-800 hover:bg-stone-100'
              }`}
            >
              {opt}
            </button>
          )
        })}
      </div>
      <div className="mt-3 flex flex-wrap items-center gap-x-6 gap-y-2">
        <span className="text-xs font-semibold text-stone-700">
          How much does this matter?
          <span className="ml-2 inline-flex gap-1">
            {WEIGHTS.map((w) => (
              <button
                key={w.value}
                type="button"
                title={w.label}
                aria-pressed={row.weight === w.value}
                onClick={() => onChange({ weight: w.value })}
                className={`rounded-md border px-2 py-1 text-xs font-semibold ${
                  row.weight === w.value
                    ? 'border-emerald-800 bg-emerald-800 text-white'
                    : 'border-stone-300 bg-white text-stone-700 hover:bg-stone-100'
                }`}
              >
                {w.value}
              </button>
            ))}
          </span>
        </span>
        <label className="flex items-center gap-2 text-xs font-medium text-stone-700">
          <input
            type="checkbox"
            checked={row.hard}
            onChange={(e) => onChange({ hard: e.target.checked })}
            className="h-4 w-4 accent-emerald-800"
          />
          Must match exactly (dealbreaker)
        </label>
      </div>
    </fieldset>
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
      {(seedMessage || notice) && (
        <div className="mb-4">
          <Notice message={seedMessage || notice || ''} />
        </div>
      )}
      {description && (
        <details className="mb-4 rounded-lg border border-stone-200 bg-white px-4 py-2 text-sm">
          <summary className="cursor-pointer font-semibold text-stone-800">
            Your saved description (from chat setup)
          </summary>
          <p className="mt-2 whitespace-pre-line text-stone-700">{description}</p>
        </details>
      )}
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
      <div className="mt-4 flex flex-wrap items-center gap-3">
        <button type="button" onClick={onSave} disabled={save.isPending} className={primaryBtnCls}>
          {save.isPending ? 'Saving…' : 'Save answers'}
        </button>
        <Link to="/matches" className="text-sm font-semibold text-emerald-800 hover:underline">
          View matches
        </Link>
      </div>
    </>
  )
}

export function QuestionnairePage() {
  const query = useQuestionnaire()
  const location = useLocation()
  const state = (location.state ?? {}) as SeedState

  if (query.isPending) return <Loading label="Loading the questionnaire…" />
  if (query.isError) return <ErrorNote message={errorMessage(query.error)} />
  return (
    <>
      <PageTitle
        title="Living-habits questionnaire"
        sub="Some prompts are drafted fresh each visit — the stored answers never change meaning."
      />
      <QuestionnaireForm
        dims={query.data.dims}
        description={query.data.description}
        notice={query.data.notice}
        seed={state.seed}
        seedMessage={state.message}
      />
    </>
  )
}
