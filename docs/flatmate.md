# Flatmate Matching Platform
## Simplified Product Reference

**Status:** Working depth slice — Go API + SQLite, React SPA, and legacy server-rendered UI in one monorepo. Demo-ready with AI on or off.
**Scope:** Global from day one — no single launch market. Seeds span Lagos, Nairobi, London, and New York.
**One-line goal:** Help people find flatmates they can actually live with, not just flatmates who fit a budget.

---

## 1. The Problem

People can find rooms and potential flatmates through friends, social media groups, and agents. What they can't easily do is tell whether they'd be **compatible** to live with before committing.

Choosing is the hard part. Budget and location are only the starting point. Cleanliness, noise, guests, routines, privacy, and how disagreements get handled decide whether a shared home works.

**Core problem:** There is no simple, structured way to judge whether two people are compatible enough to share a home.

## 2. The Solution

A platform that treats flatmate search as a **compatibility problem**:

1. Users say what they need (hard requirements) and what they prefer (flexible preferences).
2. The system removes people who don't fit the hard requirements.
3. It ranks the rest by how well their lifestyles line up.
4. It explains each match in plain language, including where the two people differ.
5. Users decide for themselves.

**Example:** Two people want the same area and have the same budget. One wants a quiet home with few visitors. The other hosts friends often. A budget-and-location search ranks them highly. This platform shows the clash up front and suggests what to discuss before they move in.

## 3. Principles

These guide every product and technical decision.

- **Practical first, lifestyle second.** Budget, location, and dates shape the score; hard requirements, blocks, and gender preferences decide eligibility. Lifestyle decides who ranks higher.
- **Compatible does not mean the same.** Show real alignment and real differences. Don't hide mismatches.
- **Users decide, the platform recommends.** Scores inform; they never replace judgment.
- **Be honest about trust.** Say exactly what has been verified, and never imply "verified" means "trustworthy".
- **Keep it explainable.** Every recommendation must come with reasons a person can read and check.
- **Collect less.** Gather only the personal data needed to match, and protect it.

## 4. Who It's For

**Primary users**
- People looking for a flatmate because they already have a room.
- People looking for a room to share.
- Two or more people looking to find accommodation together.

Landlords, agents, and housing providers are out of focus for now. Value for flatmate seekers comes first.

## 5. MVP Scope

The MVP tests one idea: **people will use structured compatibility matching instead of relying only on social media groups and word of mouth.**

**Included**
- Sign up and log in (secure cookies, rate-limited, CSRF-protected).
- Profile: arrangement mode, preferred area (+ optional relocating flag and coordinates), budget and currency, move-in date plus stay duration in months, gender and mutual gender preference, occupation, short bio, contact handle.
- Searching visibility: on by default; hide yourself once you've found a roommate and you disappear from lists and detail pages.
- Lifestyle questionnaire (see section 6): plain form, AI-drafted scenario prompts, or a capped conversational interview with skip-to-form fallback.
- Eligibility filtering on hard requirements, blocks, and gender preferences.
- Ranked matches with full score transparency: total, three group scores, top strengths, and a dimension-by-dimension table.
- Save, dismiss, and express interest. Mutual interest reveals contact details.
- Block and report; account deactivation.
- Room listings with area filter, mutual-interest chat, and one-per-pair flatmate reviews gated on verified stays.
- Listing photos via URL (seeded rooms use built-in placeholder artwork; uploads later).
- Listing pages connect with the owner in place: show interest, then chat, contact reveal, and agreement drafting on mutual interest.
- Decide/live-together tools: shared expense tracker with running balances, an AI agreement drafter (template fallback, not legal advice), natural-language match search, saved searches, affordability checks on listings, and self-confirmed contact labels.
- Trust-ops: admin report queue (dismiss or deactivate) and metrics in both UIs, per-request logging, `make backup` snapshots, CLI-only admin promotion.
- Verified stays: mutual handshake (propose/confirm) proves two users rented together; flatmate reviews require a verified stay and carry a verified tag. Listings take stay reviews (verified tenants) and visit reviews (anyone), shown in separate views with a trust score weighting tenants 3-to-1. Accepting a stay auto-marks its listing rented; owners can re-list.
- Installable PWA: precached app shell with an honest offline notice, update prompt, install icons — API calls always go to the network, never the cache.
- Action feedback everywhere: blocking loading modal for slow processes (agreement drafts, interview, auth), button spinners plus toast verdicts for fast actions (interest, save, dismiss, chat), hand pointer on all clickables, lazy-loaded routes and listing photos. Legacy forms show a working overlay and lock their submit buttons.
- Public landing page at `/` for logged-out visitors (both UIs): hero with habit-search teaser, example match and room previews, the Flatmate contrast, three pillars, an interactive scenario sandbox (SPA), trust banner, and signup CTAs. Signed-in users still land on matches. All previews are labeled examples; no fake stats, ID badges, or hotlinked photos.
- Two clients on one JSON API: the React SPA and the legacy server-rendered pages.

**Not included yet**
Payments, identity verification, landlord or agent tools, machine-learning ranking, and personality analysis. (The agreement drafter and expense tracker are informal helpers, not legal or payment tools.)

## 6. The Questionnaire

Questions ask about **observable living habits**, not personality.

