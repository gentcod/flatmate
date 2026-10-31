import { useState } from 'react'
import type { FormEvent } from 'react'
import { Link, useNavigate } from 'react-router-dom'
import {
  Card,
  ErrorNote,
  Field,
  PageTitle,
  errorMessage,
  inputCls,
  primaryBtnCls,
} from '../components'
import { useLogin, useSignup } from '../query'

function AuthForm({ mode }: { mode: 'login' | 'signup' }) {
  const [email, setEmail] = useState('')
  const [password, setPassword] = useState('')
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
              type="password"
              autoComplete={mode === 'login' ? 'current-password' : 'new-password'}
              required
              minLength={mode === 'signup' ? 8 : undefined}
              value={password}
              onChange={(e) => setPassword(e.target.value)}
              className={inputCls}
            />
          </Field>
          {active.isError && <ErrorNote message={errorMessage(active.error)} />}
          <button type="submit" disabled={active.isPending} className={primaryBtnCls}>
            {active.isPending ? 'Please wait…' : mode === 'login' ? 'Log in' : 'Sign up'}
          </button>
        </form>
      </Card>
      <p className="mt-4 text-center text-sm text-stone-600">
        {mode === 'login' ? (
          <>
            New here?{' '}
            <Link to="/signup" className="font-semibold text-emerald-800 hover:underline">
              Sign up
            </Link>
          </>
        ) : (
          <>
            Already have an account?{' '}
            <Link to="/login" className="font-semibold text-emerald-800 hover:underline">
              Log in
            </Link>
          </>
        )}
      </p>
    </div>
  )
}

export function LoginPage() {
  return <AuthForm mode="login" />
}

export function SignupPage() {
  return <AuthForm mode="signup" />
}
