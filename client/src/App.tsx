import { useEffect } from 'react'
import { BrowserRouter, Link, Navigate, Route, Routes } from 'react-router-dom'
import { bootCsrf } from './api'
import { Layout, RequireAuth } from './components'
import { AppProviders } from './query'
import { LoginPage, SignupPage } from './pages/Auth'
import { InterviewPage } from './pages/Interview'
import { MatchDetailPage } from './pages/MatchDetail'
import { MatchesPage } from './pages/Matches'
import { ProfilePage } from './pages/Profile'
import { QuestionnairePage } from './pages/Questionnaire'
import { SettingsPage } from './pages/Settings'

function NotFound() {
  return (
    <div className="mx-auto max-w-md py-20 text-center">
      <h1 className="text-2xl font-bold tracking-tight text-stone-900">Page not found</h1>
      <p className="mt-2 text-sm text-stone-600">
        <Link to="/" className="font-semibold text-emerald-800 hover:underline">
          Back home
        </Link>
      </p>
    </div>
  )
}

export default function App() {
  useEffect(() => {
    bootCsrf()
  }, [])
  return (
    <AppProviders>
      <BrowserRouter>
        <Routes>
          <Route path="/login" element={<LoginPage />} />
          <Route path="/signup" element={<SignupPage />} />
          <Route
            element={
              <RequireAuth>
                <Layout />
              </RequireAuth>
            }
          >
            <Route path="/" element={<Navigate to="/matches" replace />} />
            <Route path="/matches" element={<MatchesPage />} />
            <Route path="/match/:id" element={<MatchDetailPage />} />
            <Route path="/profile" element={<ProfilePage />} />
            <Route path="/questionnaire" element={<QuestionnairePage />} />
            <Route path="/onboarding" element={<InterviewPage />} />
            <Route path="/settings" element={<SettingsPage />} />
          </Route>
          <Route path="*" element={<NotFound />} />
        </Routes>
      </BrowserRouter>
    </AppProviders>
  )
}
