# 3-Minute Demo Script

Setup: `make tools && make tidy && make demo`, then open http://localhost:8080 (legacy UI).
All seed passwords are `Seedpass123!`.

SPA setup (needs Node 22+): `make client-install`, then with `make demo` running,
`cd client && npm run dev` and open http://localhost:5173.
SPA production: `make client-build`, then `FLATMATE_UI=spa make demo` on :8080.

## Beats (180s)

Same script on either UI; routes match (`/matches`, `/match/:id`, `/questionnaire`, `/onboarding`, `/settings`).

| Time | Action | Expected |
|---|---|---|
| 0:00 | Log in as `tunde@example.com` | Lands on Matches, ranked list |
| 0:20 | Filter by area "london", then clear | List narrows to London matches, then restores |
| 0:40 | Open Adaeze (top match) | High score, group bars, top strengths, per-dimension table |
| 1:10 | Open Chiamaka | Lower score, guest/social differences flagged "worth discussing" |
| 1:40 | Express interest in Adaeze; log in as `adaeze@example.com` in a second browser, express interest back | Mutual interest reveals contact on both sides |
| 2:10 | Dismiss one match; as Adaeze hide "Visible to matches" (Settings on SPA, profile form on legacy), back as Tunde | Both vanish from the list immediately |
| 2:30 | Chat setup → answer one prompt (needs model env, else the plain form shows) | AI follow-up or graceful fallback; drafts land in Settings |
| 2:50 | Point at AI explanation section on a match detail (present only when the model responds) | Raw breakdown always visible below it |

## Key accounts

- `adaeze@example.com` — has room in Yaba, quiet, hard no-smoking.
- `tunde@example.com` — needs room in Yaba, quiet, strong match for Adaeze.
- `tobi@example.com` — social smoker (outside only); eligible for Adaeze via bridge but scores lower with flags.
- `femi@example.com` — smokes inside, has a dog; hard-gated out for Adaeze.
- `chiamaka@example.com` — relocating Abuja → Lekki; mismatch demo.
- `grace@example.com` / `yusuf@example.com` — expense-split clash demo.
- Seeds span Lagos, Nairobi, London, and New York; `tunde`/`adaeze` (Yaba) remain the scripted pair.

## AI on/off

- Off (default): no `FLATMATE_MODEL_ENDPOINT` → detail pages show the raw breakdown; onboarding falls back to the plain form. Nothing dead-ends.
- On: set `FLATMATE_MODEL_ENDPOINT`, `FLATMATE_MODEL_NAME`, `FLATMATE_MODEL_TOKEN` (see `TODO(fill)` in CONTEXT.md §14) → "In plain language" section appears on match details.
