import { useState } from 'react'
import type { FormEvent } from 'react'
import { Link, Navigate, useNavigate } from 'react-router-dom'
import {
  Card,
  Chip,
  Icon,
  ScoreBar,
  TrustBadge,
  ghostBtnCls,
  inputCls,
  primaryBtnCls,
} from '../components'
import { useSession } from '../query'
import { OfflineBanner } from '../pwa'

const PRESETS = ['Quiet home', 'Early riser', 'Non-smoker', 'Private room', 'Team-up']

const KITCHEN_LABELS = [
  'Immediate wipe-down after every meal',
  'Dishes cleared within 2 hours',
  'Dishes soaked & cleared by 10am',
  'Casual shared sink, weekly team sweep',
]

const NOISE_LABELS = [
  'Strict silence from 21:00',
  'Soft quiet after 22:30',
  'Casual quiet after midnight',
  'Social flat, no fixed curfew',
]

const INSIGHTS = [
  'High-structure sanctuary: deep restorative silence, every night predictable.',
  'Gentle domestic harmony: predictable weeknights with room for shared dinners.',
  'Creative flow mode: night-owl friendly, best matched with fellow studio minds.',
  'Social collective: high energy, open living room, spontaneous hosting welcome.',
]

function SiteHeader() {
  return (
    <header className="sticky top-0 z-40 border-b border-sand bg-canvas/90 backdrop-blur-xl">
      <div className="mx-auto flex min-h-20 max-w-7xl flex-wrap items-center justify-between gap-3 px-5 py-2 lg:px-12">
        <span className="flex flex-col leading-none">
          <span className="font-display text-2xl tracking-tight text-ink">Flatmate</span>
          <span className="mt-0.5 text-[11px] font-semibold uppercase tracking-widest text-fog">
            Shared Living
          </span>
        </span>
        <nav className="flex items-center gap-2">
          <Link to="/login" className={ghostBtnCls}>
            Log in
          </Link>
          <Link to="/signup" className={primaryBtnCls}>
            Sign up
          </Link>
        </nav>
      </div>
    </header>
  )
}

