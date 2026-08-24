import { Navigate, Route, Routes } from 'react-router-dom'
import { AppShell } from './components/layout/AppShell'
import { Spinner } from './components/ui/States'
import { useAuth } from './context/AuthContext'
import { AtlasPage } from './pages/AtlasPage'
import { CatchesPage } from './pages/CatchesPage'
import { CrewPage } from './pages/CrewPage'
import { LoginPage } from './pages/LoginPage'
import { LuresPage } from './pages/LuresPage'
import { NewCatchPage } from './pages/NewCatchPage'
import { PlaceholderPage } from './pages/PlaceholderPage'
import { StatsPage } from './pages/StatsPage'

function Guard({ children }: { children: React.ReactNode }) {
  const { user, ready } = useAuth()
  if (!ready) return <Spinner />
  if (!user) return <Navigate to="/" replace />
  return <>{children}</>
}

export function App() {
  return (
    <Routes>
      <Route path="/" element={<LoginPage />} />
      <Route path="/privacy" element={<PlaceholderPage title="隐私政策" />} />
      <Route path="/terms" element={<PlaceholderPage title="使用条款" />} />
      <Route
        element={
          <Guard>
            <AppShell />
          </Guard>
        }
      >
        <Route path="/atlas" element={<AtlasPage />} />
        <Route path="/catches" element={<CatchesPage />} />
        <Route path="/catches/new" element={<NewCatchPage />} />
        <Route path="/crew" element={<CrewPage />} />
        <Route path="/lures" element={<LuresPage />} />
        <Route path="/stats" element={<StatsPage />} />
      </Route>
      <Route path="*" element={<Navigate to="/" replace />} />
    </Routes>
  )
}
