import { useEffect } from 'react'
import { Button } from './Button'

interface Props {
  open: boolean
  title: string
  children: React.ReactNode
  confirmLabel?: string
  cancelLabel?: string
  busy?: boolean
  hideFooter?: boolean
  onConfirm?: () => void
  onClose: () => void
}

export function Modal({
  open,
  title,
  children,
  confirmLabel = '确认',
  cancelLabel = '取消',
  busy,
  hideFooter,
  onConfirm,
  onClose,
}: Props) {
  useEffect(() => {
    if (!open) return
    const onKey = (e: KeyboardEvent) => {
      if (e.key === 'Escape') onClose()
    }
    window.addEventListener('keydown', onKey)
    return () => window.removeEventListener('keydown', onKey)
  }, [open, onClose])

  if (!open) return null

  return (
    <div className="fixed inset-0 z-[70] flex items-center justify-center bg-ink/70 px-4">
      <button type="button" className="absolute inset-0 cursor-default" aria-label="关闭遮罩" onClick={onClose} />
      <div className="panel relative z-[71] w-full max-w-lg p-5">
        <div className="mb-4 flex items-start justify-between gap-3">
          <h3 className="font-display text-xl italic text-foam">{title}</h3>
          <button type="button" className="font-mono text-xs text-mute hover:text-foam" onClick={onClose}>
            关闭
          </button>
        </div>
        <div className="text-sm text-foam/90">{children}</div>
        {!hideFooter && (
          <div className="mt-5 flex justify-end gap-2">
            <Button variant="ghost" onClick={onClose} disabled={busy}>
              {cancelLabel}
            </Button>
            {onConfirm && (
              <Button variant="copper" onClick={onConfirm} disabled={busy}>
                {busy ? '处理中…' : confirmLabel}
              </Button>
            )}
          </div>
        )}
      </div>
    </div>
  )
}
