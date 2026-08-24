import { createContext, useCallback, useContext, useEffect, useMemo, useState } from 'react'
import { apiGet, apiPost, clearTokens, getAccess, setTokens } from '../api/client'
import type { TokenUserData, UserDTO } from '../api/types'

interface AuthCtx {
  user: UserDTO | null
  ready: boolean
  login: (email: string, password: string) => Promise<void>
  logout: () => void
  refreshMe: () => Promise<void>
}

const Ctx = createContext<AuthCtx | null>(null)

export function AuthProvider({ children }: { children: React.ReactNode }) {
  const [user, setUser] = useState<UserDTO | null>(null)
  const [ready, setReady] = useState(false)

  const refreshMe = useCallback(async () => {
    if (!getAccess()) {
      setUser(null)
      return
    }
    const me = await apiGet<UserDTO>('/api/v1/me')
    setUser(me)
  }, [])

  useEffect(() => {
    refreshMe()
      .catch(() => {
        clearTokens()
        setUser(null)
      })
      .finally(() => setReady(true))
  }, [refreshMe])

  const login = useCallback(async (email: string, password: string) => {
    const data = await apiPost<TokenUserData>('/api/v1/auth/login', { email, password })
    setTokens(data.access_token, data.refresh_token)
    setUser(data.user)
  }, [])

  const logout = useCallback(() => {
    clearTokens()
    setUser(null)
  }, [])

  const value = useMemo(() => ({ user, ready, login, logout, refreshMe }), [user, ready, login, logout, refreshMe])
  return <Ctx.Provider value={value}>{children}</Ctx.Provider>
}

export function useAuth() {
  const ctx = useContext(Ctx)
  if (!ctx) throw new Error('useAuth outside provider')
  return ctx
}
