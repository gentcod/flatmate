You are a senior Go engineer and technical writer. Your job is to author a
CONTEXT.md file: the working brief that a coding agent (Muse Code) will read
at the start of every session while building a project.

## The project
A Flatmate Matching Platform: a compatibility-focused app that helps people
find compatible flatmates. It treats shared living as a compatibility problem,
not a property search. The full product spec is below.

<product_spec>
# Flatmate Matching Platform
## Simplified Product Reference

**Status:** Concept, in implementation
**First market:** Lagos, Nigeria (US is a possible later market)
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

- **Practical first, lifestyle second.** Budget, location, and dates decide eligibility. Lifestyle decides who ranks higher.
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
- Sign up and log in.
- Profile: what you're looking for, preferred area, budget, move-in window, short bio.
- Lifestyle questionnaire (see section 6).
- Eligibility filtering on hard requirements.
- Ranked matches with explanations and areas to discuss.
- Save, dismiss, and express interest. Mutual interest shares a contact request.
- Block and report.

**Not included yet**
Payments, identity verification, in-app messaging, landlord or agent tools, property listings, rent or lease tools, machine-learning ranking, and personality analysis.

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
| Working from home | How often |

Users can mark an answer as a **hard requirement** (a deal-breaker) or a flexible preference, and say how much it matters to them.

## 7. How Matching Works

Matching is **transparent and rule-based**. It is not a black box.

**Step 1: Eligibility.** Remove anyone who breaks a hard requirement: wrong arrangement, budget doesn't fit, move-in dates don't overlap, a deal-breaker like smoking or pets, or a blocked user.

**Step 2: Scoring.** Score each remaining pair from 0 to 100 using three groups: practical fit (budget, location, timing), lifestyle fit, and interpersonal fit. Weights are configurable and live outside the code.

**Step 3: Explanation.** For every match, show what lines up, what differs, and what's worth discussing, for example:
- Strong alignment on budget and daily routines.
- Similar expectations on cleanliness.
- Different views on overnight guests, so talk about it first.

## 8. Where AI Helps

AI assists the experience. It does not make decisions.

**AI is used to:**
- Run a short conversation that fills in the questionnaire, with the user confirming the answers.
- Turn the computed score breakdown into a readable explanation.
- Suggest neutral questions to discuss for flagged differences.

**AI is not used to:**
- Rank, filter, or score people.
- Infer personality or sensitive traits from what someone writes.

If the AI is unavailable, the product still works with a plain form and template explanations.

## 9. Trust and Safety (Minimum)

- Secure login.
- Block and report on every profile.
- Blocked users never see each other.
- Only first name and general area are shown. No exact addresses.
- Contact details are shared only after mutual interest.
- Users can deactivate their account.
- The UI states plainly what has been verified.

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
| Too few people in an area | Launch in a narrow area and measure match availability |
| Scores create false confidence | Always show reasons and uncertainty; keep scores advisory |
| Fraud and safety issues | Reporting, blocking, privacy controls, honest verification labels |
| People won't pay | Start free; test paid options only after the product proves useful |

## 12. Later (Not Now)

- **Monetization:** Test a time-limited access pass or boosts after value is proven. Lagos renters often commit annually, so monthly subscriptions may not fit; research this before deciding.
- **Verification:** Add identity checks when cost and operations allow.
- **Growth:** Add listings, partners, or a second city only after the first market works.
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
</product_spec>

## The situation
- This is for the Meta Global AI Developer Hackathon. I'm an individual
  participant, judged on creativity, technical ambition, and whether it works.
- Submission is due Nov 4, 9:00 PM ET. I have about 7 days. Target: submit by
  midday Nov 4.
- The build must work end to end in a ~3 minute demo.
- The CONTEXT.md you write will guide an AI coding agent, so it must be
  unambiguous and actionable, not descriptive prose.

## Decisions already made (treat as fixed, do not re-litigate)
Stack:
- Go 1.23+, standard library net/http with method-aware ServeMux
- SQLite via modernc.org/sqlite (pure Go, no cgo), plain SQL migrations
- Server-rendered html/template + htmx, single binary
- Auth: paseto + signed session cookie
- Scoring weights in a weights.yaml config, never hard-coded

Architecture rule (most important):
- The matching engine is pure Go: no network calls, no LLM, fully unit-tested.
- Muse never ranks, filters, or scores. It only (a) extracts structured
  questionnaire data from a onboarding conversation, (b) writes plain-language
  explanations of an already-computed score breakdown, (c) drafts neutral
  discussion questions for already-flagged differences. A fourth, optional
  stretch use is normalizing messy listing text (e.g. "self-con, BQ available").
- All model output is untrusted: validate against a schema, retry once, then
  fall back to a deterministic template or plain form. The demo must never
  dead-end on a model error.
- Do not infer personality or sensitive traits from free text. Send the model
  only what it needs (first name, neighborhood, questionnaire answers), never
  email, phone, or exact address.

Scope (MVP for the hackathon):
- In: signup/login, profile, conversational onboarding, eligibility filter,
  weighted rule-based scoring with per-dimension breakdown, match list/detail
  with explanation and discussion prompts, save/dismiss/express interest,
  mutual interest reveals a contact request (no chat), block and report,
  25-30 synthetic Lagos seed profiles.
- Out: payments, identity verification, real-time messaging, landlord/agent
  features, ML ranking, analytics dashboards, multi-country support.

Matching model:
- Stage 1 eligibility (mode pairing, budget fit, move-in overlap, hard
  constraints like smoking/pets, blocks), Stage 2 weighted scoring 0-100
  (practical / lifestyle / interpersonal), Stage 3 a Breakdown struct that is
  the contract between the engine, the UI, and Muse.
- Compatibility is not similarity: some dimensions are flagged for discussion
  rather than penalized.

## Required sections in CONTEXT.md
1. How the agent should work in this repo (rules of engagement)
2. Hackathon constraints
3. Product in brief + principles that affect code
4. Scope (in / out)
5. Architecture and the Muse boundary (with the hard rules)
6. Muse touchpoints, each with input, output JSON shape, and rules
7. Resilience and fallbacks (include a Go LLM interface and a generic
   CompleteJSON[T] signature)
8. Stack and repo layout (directory tree)
9. Data model (tables and key columns)
10. Matching spec: questionnaire dimensions as a table (name, type, scale,
    how compared), eligibility rules, scoring approach, and the Breakdown and
    DimensionResult Go structs
11. Safety and privacy minimums
12. Day-by-day build order through Nov 4, plus what to cut first if behind
13. Definition of done and a 3-minute demo script
14. Open items for the developer

## Quality bar
- Written as instructions to an agent: imperative, specific, testable.
- Concrete over general: use tables, schemas, and structs where they remove
  ambiguity.
- Maximum ~250 lines. No marketing language. No repeating the spec; link to
  docs/flatmate.md for background.
- Do NOT invent details about the Meta Model API (endpoints, parameters,
  model names, structured-output support). Put each as TODO(fill) in section
  14 for me to complete.
- If the spec conflicts with the decisions above, the decisions win for the
  hackathon; note the conflict briefly.

## Process (follow in order)
Step 1: Do NOT write the file yet. Reply with (a) your understanding of the
project in at most 10 bullets, and (b) at most 5 open questions that would
materially change the document. Then stop and wait for my answers.
Step 2: After I answer, write the complete CONTEXT.md in a single markdown
code block.
Step 3: Review your own draft against the quality bar. List any ambiguities
an agent could misread, any section that is thin, and any assumption you made
that I should confirm. Then output a corrected final version.