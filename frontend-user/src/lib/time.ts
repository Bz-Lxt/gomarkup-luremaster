const BEIJING_OFFSET_MS = 8 * 60 * 60 * 1000

export function nowBeijing(): Date {
  return new Date(Date.now() + BEIJING_OFFSET_MS)
}

function pad(n: number) {
  return String(n).padStart(2, '0')
}

export function formatDateTime(input?: string | Date | null): string {
  if (!input) return '—'
  const d = typeof input === 'string' ? parseAny(input) : input
  if (!d || Number.isNaN(d.getTime())) return typeof input === 'string' ? input : '—'
  const bj = new Date(d.getTime() + BEIJING_OFFSET_MS)
  return `${bj.getUTCFullYear()}-${pad(bj.getUTCMonth() + 1)}-${pad(bj.getUTCDate())} ${pad(bj.getUTCHours())}:${pad(bj.getUTCMinutes())}:${pad(bj.getUTCSeconds())}`
}

export function formatLocalMinute(d = nowBeijing()): string {
  return `${d.getUTCFullYear()}-${pad(d.getUTCMonth() + 1)}-${pad(d.getUTCDate())} ${pad(d.getUTCHours())}:${pad(d.getUTCMinutes())}`
}

function parseAny(s: string): Date {
  const trimmed = s.trim()
  if (/^\d{4}-\d{2}-\d{2} \d{2}:\d{2}(:\d{2})?$/.test(trimmed)) {
    const iso = trimmed.replace(' ', 'T') + (trimmed.length === 16 ? ':00' : '') + '+08:00'
    return new Date(iso)
  }
  return new Date(trimmed)
}

export function sleep(ms: number) {
  return new Promise((r) => setTimeout(r, ms))
}