function Hero() {
  const navigate = useNavigate()
  const [nl, setNl] = useState('quiet graphic designer, early riser, plant lover')
  const go = (e: FormEvent) => {
    e.preventDefault()
    navigate('/signup')
  }
  return (
    <section className="mx-auto max-w-7xl px-5 pt-10 lg:px-12">
      <div className="mb-4 flex items-center gap-3">
        <span className="inline-flex items-center gap-1.5 rounded-full bg-parchment px-3 py-1 text-[13px] font-semibold text-eucalyptus">
          <span aria-hidden="true" className="h-2 w-2 rounded-full bg-eucalyptus" />
          Intentionally aligned co-living
        </span>
      </div>
      <div className="grid grid-cols-1 items-start gap-8 lg:grid-cols-12">
        <div className="flex flex-col justify-center lg:col-span-7">
          <h1 className="font-display text-4xl tracking-tight text-ink lg:text-[52px] lg:leading-[1.08]">
            Find a home where your rhythms <em className="text-terracotta">actually match</em>.
          </h1>
          <p className="mb-6 mt-4 max-w-xl text-[17px] leading-relaxed text-sage">
            Beyond generic classifieds. Discover flatmates and rooms matched by sleep
            schedules, cleanliness standards, guest boundaries, and affordability.
          </p>
          <Card>
            <form onSubmit={go} className="flex flex-col gap-3">
              <div className="flex flex-col gap-2">
                <label
                  htmlFor="landing-nl"
                  className="flex items-center justify-between text-[13px] font-semibold text-sage"
                >
                  <span>Express what you seek in human terms</span>
                  <span className="text-[11px] font-normal text-fog">AI habit filter</span>
                </label>
                <div className="relative flex items-center">
                  <span className="pointer-events-none absolute left-3.5 text-sage">
                    <Icon name="psychology_alt" />
                  </span>
                  <input
                    id="landing-nl"
                    value={nl}
                    onChange={(e) => setNl(e.target.value)}
                    maxLength={200}
                    className={`${inputCls} pl-11`}
                  />
                </div>
              </div>
              <div className="flex flex-wrap items-center gap-2">
                <span className="mr-1 text-[13px] text-fog">Presets:</span>
                {PRESETS.map((p) => (
                  <button
                    key={p}
                    type="button"
                    onClick={() => setNl((v) => (v ? `${v}, ${p.toLowerCase()}` : p))}
                    className="cursor-pointer rounded-full bg-parchment px-2.5 py-1 text-[13px] font-medium text-ink hover:bg-sand"
                  >
                    {p}
                  </button>
                ))}
              </div>
              <div className="flex flex-col gap-3 pt-1 sm:flex-row">
                <button type="submit" className={`${primaryBtnCls} flex items-center justify-center gap-2 px-6 py-3 text-[15px]`}>
                  Explore matches
                  <Icon name="arrow_forward" cls="text-[18px]" />
                </button>
                <a
                  href="#how"
                  className={`${ghostBtnCls} flex items-center justify-center gap-2 px-5 py-3 text-[15px]`}
                >
                  <Icon name="tune" cls="text-[18px]" />
                  See how it works
                </a>
              </div>
            </form>
          </Card>
          <div className="mt-6 flex flex-wrap items-center gap-x-6 gap-y-2 text-[13px] text-sage">
            <span className="flex items-center gap-2">
              <Icon name="functions" cls="text-[18px] text-eucalyptus" />
              Fixed compatibility formula
            </span>
            <span className="flex items-center gap-2">
              <Icon name="shield_person" cls="text-[18px] text-eucalyptus" />
              Scores advise; you decide
            </span>
            <span className="flex items-center gap-2">
              <Icon name="lock" cls="text-[18px] text-terracotta" />
              No exact addresses shared
            </span>
          </div>
        </div>
        <div className="lg:col-span-5">
          <Card>
            <p className="mb-3 text-[11px] font-semibold uppercase tracking-wider text-fog">
              Example match preview
            </p>
            <div className="flex items-start gap-4">
              <span
                aria-hidden="true"
                className="flex h-16 w-16 shrink-0 items-center justify-center rounded-full bg-canopy font-display text-2xl text-canvas"
              >
                J
              </span>
              <div className="min-w-0 flex-1">
                <div className="flex items-center justify-between gap-2">
                  <h3 className="truncate text-[20px] font-semibold text-ink">Jordan, 26</h3>
                  <span className="shrink-0 rounded-full bg-mist px-2 py-0.5 text-[11px] font-semibold text-eucalyptus">
                    94% match
                  </span>
                </div>
                <p className="mt-0.5 truncate text-[13px] text-fog">
                  Industrial designer · Example area
                </p>
                <div className="mt-1.5">
                  <TrustBadge selfConfirmed={false} />
                </div>
              </div>
            </div>
            <div className="mt-4 flex flex-col gap-3 rounded-xl bg-parchment/70 p-3">
              <div>
                <div className="mb-1 flex justify-between text-[11px]">
                  <span className="text-sage">Sleep schedule</span>
                  <span className="font-semibold text-eucalyptus">Synchronized</span>
                </div>
                <ScoreBar value={95} />
              </div>
              <div>
                <div className="mb-1 flex justify-between text-[11px]">
                  <span className="text-sage">Kitchen standard</span>
                  <span className="font-semibold text-terracotta">Within 2 hours</span>
                </div>
                <ScoreBar value={88} tone="pivotal" />
              </div>
            </div>
            <p className="mt-3 font-display text-[15px] italic leading-snug text-sage">
              “Quiet weekday evenings, shared weekend market strolls, occasional community
              cooking.”
            </p>
          </Card>
          <div className="-mt-2 ml-6">
            <Card>
              <div className="flex gap-4">
                <img
                  src="/placeholders/room-1.svg"
                  alt=""
                  loading="lazy"
                  className="h-24 w-24 shrink-0 rounded-xl object-cover"
                />
                <div className="min-w-0 flex-1">
                  <div className="flex items-center justify-between gap-2">
                    <span className="text-[13px] font-semibold text-terracotta">
                      Example room
                    </span>
                    <span className="text-[20px] font-semibold text-ink">
                      $720
                      <span className="text-[13px] font-normal text-fog">/mo</span>
                    </span>
                  </div>
                  <h4 className="mt-0.5 truncate font-semibold text-ink">
                    High-ceiling bedroom
                  </h4>
                  <p className="truncate text-[13px] text-sage">Example area · 22 m²</p>
                  <p className="mt-2 inline-flex items-center gap-1.5 self-start rounded bg-mist px-2 py-1 text-[11px] font-semibold text-eucalyptus">
                    <Icon name="account_balance_wallet" cls="text-[14px]" />
                    Meets 30% rent rule
                  </p>
                </div>
              </div>
            </Card>
          </div>
        </div>
      </div>
    </section>
  )
}

