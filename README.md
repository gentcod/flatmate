# Flatmate Matching Platform

Find flatmates you can actually live with — matched on compatibility,
not just budget and location.

## The idea

Budget and area get you a shortlist; cleanliness, noise, guests, routines,
and conflict style decide whether sharing a home works. This app makes that
judgment structured: answer questions across 12 living-habit dimensions, and
get ranked matches with plain-language explanations of what lines up, what
differs, and what's worth discussing before moving in.

## How it works

1. **Filter** — hard requirements (deal-breakers, blocks) remove non-starters.
2. **Score** — a pure rule-based engine ranks the rest 0–100 on practical,
   lifestyle, and interpersonal fit. Weights live in `weights.yaml`.
3. **Explain** — every match shows reasons and discussion prompts. AI only
   narrates precomputed scores; it never ranks or filters, and the app works
   fully with AI disabled.

## What's inside

Signup/login, profiles, questionnaire (form or AI-drafted), ranked matches,
save/dismiss/express interest, mutual-interest contact reveal, block/report,
room listings, mutual-interest chat, flatmate reviews, expense tracking,
agreement drafting, natural-language search, admin moderation, metrics,
verified stays with trust-scored reviews, alerts for interests/stays/reviews, a public landing page,
and 27 seed profiles. Single Go binary, SQLite, JSON API plus two UIs:
the legacy server-rendered pages and the React SPA in `client/`.

## Run it

```sh
make test                               # unit tests (recommended first run)
make serve                              # tidy, build, seed, run: legacy UI at http://localhost:8080
```

SPA dev (needs Node 22+):

```sh
make client-install                     # once: install client dependencies
make serve                              # Go API on :8080 (terminal 1)
cd client && npm run dev                # SPA at http://localhost:5173 (terminal 2)
```

SPA production: `make client-setup`, then `FLATMATE_UI=spa make serve` —
Go serves `client/dist` itself on `:8080`.

PWA (installable): icons generate from `client/public/icon.svg` as part of
`make client-setup`. The app precaches its shell for offline page views
(saving still needs a connection) and prompts when a new version is ready.
Re-run `make pwa-assets` if you change the icon.

Ops: `make backup` snapshots the DB into `backups/`;
`make promote EMAIL=x@y.z` / `make demote EMAIL=x@y.z` manage admins,
who triage reports at `/admin` (SPA) or `/admin/reports` (legacy).

Config comes from a `.env` file or the environment (see `.env.example`).
Seed logins use password `Seedpass123!` (e.g. `tunde@example.com`).
Demo script: [docs/demo.md](docs/demo.md). Build contract: [CONTEXT.md](CONTEXT.md).
Design system: [docs/designs](docs/designs) (Warm Editorial — Newsreader + Plus Jakarta Sans).
