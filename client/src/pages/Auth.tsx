import { useState } from 'react'
import type { FormEvent } from 'react'
import { Link, useNavigate } from 'react-router-dom'
import {
  Card,
  ErrorNote,
  Field,
  LoadingModal,
  PageTitle,
  errorMessage,
  inputCls,
  primaryBtnCls,
} from '../components'
import { useLogin, useSignup } from '../query'
import { OfflineBanner } from '../pwa'

function AuthForm({ mode }: { mode: 'login' | 'signup' }) {
  const [email, setEmail] = useState('')
  const [password, setPassword] = useState('')
  const [showPw, setShowPw] = useState(false)
  const navigate = useNavigate()
  const login = useLogin()
  const signup = useSignup()
  const active = mode === 'login' ? login : signup

  const onSubmit = (e: FormEvent) => {
    e.preventDefault()
    active.mutate({ email, password }, { onSuccess: () => navigate('/matches') })
  }

  return (
    <div className="mx-auto max-w-md">
      <OfflineBanner />
      <div className="mb-6 flex flex-col leading-none">
        <span className="font-display text-2xl tracking-tight text-ink">Flatmate</span>
        <span className="mt-0.5 text-[11px] font-semibold uppercase tracking-widest text-fog">
          Shared Living
        </span>
      </div>
      <PageTitle
        title={mode === 'login' ? 'Log in' : 'Create your account'}
        sub={
          mode === 'login'
            ? 'Welcome back. Your matches are waiting.'
            : 'One account, honest answers, better flatmates.'
        }
      />
      <Card>
        <form onSubmit={onSubmit} className="flex flex-col gap-4">
          <Field label="Email">
            <input
              type="email"
              autoComplete="email"
              required
              maxLength={254}
              value={email}
              onChange={(e) => setEmail(e.target.value)}
              className={inputCls}
            />
          </Field>
          <Field label="Password" hint={mode === 'signup' ? 'At least 8 characters.' : undefined}>
            <input
              type={showPw ? 'text' : 'password'}
              autoComplete={mode === 'login' ? 'current-password' : 'new-password'}
              required
              minLength={mode === 'signup' ? 8 : undefined}
              value={password}
              onChange={(e) => setPassword(e.target.value)}
              className={inputCls}
            />
          </Field>
          <div className="-mt-2 flex justify-end">
            <button
              type="button"
              onClick={() => setShowPw((v) => !v)}
              className="cursor-pointer text-[13px] font-semibold text-eucalyptus hover:underline"
            >
              {showPw ? 'Hide password' : 'Show password'}
            </button>
          </div>
          {active.isError && <ErrorNote message={errorMessage(active.error)} />}
          <button type="submit" disabled={active.isPending} className={primaryBtnCls}>
            {active.isPending ? 'Please wait…' : mode === 'login' ? 'Log in' : 'Sign up'}
          </button>
        </form>
        {active.isPending && (
          <LoadingModal
            label={mode === 'login' ? 'Logging you in…' : 'Creating your account…'}
          />
        )}
      </Card>
      <p className="mt-4 text-center text-sm text-sage">
        {mode === 'login' ? (
          <>
            New here?{' '}
            <Link to="/signup" className="font-semibold text-eucalyptus hover:underline">
              Sign up
            </Link>
          </>
        ) : (
          <>
            Already have an account?{' '}
            <Link to="/login" className="font-semibold text-eucalyptus hover:underline">
              Log in
            </Link>
          </>
        )}
      </p>
      <footer className="mt-8 text-center text-[13px] text-fog">
        Flatmate demo — profiles are unverified. Scores advise; you decide.
      </footer>
    </div>
  )
}

export function LoginPage() {
  return <AuthForm mode="login" />
}

export function SignupPage() {
  return <AuthForm mode="signup" />
}