| Topic | Examples of what it covers |
|---|---|
| Cleanliness | Tidiness standards, chores |
| Noise | Tolerance for noise at home |
| Sleep and routines | Early, flexible, or late schedule |
| Guests | How often, overnight or not |
| Privacy | Need for personal space |
| Social level | Quiet home vs. social home |
| Shared expenses | Split equally, itemized, or mixed |
| Disagreements | Talk now, cool off first, or write it down |
| Smoking and pets | No, outside only, yes; none, ok, have |
| Working from home | How often (0–5 days) |
| Employment | What best describes your work right now |

That's 12 dimensions. Six of them (cleanliness, noise, guests, privacy, social level, disagreements) get AI-drafted scenario prompts that vary each visit — the stored answers and scoring never change meaning. Factual axes (sleep, expenses, smoking, pets, working from home, employment) keep direct wording.

Users can mark an answer as a **hard requirement** (a deal-breaker) or a flexible preference, and say how much it matters to them (weight 1–3).

## 7. How Matching Works

Matching is **transparent and rule-based**. It is not a black box.

**Step 1: Eligibility.** Remove anyone who breaks a hard requirement: a deal-breaker answer, a blocked pair, a mismatched gender preference, or a hidden/deactivated account. Arrangement mode, budget, dates, and area affect the score but never disqualify anyone on their own.

**Step 2: Scoring.** Score each remaining pair from 0 to 100 using three groups: practical fit (budget, location, timing), lifestyle fit, and interpersonal fit. Weights are configurable and live outside the code.

**Step 3: Explanation.** For every match, show what lines up, what differs, and what's worth discussing, for example:
- Strong alignment on budget and daily routines.
- Similar expectations on cleanliness.
- Different views on overnight guests, so talk about it first.

## 8. Where AI Helps

AI assists the experience. It does not make decisions.

**AI is used to:**
- Run a short capped conversation that drafts questionnaire answers, with the user reviewing and confirming everything before it counts.
- Draft fresh scenario prompts for six questionnaire axes on each visit.
- Clean up messy area text on profile save (the user's wording is kept on any failure).
- Turn the computed score breakdown into a readable explanation.
- Suggest neutral questions to discuss for flagged differences, plus one practical compromise per difference.
- Draft a plain-language flatmate agreement from both sides' profiles and answers (template fallback; never legal advice).
- Parse a natural-language line ("quiet cleaner under 300k") into match filters with a shown interpretation.

**AI is not used to:**
- Rank, filter, or score people.
- Infer personality or sensitive traits from what someone writes.

If the AI is unavailable, the product still works with a plain form and template explanations. Every model call is schema-validated with a deterministic fallback, so the demo never dead-ends on a model error.

## 9. Trust and Safety (Minimum)

- Secure login: salted password hashes, signed session cookies, CSRF tokens, login rate limits.
- Searching visibility toggle: hide yourself from all matching at any time.
- Block and report on every profile.
- Chat is limited to mutual-interest pairs; hidden, blocked, and deactivated users can't be messaged, and listing owners are reportable. Opening a thread marks it read; unread counts show as badges on the inbox, the SPA nav, and the legacy inbox.
- Blocked users never see each other.
- Only first name and general area are shown. No exact addresses; the model never receives contact details or precise coordinates.
- Contact details are shared only after mutual interest.
- Users can deactivate their account.
- The UI states plainly what has been verified.
- Contacts carry honest labels: self-confirmed or unverified. There is still no identity verification.
- Admins triage reports from a queue (dismiss, or deactivate the reported user); admin rights are granted from the command line, never self-serve.
- Flatmate reviews require a verified stay and carry a verified tag. Listing reviews split into tenant (verified stay, 3x weight) and visitor views with a trust score.

## 10. Measuring Success

Measure whether people form useful connections, not time spent in the app.

- Profile and questionnaire completion.
- Users who receive at least one eligible match.
- Mutual interest and conversations started.
- Users who report finding a suitable flatmate.
- Reports and blocks per active user.

## 11. Key Risks

| Risk | Response |
|---|---|
| Too few people in an area | Global scope with city-agnostic matching; area scores fit but never gates, and a city filter can follow if cross-city pairs confuse |
| Scores create false confidence | Always show reasons and uncertainty; keep scores advisory |
| Fraud and safety issues | Reporting, blocking, privacy controls, honest verification labels |
| People won't pay | Start free; test paid options only after the product proves useful |

## 12. Later (Not Now)

- **Monetization:** Test a time-limited access pass or boosts after value is proven. Rent norms vary by market (annual lump sums vs. monthly), so monthly subscriptions may not fit everywhere; research this before deciding.
- **Verification:** Self-confirmed contact labels ship now; add identity checks when cost and operations allow.
- **Growth:** Add partners or new cities only after the first markets work.
- **Personality research:** Consider only if validated against real outcomes.

## 13. Decision Test

Before adding anything, ask:

1. Does it help people find a more compatible flatmate?
2. Is it needed to test the MVP idea?
3. Is there a simpler way to get the same result?

If the answer to the first two is no, defer it.

## 14. Product Definition

The Flatmate Matching Platform helps people make better-informed decisions about who they live with, by combining practical housing requirements with lifestyle compatibility and clear explanations.

**It wins by improving the quality of flatmate decisions, not by having the most listings or the most features.**