function Contrast() {
  return (
    <section className="mt-12 w-full bg-parchment py-10">
      <div className="mx-auto max-w-7xl px-5 lg:px-12">
        <div className="mb-6 max-w-2xl">
          <p className="text-[13px] font-semibold uppercase tracking-wider text-terracotta">
            The roommate dilemma
          </p>
          <h2 className="mt-1 font-display text-3xl tracking-tight text-ink">
            Shared living shouldn&apos;t feel like a lottery.
          </h2>
          <p className="mt-2 text-[15px] text-sage">
            Traditional classifieds focus on square meters and rent. They ignore how people
            actually live together — until the friction starts.
          </p>
        </div>
        <div className="grid grid-cols-1 gap-6 md:grid-cols-2">
          <Card>
            <p className="mb-4 flex items-center gap-2 text-[13px] font-semibold uppercase tracking-wider text-fog">
              <Icon name="cancel" cls="text-[20px]" />
              The standard marketplace
            </p>
            <ul className="flex flex-col gap-3.5 text-[15px] text-sage">
              <li className="flex items-start gap-2.5">
                <span aria-hidden="true" className="mt-0.5 text-err">•</span>
                Hundreds of copy-pasted “is this still free?” messages.
              </li>
              <li className="flex items-start gap-2.5">
                <span aria-hidden="true" className="mt-0.5 text-err">•</span>
                Casting-style viewings that feel like hostile job interviews.
              </li>
              <li className="flex items-start gap-2.5">
                <span aria-hidden="true" className="mt-0.5 text-err">•</span>
                Unspoken expectations around guests, cleaning, and quiet hours.
              </li>
              <li className="flex items-start gap-2.5">
                <span aria-hidden="true" className="mt-0.5 text-err">•</span>
                Hidden bills, shady sublets, zero clarity before signing.
              </li>
            </ul>
            <p className="mt-6 rounded-xl bg-canvas p-3 text-[13px] text-fog">
              Result: friction discovered after move-in.
            </p>
          </Card>
          <Card>
            <p className="mb-4 flex items-center gap-2 text-[13px] font-semibold uppercase tracking-wider text-eucalyptus">
              <Icon name="check_circle" cls="text-[20px]" />
              The Flatmate way
            </p>
            <ul className="flex flex-col gap-3.5 text-[15px] text-ink">
              <li className="flex items-start gap-2.5">
                <span aria-hidden="true" className="mt-0.5 font-bold text-eucalyptus">✓</span>
                Lifestyle indicators matched on 12 explicit dimensions before chatting.
              </li>
              <li className="flex items-start gap-2.5">
                <span aria-hidden="true" className="mt-0.5 font-bold text-eucalyptus">✓</span>
                Affordability checks against your income before you fall for a room.
              </li>
              <li className="flex items-start gap-2.5">
                <span aria-hidden="true" className="mt-0.5 font-bold text-eucalyptus">✓</span>
                Honest contact labels — self-confirmed or unverified, nothing faked.
              </li>
              <li className="flex items-start gap-2.5">
                <span aria-hidden="true" className="mt-0.5 font-bold text-eucalyptus">✓</span>
                House agreements drafted together before move-in.
              </li>
            </ul>
            <p className="mt-6 rounded-xl bg-mist p-3 text-[13px] font-semibold text-eucalyptus">
              Result: alignment checked before you chat.
            </p>
          </Card>
        </div>
      </div>
    </section>
  )
}

const PILLARS = [
  {
    icon: 'bedtime',
    title: '12-dimension rhythm calibration',
    body: 'Compare sleep schedules, dishwashing timing, remote-work hours, guest norms, and noise tolerance — without awkward small talk.',
    chips: ['Sleep sync', 'Kitchen rules', 'Guest cadence'],
  },
  {
    icon: 'account_balance',
    title: 'Affordability & shared expenses',
    body: 'Every room carries an honest rent figure plus an affordability check. Split bills with a running balance once you live together.',
    chips: ['Rent checks', 'Expense ledger'],
  },
  {
    icon: 'verified_user',
    title: 'Honest trust-lite verification',
    body: 'No fake badges. Contacts are self-confirmed or unverified, reviews come one-per-pair, and reports reach a human queue.',
    chips: ['Self-confirmed labels', 'One-per-pair reviews'],
  },
]

