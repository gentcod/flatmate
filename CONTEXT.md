# CONTEXT.md — Flatmate Matching Platform (hackathon build brief)

Read `docs/flatmate.md` for product background. This file is the build contract. Follow it imperatively.

## 1. Rules of engagement
- Generate and optimize code only. Do NOT run app-related commands (`go build|run|test|vet`, `make`, binary, sqlite writes).
- Exception: you MAY create/edit `Makefile`; never invoke its targets.
- Request user approval before any execution; user runs approved `make` targets.
- SQL lives in `sql/queries/*.sql`; Go uses generated `internal/database`. Never inline SQL in handlers.
- Keep changes minimal, stdlib-first, typed, commented only where non-obvious.
- Update this file when a decision changes; note conflicts with spec explicitly.

## 2. Hackathon constraints
- Due Nov 4 9:00 PM ET; target submit midday Nov 4. Today Oct 29.
- Demo must work end to end in ~3 min, with AI on and off.
- Judged on creativity, technical ambition, whether it works.
- Optimize for demo reliability over breadth; cut per §12 when behind.

## 3. Product in brief + code principles
- Goal: help people judge flatmate compatibility before committing.
- Practical (budget/area/dates) feeds fit score; lifestyle ranks; explanations show matches AND mismatches.
- Scores advise; users decide. Never imply "verified" means trustworthy.
- Explain every recommendation in plain checkable language. Collect minimal personal data.

## 4. Scope
- In: signup/login, profile, conversational onboarding, eligibility filter, weighted scoring + breakdown, match list/detail + discussion prompts, save/dismiss/interest, mutual-interest contact reveal, block/report, 25–30 seeds.
- Out: payments, identity verification, real-time chat (user ideal but deferred for hackathon; contact reveal only), landlord/agent tools, listings, rent/lease, ML ranking, analytics, multi-currency/locale logic.

## 5. Architecture + Muse boundary (hard rules)
- Matching engine is pure Go: no network, no LLM, no clock reads; fully unit-tested.
- Muse MUST NEVER rank, filter, or score. Allowed only: (a) extract onboarding answers, (b) narrate a computed breakdown, (c) draft questions for flagged differences, (d) stretch: normalize messy area/listing text.
- Send model only first name, area text, answers. Never email/phone/exact address or precise coords.
- All model output is untrusted: schema-validate, retry once, fall back. Demo never dead-ends on model error.
- `Breakdown` struct (§10) is the sole engine→UI→Muse contract.

## 6. Muse touchpoints
- T1 onboard-extract. In: `{transcript, questions}`. Out: `{"answers":[{"dimension":"sleep","value":"late","hard":false,"weight":2}]}`. Rules: map only to §10 dimensions; confirm in UI before save; never infer traits.
- T2 explain-breakdown. In: `{first_name, breakdown}`. Out: `{"bullets":["..."],"discuss":["..."]}`. Rules: restate computed scores only; never invent scores or rankings.
- T3 discuss-questions. In: `{differences:["guests"]}`. Out: `{"questions":["..."]}`. Rules: neutral tone, one question per difference, no judgment.
- T4 normalize-area. In: `{raw:"self-con, BQ available"}`. Out: `{"area_text":"...","note":"..."}`. Rules: preserve user wording in `note`; never geocode to exact address. Wired into profile save (3s timeout, rate-limited); failures keep user text.
- T5 interview-step. In: `{transcript, covered:[...]}`. Out: `{"done":false,"question":"...","answers":[...]}` (answers cumulative). Rules: same dimension/value rules as T1; exactly one short follow-up unless done; done when all covered or user stops.
- T6 suggest-bridges. In: `{differences:["sleep"]}`. Out: `{"bridges":[{"dimension":"sleep","suggestion":"..."}]}`. Rules: one practical compromise each, names its difference, neutral, no invented facts.
- T7 generate-questions. In: `{dimensions:[...]}`. Out: `{"questions":[{"dimension":"...","variants":["...","..."]}]}`. Rules: natural one-sentence scenarios answerable by exactly the listed options; never name the dimension/scale. Applies to cleanliness, noise, guests, privacy, social, disagreements; sleep/expenses/smoking/pets/wfh/employment stay direct. Caller picks one random variant per load; any failure falls back to labels.

## 7. Resilience + fallbacks
- Every model call: validate → retry once on invalid/timeout → deterministic template/form.
- Templates live in code (`internal/ai/fallback.go`); plain form always available for T1.
- Never block navigation on model latency; render page with template text first.
```go
type LLM interface {
  CompleteRaw(ctx context.Context, prompt, schema string) (json.RawMessage, error)
}
func CompleteJSON[T any](ctx context.Context, llm LLM, prompt, schema string, validate func(T) error) (T, error)
```

## 8. Stack + repo layout
- Go 1.23+, stdlib `net/http` ServeMux; `modernc.org/sqlite`; `html/template`+htmx; paseto+cookie; `weights.yaml`; sqlc queries in `sql/queries` → `internal/database`; env via `internal/util`.
```
cmd/server/main.go  internal/{auth,profile,match,ai,web,db,database,util}/
web/{templates,static}/  sql/queries/*.sql  migrations/*.sql
weights.yaml  seeds/global.json  Makefile  sqlc.yaml  CONTEXT.md  docs/flatmate.md
```

