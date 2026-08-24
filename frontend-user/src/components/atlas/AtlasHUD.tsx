import { STRUCTURE_COLOR, STRUCTURE_LABEL, STRUCTURES } from '../../lib/enums'

interface Props {
  count: number
  pressure?: string
  tide?: string
  score?: number
  active: Set<string>
  onToggle: (key: string) => void
  onAll: () => void
}

export function AtlasHUD({ count, pressure, tide, score, active, onToggle, onAll }: Props) {
  return (
    <div className="pointer-events-none absolute inset-0 z-10">
      <div className="pointer-events-auto absolute left-3 top-3 animate-fadeUp panel px-4 py-3" style={{ animationDelay: '0ms' }}>
        <p className="font-mono text-[10px] tracking-[0.2em] text-mute">SPOTS</p>
        <p className="font-display text-3xl italic leading-none text-sonar">{count}</p>
        <p className="mt-1 text-xs text-mute">枚铜钉标点</p>
      </div>

      <div
        className="pointer-events-auto absolute right-3 top-3 animate-fadeUp panel w-[min(46vw,220px)] px-4 py-3"
        style={{ animationDelay: '80ms' }}
      >
        <p className="font-mono text-[10px] tracking-[0.2em] text-mute">HYDRO MINI</p>
        <div className="mt-2 space-y-2 text-xs">
          <div>
            <div className="flex justify-between text-mute">
              <span>气压</span>
              <span className="font-mono text-pressure">{pressure ?? '—'}</span>
            </div>
            <div className="mt-1 h-1.5 overflow-hidden rounded-full bg-ink">
              <div className="h-full w-2/3 bg-pressure/80" />
            </div>
          </div>
          <div>
            <div className="flex justify-between text-mute">
              <span>潮汐</span>
              <span className="font-mono text-tide">{tide ?? '—'}</span>
            </div>
            <div className="mt-1 h-1.5 overflow-hidden rounded-full bg-ink">
              <div className="h-full w-1/2 bg-tide/80" />
            </div>
          </div>
          <p className="font-mono text-sonar">咬口 {score != null ? score.toFixed(0) : '—'}</p>
        </div>
      </div>

      <div
        className="pointer-events-auto absolute inset-x-0 bottom-3 mx-3 animate-fadeUp overflow-x-auto panel px-3 py-2"
        style={{ animationDelay: '160ms' }}
      >
        <div className="flex min-w-max items-center gap-2">
          <button
            type="button"
            onClick={onAll}
            className="rounded-full border border-foam/20 px-3 py-1 text-xs text-foam hover:border-sonar"
          >
            全部
          </button>
          {STRUCTURES.map((s) => {
            const on = active.has(s)
            return (
              <button
                key={s}
                type="button"
                onClick={() => onToggle(s)}
                className={`flex items-center gap-1.5 rounded-full border px-3 py-1 text-xs ${
                  on ? 'border-foam/40 text-foam' : 'border-transparent text-mute opacity-50'
                }`}
              >
                <span className="h-2 w-2 rounded-full" style={{ background: STRUCTURE_COLOR[s] }} />
                {STRUCTURE_LABEL[s]}
              </button>
            )
          })}
        </div>
      </div>
    </div>
  )
}
