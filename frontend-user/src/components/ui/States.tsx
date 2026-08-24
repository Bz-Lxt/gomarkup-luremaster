export function Spinner({ label = '装载海图…' }: { label?: string }) {
  return (
    <div className="flex flex-col items-center justify-center gap-3 py-16 text-mute">
      <div className="h-8 w-8 animate-spin rounded-full border-2 border-foam/15 border-t-sonar" />
      <p className="font-mono text-xs tracking-widest">{label}</p>
    </div>
  )
}

export function EmptyState({ title, hint }: { title: string; hint?: string }) {
  return (
    <div className="panel mx-4 my-8 flex flex-col items-center gap-2 px-6 py-14 text-center">
      <p className="font-display text-2xl italic text-foam">{title}</p>
      {hint ? <p className="text-sm text-mute">{hint}</p> : null}
    </div>
  )
}

export function ErrorState({ message, onRetry }: { message: string; onRetry?: () => void }) {
  return (
    <div className="panel mx-4 my-8 flex flex-col items-center gap-3 px-6 py-12 text-center">
      <p className="font-display text-2xl italic text-copper">海图中断</p>
      <p className="text-sm text-mute">{message}</p>
      {onRetry ? (
        <button
          type="button"
          onClick={onRetry}
          className="rounded-sm bg-copper px-4 py-2 text-sm text-ink hover:brightness-110"
        >
          重新探测
        </button>
      ) : null}
    </div>
  )
}