function Pillars() {
  return (
    <section className="mx-auto max-w-7xl px-5 py-12 lg:px-12">
      <div className="mx-auto mb-8 max-w-2xl text-center">
        <p className="text-[13px] font-semibold uppercase tracking-wider text-sage">
          Built for sanctuary
        </p>
        <h2 className="mt-1 font-display text-3xl tracking-tight text-ink">
          Three pillars of thoughtful co-living
        </h2>
      </div>
      <div className="grid grid-cols-1 gap-6 md:grid-cols-3">
        {PILLARS.map((p, i) => (
          <Card key={p.title}>
            <div className="flex h-full flex-col justify-between gap-6">
              <div>
                <span className="mb-4 flex h-12 w-12 items-center justify-center rounded-xl bg-parchment text-eucalyptus">
                  <Icon name={p.icon} cls="text-[26px]" />
                </span>
                <p className="text-[13px] font-semibold text-terracotta">
                  Pillar 0{i + 1}
                </p>
                <h3 className="mb-2 mt-1 text-[20px] font-semibold text-ink">{p.title}</h3>
                <p className="text-[15px] leading-relaxed text-sage">{p.body}</p>
              </div>
              <div className="flex flex-wrap gap-1.5">
                {p.chips.map((c) => (
                  <Chip key={c}>{c}</Chip>
                ))}
              </div>
            </div>
          </Card>
        ))}
      </div>
    </section>
  )
}

function Sandbox() {
  const [kitchen, setKitchen] = useState(3)
  const [noise, setNoise] = useState(2)
  const k = kitchen - 1
  const n = noise - 1
  const score = Math.max(72, 96 - Math.abs(k - n) * 7)
  const insight = INSIGHTS[Math.min(Math.round((k + n) / 2), INSIGHTS.length - 1)]
  return (
    <section id="how" className="w-full bg-eucalyptus py-12">
      <div className="mx-auto max-w-7xl px-5 lg:px-12">
        <div className="mb-8 flex flex-col justify-between gap-4 md:flex-row md:items-end">
          <div>
            <p className="text-[13px] font-semibold uppercase tracking-wider text-[#aacec0]">
              How Flatmate works
            </p>
            <h2 className="mt-1 font-display text-3xl tracking-tight text-white">
              From everyday scenarios to effortless harmony
            </h2>
          </div>
          <p className="max-w-md text-[15px] text-[#aacec0]">
            Instead of clinical personality tests, micro-scenarios reflect real domestic
            life.
          </p>
        </div>
        <div className="mb-10 grid grid-cols-1 gap-6 md:grid-cols-3">
          {[
            ['Set your living rhythm', 'A short conversational setup or the plain 12-dimension form — your choice.'],
            ['Discover mutual alignment', 'Browse transparent profiles. See where you sync and where you differ.'],
            ['Co-create house agreements', 'Connect on mutual interest and agree on quiet hours, chores, and bills.'],
          ].map(([t, b], i) => (
            <div key={t} className="flex items-start gap-4">
              <span className="flex h-8 w-8 shrink-0 items-center justify-center rounded-full bg-canopy text-[16px] font-semibold text-white">
                {i + 1}
              </span>
              <div>
                <h4 className="font-semibold text-white">{t}</h4>
                <p className="mt-1 text-sm text-[#aacec0]">{b}</p>
              </div>
            </div>
          ))}
        </div>
        <Card>
          <div className="flex flex-col justify-between gap-4 border-b border-sand pb-4 md:flex-row md:items-center">
            <div>
              <p className="text-[13px] font-semibold uppercase tracking-wide text-eucalyptus">
                Interactive sandbox · demo only, illustrative score
              </p>
              <h3 className="mt-0.5 font-display text-2xl tracking-tight text-ink">
                Scenario: “The sink at midnight”
              </h3>
            </div>
            <p className="flex items-center gap-2 text-[13px] text-sage">
              Live alignment score:
              <span className="text-[20px] font-semibold tabular-nums text-eucalyptus">
                {score}%
              </span>
            </p>
          </div>
          <div className="grid grid-cols-1 items-center gap-8 pt-4 md:grid-cols-2">
            <div className="flex flex-col gap-5">
              <div>
                <div className="mb-2 flex justify-between text-sm">
                  <span className="font-medium text-ink">Your kitchen standard</span>
                  <span className="font-semibold text-terracotta">{KITCHEN_LABELS[k]}</span>
                </div>
                <input
                  type="range"
                  min={1}
                  max={4}
                  value={kitchen}
                  onChange={(e) => setKitchen(Number(e.target.value))}
                  aria-label="Your kitchen standard"
                  className="w-full cursor-pointer accent-eucalyptus"
                />
                <div className="mt-1 flex justify-between text-[11px] text-fog">
                  <span>Immediate</span>
                  <span>2 hours</span>
                  <span>Morning</span>
                  <span>Weekly</span>
                </div>
              </div>
              <div>
                <div className="mb-2 flex justify-between text-sm">
                  <span className="font-medium text-ink">Weekday quiet hours</span>
                  <span className="font-semibold text-terracotta">{NOISE_LABELS[n]}</span>
                </div>
                <input
                  type="range"
                  min={1}
                  max={4}
                  value={noise}
                  onChange={(e) => setNoise(Number(e.target.value))}
                  aria-label="Weekday quiet hours"
                  className="w-full cursor-pointer accent-eucalyptus"
                />
                <div className="mt-1 flex justify-between text-[11px] text-fog">
                  <span>21:00</span>
                  <span>22:30</span>
                  <span>00:00</span>
                  <span>None</span>
                </div>
              </div>
            </div>
            <div className="flex h-full flex-col justify-between rounded-xl bg-parchment p-4">
              <div className="flex flex-col gap-3">
                <p className="flex items-center gap-2 font-semibold text-eucalyptus">
                  <Icon name="psychology" cls="text-[20px]" />
                  What this means
                </p>
                <p className="text-[15px] leading-relaxed text-sage">{insight}</p>
              </div>
              <div className="mt-4 flex items-center justify-between border-t border-sand pt-3">
                <span className="text-[13px] text-fog">Illustrative — real scores use 12 dimensions</span>
                <Link
                  to="/signup"
                  className="flex items-center gap-1 text-sm font-semibold text-eucalyptus hover:underline"
                >
                  Get your score
                  <Icon name="chevron_right" cls="text-[16px]" />
                </Link>
              </div>
            </div>
          </div>
        </Card>
      </div>
    </section>
  )
}