## 9. Data model (SQLite, key columns)
- `users(id, email_hash, pass_hash, created_at, deactivated_at)`
- `profiles(user_id PK, first_name, mode, gender, gender_pref, occupation, area_text, lat, lng, relocating BOOL, budget_min, budget_max, currency, move_earliest, move_latest DERIVED as move_earliest + duration_months, duration_months 1–36, bio, contact_handle TEXT free-text phone/email/WhatsApp)`
- `onboarding_drafts(user_id PK, description, drafts JSON, created_at, updated_at)` — saved AI description flow state, shown back on onboarding/questionnaire for reference
- `answers(user_id, dimension, value, hard BOOL, weight INT, PRIMARY KEY(user_id, dimension))`
- `actions(actor, target, kind, created_at)` kind ∈ save|dismiss|interest; mutual interest = both interest.
- `blocks(blocker, blocked)` + `reports(reporter, target, reason, created_at)`; blocked pairs never match.

## 10. Matching spec
| dimension | type | scale | compare |
|---|---|---|---|
| cleanliness | scale | 1–5 | closeness |
| noise | scale | 1–5 | tolerance overlap |
| sleep | choice | early/flex/late | exact=1 else flex-bridge |
| guests | scale | 1–5 | closeness; large gap → discuss |
| privacy | scale | 1–5 | closeness |
| social | choice | quiet/mixed/social | mismatch → discuss, partial penalty |
| expenses | choice | equal/itemized/mixed | exact=1 else mixed-bridge |
| disagreements | choice | talk/cooldown/write | mismatch → discuss, no heavy penalty |
| smoking | choice | no/outside/yes | hard-only gate, else score |
| pets | choice | none/ok/have | hard-only gate, else score |
| wfh | scale | 0–5 days | overlap |
| employment | choice | employed/self-employed/student/unemployed | exact=1 else discuss, no bridge |
- Eligibility removes ONLY: self, either-direction blocks, violations of explicit `hard=true` on ANY dimension (all 12 may be hard), and mutual gender-preference mismatch (each side's `gender_pref` anyone|women|men|nonbinary must include the other's `gender`; anyone matches all, unset gender matches only anyone). All mode cross-pairs eligible. Mode, budget, dates, area NEVER auto-block (per user answers; conflicts with spec §7 auto-gates — answers win). Gender is never scored.
- Scoring: per-dimension 0–100 → group means (practical: budget/area/dates; lifestyle: cleanliness/noise/sleep/guests/privacy/social/wfh; interpersonal: expenses/disagreements) + smoking/pets as lifestyle modifiers → weighted 0–100 from `weights.yaml` (`practical,lifestyle,interpersonal`). Move windows derive from `move_earliest` + `duration_months` (the until-date is no longer user input). Compatibility≠similarity: flagged gaps cap dimension at 60 and emit `discuss`. Relocating users match on `area_text`; coords are bonus only, never a filter.

```go
type DimensionResult struct { Name string; Score int; Weight float64; Aligned, Discuss bool; Note string }
type Breakdown struct { Total, Practical, Lifestyle, Interpersonal int; Dimensions []DimensionResult; Alignments, Differences, Discuss []string }
```

## 11. Safety + privacy minimums
- Show first name + general area only; no exact address; coords never rendered precisely.
- Reveal `contact_handle` (user-supplied phone/email/WhatsApp) only after mutual interest; log reveal in `actions`.
- Block/report on every profile; deactivation hides profile immediately.
- State "unverified" plainly; never show unverified as trusted.
- CSRF: double-submit cookie + hidden form token on every POST (see `internal/auth`); failures 403.
- Rate limits (in-memory, per-IP login/signup 10/min, per-user AI routes 30/min): auth failures re-render with a message; AI limits degrade to non-AI output, never 429.
- Cookies HttpOnly + SameSite=Lax always; Secure unless loopback (override `FLATMATE_COOKIE_SECURE=0/1`).
- Refuse to boot without `FLATMATE_SESSION_KEY` except on loopback, where ephemeral keys warn.
- Interview state is in-memory (2h idle TTL) plus the persisted transcript/drafts row; a restart only restarts interviews.

## 12. Build order + cuts
- Oct 29: this file + scaffold + migrations + `weights.yaml`.
- Oct 30: auth/profile/questionnaire form baseline (no AI).
- Oct 31: engine + unit tests (generated, not run).
- Nov 1: match list/detail + actions + safety.
- Nov 2: Muse touchpoints + fallbacks.
- Nov 3: seeds + `Makefile` + polish.
- Nov 4: demo rehearse, submit midday.
- Cut first: T4 stretch, conversational polish, extra seeds. Never cut eligibility/scoring/explanation/reveal/block.

## 13. Definition of done + 3-min demo
- Done: fresh DB → seed → signup → onboard → ≥1 eligible match with explanation → interest → mutual reveal → block works; no model call needed to complete.
- Demo (180s): 0:00 signup+onboard (seeded user) → 0:45 match list → 1:15 detail + explanation + discuss → 2:00 interest → 2:20 mutual reveal → 2:40 block/report + fallback note.

## 14. Open items (developer fills)
- DONE: Meta Model API endpoint URL = `POST https://api.meta.ai/v1/responses` (Responses API; see `internal/ai/http.go`).
- DONE: auth method + credential placement = `Authorization: Bearer` header with `FLATMATE_MODEL_TOKEN` from `.env` (never logged, never sent to clients).
- DONE: model name + version pin = `muse-wonder-1.0` via `FLATMATE_MODEL_NAME`.
- DONE: request parameters = temperature 0.2, max_output_tokens 8192, store=false; HTTP client timeout 30s; per-touchpoint ctx deadlines 12–25s.
- DONE: structured-output/JSON-schema support + validation path = `text.format` json_schema strict (schemas normalized by `enforceStrict`: additionalProperties=false, all properties required) + client-side semantic validators in `internal/ai/touchpoints.go`; `CompleteJSON` retries once, then deterministic fallback.
- OPEN: rate limits + retry/backoff policy + demo quota — check the dashboard / pricing-rate-limits page. Current posture (retry once, then template/form fallback, never block navigation) stands for the demo.
