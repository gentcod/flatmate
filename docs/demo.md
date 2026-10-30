# 3-Minute Demo Script

Setup: `make tools && make tidy && make demo`, then open http://localhost:8080.
All seed passwords are `Seedpass123!`.

## Beats (180s)

| Time | Action | Expected |
|---|---|---|
| 0:00 | Log in as `tunde@example.com` | Lands on Matches, ranked list |
| 0:30 | Open Adaeze (top match) | High score, "what lines up" list |
| 1:00 | Open Chiamaka | Lower score, guest/social differences flagged "worth discussing" |
| 1:30 | Express interest in Adaeze; log in as `adaeze@example.com` in a second browser, express interest back | Mutual interest reveals contact on both sides |
| 2:10 | Dismiss one match, block another | Both vanish from the list immediately |
| 2:30 | Questionnaire → "describe your habits in words" (needs model env, else form fallback shows) | AI draft or graceful plain form |
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
