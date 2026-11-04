import { Suspense, lazy, useEffect } from 'react'
import { BrowserRouter, Link, Route, Routes } from 'react-router-dom'
import { bootCsrf } from './api'
import { Layout, Loading, RequireAuth } from './components'
import { AppProviders } from './query'

const AdminPage = lazy(() => import('./pages/Admin').then((m) => ({ default: m.AdminPage })))
const ChatPage = lazy(() => import('./pages/Chat').then((m) => ({ default: m.ChatPage })))
const ExpensesPage = lazy(() =>
  import('./pages/Expenses').then((m) => ({ default: m.ExpensesPage })),
)
const InterviewPage = lazy(() =>
  import('./pages/Interview').then((m) => ({ default: m.InterviewPage })),
)
const LandingPage = lazy(() =>
  import('./pages/Landing').then((m) => ({ default: m.LandingPage })),
)
const ListingDetailPage = lazy(() =>
  import('./pages/ListingDetail').then((m) => ({ default: m.ListingDetailPage })),
)
const ListingsPage = lazy(() =>
  import('./pages/Listings').then((m) => ({ default: m.ListingsPage })),
)
const LoginPage = lazy(() => import('./pages/Auth').then((m) => ({ default: m.LoginPage })))
const MatchDetailPage = lazy(() =>
  import('./pages/MatchDetail').then((m) => ({ default: m.MatchDetailPage })),
)
const MatchesPage = lazy(() =>
  import('./pages/Matches').then((m) => ({ default: m.MatchesPage })),
)
const ProfilePage = lazy(() =>
  import('./pages/Profile').then((m) => ({ default: m.ProfilePage })),
)
const QuestionnairePage = lazy(() =>
  import('./pages/Questionnaire').then((m) => ({ default: m.QuestionnairePage })),
)
const SettingsPage = lazy(() =>
  import('./pages/Settings').then((m) => ({ default: m.SettingsPage })),
)
const SignupPage = lazy(() => import('./pages/Auth').then((m) => ({ default: m.SignupPage })))
const ThreadPage = lazy(() =>
  import('./pages/Thread').then((m) => ({ default: m.ThreadPage })),
)

function NotFound() {
  return (
    <div className="mx-auto max-w-md py-20 text-center">
      <h1 className="font-display text-3xl tracking-tight text-ink">Page not found</h1>
      <p className="mt-2 text-sm text-sage">
        <Link to="/" className="font-semibold text-eucalyptus hover:underline">
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
        <Suspense fallback={<Loading label="Loading this page…" />}>
          <Routes>
            <Route path="/" element={<LandingPage />} />
            <Route path="/login" element={<LoginPage />} />
            <Route path="/signup" element={<SignupPage />} />
            <Route
              element={
                <RequireAuth>
                  <Layout />
                </RequireAuth>
              }
            >
              <Route path="/matches" element={<MatchesPage />} />
              <Route path="/match/:id" element={<MatchDetailPage />} />
              <Route path="/profile" element={<ProfilePage />} />
              <Route path="/questionnaire" element={<QuestionnairePage />} />
              <Route path="/onboarding" element={<InterviewPage />} />
              <Route path="/listings" element={<ListingsPage />} />
              <Route path="/listing/:id" element={<ListingDetailPage />} />
              <Route path="/chat" element={<ChatPage />} />
              <Route path="/chat/:id" element={<ThreadPage />} />
              <Route path="/expenses/:id" element={<ExpensesPage />} />
              <Route path="/settings" element={<SettingsPage />} />
              <Route path="/admin" element={<AdminPage />} />
            </Route>
            <Route path="*" element={<NotFound />} />
          </Routes>
        </Suspense>
      </BrowserRouter>
    </AppProviders>
  )
}
