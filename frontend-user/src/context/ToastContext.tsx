import { createContext, useCallback, useContext, useMemo, useRef, useState } from 'react'

export type ToastKind = 'ok' | 'err' | 'info'

interface ToastItem {
  id: number
  kind: ToastKind
  message: string
}

interface ToastCtx {
  toast: (message: string, kind?: ToastKind) => void
}

const Ctx = createContext<ToastCtx | null>(null)

export function ToastProvider({ children }: { children: React.ReactNode }) {
  const [items, setItems] = useState<ToastItem[]>([])
  const seq = useRef(1)

  const dismiss = useCallback((id: number) => {
    setItems((xs) => xs.filter((t) => t.id !== id))
  }, [])

  const toast = useCallback((message: string, kind: ToastKind = 'info') => {
    const id = seq.current++
    setItems((xs) => [...xs, { id, kind, message }])
    window.setTimeout(() => dismiss(id), 5000)
  }, [dismiss])

  const value = useMemo(() => ({ toast }), [toast])

  return (
    <Ctx.Provider value={value}>
      {children}
      <div className="fixed right-4 top-4 z-[80] flex w-[min(92vw,360px)] flex-col gap-2">
        {items.map((t) => (
          <div
            key={t.id}
            className={`panel flex items-start gap-3 px-4 py-3 text-sm ${
              t.kind === 'err' ? 'border-copper/50' : t.kind === 'ok' ? 'border-sonar/40' : 'border-tide/30'
            }`}
          >
            <span
              className={`mt-1 h-2 w-2 shrink-0 rounded-full ${
                t.kind === 'err' ? 'bg-copper' : t.kind === 'ok' ? 'bg-sonar' : 'bg-tide'
              }`}
            />
            <p className="flex-1 leading-relaxed">{t.message}</p>
            <button
              type="button"
              className="font-mono text-xs text-mute hover:text-foam"
              onClick={() => dismiss(t.id)}
            >
              关闭
            </button>
          </div>
        ))}
      </div>
    </Ctx.Provider>
  )
}

export function useToast() {
  const ctx = useContext(Ctx)
  if (!ctx) throw new Error('useToast outside provider')
  return ctx
}
