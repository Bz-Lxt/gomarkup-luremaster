import type { APIEnvelope, TokenUserData } from './types'

export const ACCESS_KEY = 'lm_access'
export const REFRESH_KEY = 'lm_refresh'

const API_BASE = import.meta.env.VITE_API_BASE ?? ''

export class ApiError extends Error {
  code: string
  constructor(code: string, message: string) {
    super(message)
    this.code = code
    this.name = 'ApiError'
  }
}

export function getAccess() {
  return localStorage.getItem(ACCESS_KEY) ?? ''
}

export function getRefresh() {
  return localStorage.getItem(REFRESH_KEY) ?? ''
}

export function setTokens(access: string, refresh: string) {
  localStorage.setItem(ACCESS_KEY, access)
  localStorage.setItem(REFRESH_KEY, refresh)
}

export function clearTokens() {
  localStorage.removeItem(ACCESS_KEY)
  localStorage.removeItem(REFRESH_KEY)
}

async function parseEnvelope<T>(res: Response): Promise<T> {
  let body: APIEnvelope<T> | null = null
  try {
    body = (await res.json()) as APIEnvelope<T>
  } catch {
    throw new ApiError('INTERNAL', '响应无法解析')
  }
  if (!body || typeof body !== 'object') {
    throw new ApiError('INTERNAL', '响应格式错误')
  }
  if (body.ok) return body.data
  throw new ApiError(body.error?.code ?? 'INTERNAL', body.error?.message ?? '请求失败')
}

async function tryRefresh(): Promise<boolean> {
  const refresh = getRefresh()
  if (!refresh) return false
  try {
    const res = await fetch(`${API_BASE}/api/v1/auth/refresh`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ refresh_token: refresh }),
    })
    const data = await parseEnvelope<TokenUserData>(res)
    setTokens(data.access_token, data.refresh_token)
    return true
  } catch {
    clearTokens()
    return false
  }
}

export async function api<T>(path: string, init: RequestInit = {}, retry = true): Promise<T> {
  const headers = new Headers(init.headers)
  const isForm = init.body instanceof FormData
  if (!isForm && init.body && !headers.has('Content-Type')) {
    headers.set('Content-Type', 'application/json')
  }
  const token = getAccess()
  if (token) headers.set('Authorization', `Bearer ${token}`)

  const res = await fetch(`${API_BASE}${path}`, { ...init, headers })
  if (res.status === 401 && retry && !path.includes('/auth/')) {
    const ok = await tryRefresh()
    if (ok) return api<T>(path, init, false)
  }
  return parseEnvelope<T>(res)
}

export function apiGet<T>(path: string) {
  return api<T>(path)
}

export function apiPost<T>(path: string, body?: unknown) {
  return api<T>(path, {
    method: 'POST',
    body: body === undefined ? undefined : JSON.stringify(body),
  })
}
