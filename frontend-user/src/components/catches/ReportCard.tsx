import { useRef, useState } from 'react'
import { toPng } from 'html-to-image'
import type { CatchDTO, SpotDTO } from '../../api/types'
import {
  LURE_LABEL,
  MOON_LABEL,
  PRESSURE_LABEL,
  SPECIES_LABEL,
  TIDE_LABEL,
  labelOf,
} from '../../lib/enums'
import { formatDateTime } from '../../lib/time'
import { Button } from '../ui/Button'
import { useToast } from '../../context/ToastContext'

interface Props {
  rec: CatchDTO
  spot?: SpotDTO
}

export function ReportCard({ rec, spot }: Props) {
  const ref = useRef<HTMLDivElement>(null)
  const { toast } = useToast()
  const [busy, setBusy] = useState(false)
  const hydro = rec.hydro ?? { status: rec.hydro_status }
  const score = hydro.bite_score ?? 0

  async function exportPng() {
    if (!ref.current) return
    setBusy(true)
    try {
      const url = await toPng(ref.current, { pixelRatio: 2, cacheBust: true, backgroundColor: '#0E2420' })
      const a = document.createElement('a')
      a.href = url
      a.download = `lure-report-${rec.id.slice(0, 8)}.png`
      a.click()
      toast('战报已导出', 'ok')
    } catch {
      toast('导出失败，请重试', 'err')
    } finally {
      setBusy(false)
    }
  }

  return (
    <article className="space-y-3">
      <div
        ref={ref}
        className="copper-edge panel relative overflow-hidden p-5"
        style={{ backgroundImage: 'linear-gradient(160deg, #122e28 0%, #0E2420 55%, #0a1c18 100%)' }}
      >
        <div className="pointer-events-none absolute -right-6 -top-8 h-28 w-28 rounded-full border border-sonar/20" />
        <div className="flex items-start justify-between gap-3">
          <div>
            <p className="font-mono text-[10px] tracking-[0.22em] text-mute">NIGHT REPORT</p>
            <h3 className="font-display text-2xl italic">{labelOf(SPECIES_LABEL, rec.species)}</h3>
            <p className="text-xs text-mute">{spot?.name ?? '未知标点'} · {formatDateTime(rec.caught_at)}</p>
          </div>
          {hydro.frenzy && (
            <span className="rounded-sm border border-sonar bg-sonar/15 px-2 py-1 font-display text-sm italic text-sonar">
              疯狂开口
            </span>
          )}
        </div>

        <div className="mt-4 flex items-end justify-between">
          <div>
            <p className="font-mono text-[10px] text-mute">LENGTH</p>
            <p className="font-display text-6xl italic leading-none text-foam">
              {rec.length_cm}
              <span className="ml-1 font-body text-base not-italic text-mute">cm</span>
            </p>
          </div>
          <div className="text-right">
            <p className="font-mono text-3xl text-sonar">{score ? score.toFixed(0) : '—'}</p>
            <p className="text-xs text-mute">咬口分</p>
          </div>
        </div>

        <div className="mt-5 grid grid-cols-2 gap-3 text-xs md:grid-cols-4">
          <Meta k="拟饵" v={`${labelOf(LURE_LABEL, rec.lure_type)} ${rec.lure_color}`} color={rec.lure_color} />
          <Meta k="水深" v={rec.water_depth_m != null ? `${rec.water_depth_m} m` : '—'} />
          <Meta k="风向" v={hydro.wind_dir_label ?? '—'} />
          <Meta k="气压" v={labelOf(PRESSURE_LABEL, hydro.pressure_trend ?? '')} />
          <Meta k="潮汐" v={labelOf(TIDE_LABEL, hydro.tide_window ?? '')} />
          <Meta k="月相" v={labelOf(MOON_LABEL, hydro.moon_phase ?? '')} />
          <Meta k="放流" v={rec.released ? '是' : '否'} />
          <Meta k="绑定" v={hydro.status ?? rec.hydro_status} />
        </div>

        <MoonRing phase={hydro.moon_phase} illum={hydro.moon_illum_pct} />
      </div>
      <div className="flex justify-end">
        <Button variant="tide" onClick={exportPng} disabled={busy}>
          {busy ? '渲图中…' : '导出 PNG'}
        </Button>
      </div>
    </article>
  )
}

function Meta({ k, v, color }: { k: string; v: string; color?: string }) {
  return (
    <div className="rounded-sm border border-foam/10 bg-ink/30 px-2 py-2">
      <p className="text-mute">{k}</p>
      <p className="mt-0.5 flex items-center gap-1.5 font-mono">
        {color ? <span className="h-2.5 w-2.5 rounded-sm border border-foam/20" style={{ background: guessColor(color) }} /> : null}
        {v || '—'}
      </p>
    </div>
  )
}

function MoonRing({ phase, illum }: { phase?: string; illum?: number }) {
  const pct = Math.max(0, Math.min(100, illum ?? 50))
  return (
    <div className="mt-4 flex items-center gap-3">
      <div
        className="h-10 w-10 rounded-full border border-foam/30"
        style={{
          background: `linear-gradient(90deg, #E8DCC4 ${pct}%, #07120F ${pct}%)`,
          boxShadow: '0 0 12px rgba(232,220,196,0.2)',
        }}
      />
      <p className="font-mono text-xs text-mute">
        {labelOf(MOON_LABEL, phase ?? '')} · 照度 {pct.toFixed(0)}%
      </p>
    </div>
  )
}

function guessColor(name: string) {
  if (name.includes('银') || name.includes('白')) return '#d9d4c4'
  if (name.includes('金')) return '#F2C14E'
  if (name.includes('橙') || name.includes('红')) return '#D4783A'
  if (name.includes('青') || name.includes('绿')) return '#3F7D4E'
  if (name.includes('暗') || name.includes('黑')) return '#2a2a2a'
  if (name.includes('荧')) return '#C6F04A'
  return '#5EC8D8'
}