const EXAMPLE_ROOMS = [
  { img: '/placeholders/room-1.svg', area: 'Example area', rent: '$720', title: 'Quiet room facing the courtyard', meta: '24 m² · 3-person flat', owner: 'L', ownerName: 'Lee', habits: 'Night owl · Plant lover · Hybrid work' },
  { img: '/placeholders/room-2.svg', area: 'Example area', rent: '$690', title: 'Bright creative haven', meta: '19 m² · 2-person flat', owner: 'S', ownerName: 'Sam', habits: 'Early riser · Quiet nights' },
  { img: '/placeholders/room-3.svg', area: 'Example area', rent: '$640', title: 'Attic studio with balcony', meta: '18 m² · 2-person flat', owner: 'F', ownerName: 'Frank', habits: 'Film enthusiast · Respectful' },
]

function ExampleRooms() {
  return (
    <section className="mx-auto max-w-7xl px-5 py-12 lg:px-12">
      <div className="mb-8 max-w-2xl">
        <p className="text-[13px] font-semibold uppercase tracking-wider text-terracotta">
          Example rooms
        </p>
        <h2 className="mt-1 font-display text-3xl tracking-tight text-ink">
          Room openings with transparent flatmate profiles
        </h2>
        <p className="mt-2 text-[15px] text-sage">
          Illustrative examples — sign up to browse real rooms near you.
        </p>
      </div>
      <div className="grid grid-cols-1 gap-6 md:grid-cols-2 lg:grid-cols-3">
        {EXAMPLE_ROOMS.map((r) => (
          <article
            key={r.title}
            className="flex flex-col overflow-hidden rounded-2xl border border-sand bg-white shadow-card transition-shadow hover:shadow-lift"
          >
            <div className="relative">
              <img src={r.img} alt="" loading="lazy" className="aspect-[16/9] w-full object-cover" />
              <span className="absolute left-3 top-3 rounded bg-white/90 px-2.5 py-1 text-[13px] font-semibold text-eucalyptus backdrop-blur">
                {r.area}
              </span>
              <span className="absolute bottom-3 right-3 rounded-lg bg-eucalyptus px-3 py-1 text-[20px] font-semibold text-white">
                {r.rent}
                <span className="text-sm font-normal text-[#aacec0]">/mo</span>
              </span>
            </div>
            <div className="flex flex-1 flex-col justify-between gap-3 p-5">
              <div>
                <p className="text-[13px] text-fog">{r.meta}</p>
                <h3 className="mt-0.5 line-clamp-1 font-semibold text-ink">{r.title}</h3>
                <div className="mt-3 flex items-center gap-3 rounded-xl bg-parchment p-2.5">
                  <span
                    aria-hidden="true"
                    className="flex h-10 w-10 shrink-0 items-center justify-center rounded-full bg-canopy text-[15px] font-semibold text-white"
                  >
                    {r.owner}
                  </span>
                  <div className="min-w-0">
                    <p className="truncate text-sm font-semibold text-ink">{r.ownerName}</p>
                    <p className="truncate text-[11px] text-sage">{r.habits}</p>
                  </div>
                </div>
              </div>
              <div className="flex items-center justify-between border-t border-sand pt-3">
                <span className="flex items-center gap-1.5 text-[13px] font-semibold text-eucalyptus">
                  <Icon name="tune" cls="text-[16px]" />
                  Compatibility explained
                </span>
                <Link
                  to="/signup"
                  className="text-sm font-semibold text-terracotta hover:underline"
                >
                  View rooms
                </Link>
              </div>
            </div>
          </article>
        ))}
      </div>
    </section>
  )
}

