import type { SpotDTO } from '../../api/types'
import { STRUCTURE_COLOR, STRUCTURE_LABEL, VISIBILITY_LABEL, WATER_LABEL, labelOf } from '../../lib/enums'
import { formatDateTime } from '../../lib/time'
import { Button } from '../ui/Button'

interface Props {
  spot: SpotDTO | null
  onClose: () => void
  onRecord: (spotId: string) => void
}

export function SpotDrawer({ spot, onClose, onRecord }: Props) {
  if (!spot) return null
  return (
    <aside className="absolute right-0 top-0 z-20 flex h-full w-full max-w-sm flex-col border-l border-foam/10 bg-kelp/95 p-5 backdrop-blur md:max-w-sm">
      <div className="mb-4 flex items-start justify-between">
        <div>
          <p className="font-mono text-[10px] tracking-[0.2em] text-mute">SECRET MARK</p>
          <h2 className="font-display text-2xl italic">{spot.name}</h2>
        </div>
        <button type="button" className="font-mono text-xs text-mute hover:text-foam" onClick={onClose}>
          关闭
        </button>
      </div>
      <div className="space-y-3 text-sm">
        <p className="flex items-center gap-2">
          <span className="h-2.5 w-2.5 rounded-full" style={{ background: STRUCTURE_COLOR[spot.structure] }} />
          {labelOf(STRUCTURE_LABEL, spot.structure)} · {labelOf(WATER_LABEL, spot.water_type)}
        </p>
        <p className="text-mute">可见性 {labelOf(VISIBILITY_LABEL, spot.visibility)}</p>
        <p className="font-mono text-xs text-tide">
          {spot.lat.toFixed(5)}, {spot.lon.toFixed(5)}
        </p>
        {spot.fuzzed && (
          <p className="rounded-sm border border-pressure/40 bg-pressure/10 px-3 py-2 text-xs text-pressure">
            坐标已网格脱敏
          </p>
        )}
        <p className="text-mute">感潮 {spot.tidal ? '是' : '否'} · 岸向 {spot.shore_bearing}°</p>
        {spot.note ? <p className="leading-relaxed">{spot.note}</p> : null}
        {spot.depths?.length ? (
          <div>
            <p className="mb-1 text-xs text-mute">水深剖面</p>
            {spot.depths.map((d, i) => (
              <p key={i} className="font-mono text-xs">
                +{d.offset_m}m → {d.depth_m}m
              </p>
            ))}
          </div>
        ) : null}
        <p className="font-mono text-[10px] text-mute">{formatDateTime(spot.created_at)}</p>
      </div>
      <div className="mt-auto flex gap-2 pt-6">
        <Button variant="ghost" className="flex-1" onClick={onClose}>
          收起
        </Button>
        <Button variant="sonar" className="flex-1" onClick={() => onRecord(spot.id)}>
          记一口
        </Button>
      </div>
    </aside>
  )
}