function TrustBanner() {
  return (
    <section className="mx-auto max-w-7xl px-5 pb-12 lg:px-12">
      <div className="flex flex-col items-center justify-between gap-6 rounded-2xl bg-parchment p-6 lg:flex-row lg:p-10">
        <div className="flex max-w-2xl items-start gap-4">
          <span className="flex h-12 w-12 shrink-0 items-center justify-center rounded-full bg-mist text-eucalyptus">
            <Icon name="verified" cls="text-[24px]" />
          </span>
          <div>
            <h3 className="font-display text-2xl tracking-tight text-ink">
              “Compatibility scores are purely advisory insights. You decide who you share
              home with.”
            </h3>
            <p className="mt-1 text-[15px] leading-relaxed text-sage">
              Hide your profile instantly whenever you pause your search. Verification is
              voluntary and transparent: self-confirmed or unverified, always labeled.
            </p>
          </div>
        </div>
        <Link to="/signup" className={`${primaryBtnCls} shrink-0 px-6 py-3 text-[15px]`}>
          Create your profile
        </Link>
      </div>
    </section>
  )
}

function FinalCta() {
  return (
    <section className="mx-auto max-w-7xl px-5 pb-12 lg:px-12">
      <div className="relative flex flex-col items-center justify-between gap-6 overflow-hidden rounded-2xl bg-eucalyptus p-6 text-center shadow-lift lg:flex-row lg:p-10 lg:text-left">
        <div className="relative z-10 max-w-xl">
          <p className="text-[13px] font-semibold uppercase tracking-wider text-[#aacec0]">
            Stop gambling on strangers
          </p>
          <h2 className="mb-2 mt-2 font-display text-3xl tracking-tight text-white lg:text-4xl">
            Ready to find flatmates who match your domestic pace?
          </h2>
          <p className="text-[17px] leading-relaxed text-[#aacec0]">
            Create your living-habit profile in minutes and browse rooms with honest
            flatmate profiles.
          </p>
        </div>
        <div className="relative z-10 flex w-full flex-col items-center gap-3 sm:flex-row lg:w-auto">
          <Link
            to="/signup"
            className="w-full rounded-xl bg-terracotta px-7 py-3.5 text-center font-semibold text-white shadow-sm hover:bg-clay sm:w-auto"
          >
            Get started free
          </Link>
          <Link
            to="/signup"
            className="w-full rounded-xl bg-canopy px-6 py-3.5 text-center font-semibold text-white hover:bg-eucalyptus sm:w-auto"
          >
            Browse rooms
          </Link>
        </div>
      </div>
    </section>
  )
}

export function LandingPage() {
  const session = useSession()
  if (session.data?.authed) return <Navigate to="/matches" replace />
  return (
    <div className="min-h-screen">
      <OfflineBanner />
      <SiteHeader />
      <main>
        <Hero />
        <Contrast />
        <Pillars />
        <Sandbox />
        <ExampleRooms />
        <TrustBanner />
        <FinalCta />
      </main>
      <footer className="border-t border-sand py-8 text-center">
        <p className="mx-auto max-w-xl px-5 font-display text-[19px] italic text-sage">
          Flatmate compatibility scores are purely advisory insights. You decide who you
          share home with.
        </p>
      </footer>
    </div>
  )
}
