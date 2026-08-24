import { useEffect, useState } from 'react'
import { apiGet, apiPost, ApiError } from '../api/client'
import type { AdviceDTO, SpotDTO } from '../api/types'
import { Button } from '../components/ui/Button'
import { Select, TextInput } from '../components/ui/Field'
import { EmptyState, ErrorState } from '../components/ui/States'
import { useToast } from '../context/ToastContext'
import {
  LAYER_LABEL,
  PRESSURE_LABEL,
  PRESSURE_TRENDS,
  RETRIEVE_LABEL,
  SPECIES,
  SPECIES_LABEL,
  TIDE_LABEL,
  TIDE_WINDOWS,
  labelOf,
} from '../lib/enums'

export function LuresPage() {
  const { toast } = useToast()
  const [spots, setSpots] = useState<SpotDTO[]>([])
  const [species, setSpecies] = useState('YELLOWCHECK')
  const [spotId, setSpotId] = useState('')
  const [trend, setTrend] = useState('CRASH_DOWN')
  const [tide, setTide] = useState('THIRD')
  const [temp, setTemp] = useState('22')
  const [list, setList] = useState<AdviceDTO[] | null>(null)
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState('')

  useEffect(() => {
    apiGet<SpotDTO[]>('/api/v1/spots')
      .then((xs) => {
        setSpots(xs)
        if (xs[0]) setSpotId(xs[0].id)
      })
      .catch((e) => setError(e instanceof ApiError ? e.message : '标点加载失败'))
  }, [])

  async function recommend() {
    setBusy(true)
    setError('')
    try {
      const data = await apiPost<AdviceDTO[]>('/api/v1/lures/recommend', {
        species,
        spot_id: spotId,
        pressure_trend: trend,
        tide_window: tide,
        water_temp_c: temp ? Number(temp) : null,
      })
      setList(data)
      toast('推荐已算出', 'ok')
    } catch (e) {
      setError(e instanceof ApiError ? e.message : '推荐失败')
      toast(e instanceof ApiError ? e.message : '推荐失败', 'err')
    } finally {
      setBusy(false)
    }
  }

  return (
    <div className="w-full p-4 md:p-6">
      <p className="font-mono text-[10px] tracking-[0.22em] text-mute">LURE BAYES</p>
      <h1 className="mb-6 font-display text-4xl italic">拟饵推荐</h1>
      <div className="grid w-full grid-cols-1 gap-3 xs:grid-cols-2 md:grid-cols-3">
        <Select label="目标鱼种" value={species} onChange={(e) => setSpecies(e.target.value)}>
          {SPECIES.map((s) => (
            <option key={s} value={s}>
              {SPECIES_LABEL[s]}
            </option>
          ))}
        </Select>
        <Select label="标点" value={spotId} onChange={(e) => setSpotId(e.target.value)}>
          {spots.map((s) => (
            <option key={s.id} value={s.id}>
              {s.name}
            </option>
          ))}
        </Select>
        <Select label="气压趋势" value={trend} onChange={(e) => setTrend(e.target.value)}>
          {PRESSURE_TRENDS.map((s) => (
            <option key={s} value={s}>
              {PRESSURE_LABEL[s]}
            </option>
          ))}
        </Select>
        <Select label="潮汐窗口" value={tide} onChange={(e) => setTide(e.target.value)}>
          {TIDE_WINDOWS.map((s) => (
            <option key={s} value={s}>
              {TIDE_LABEL[s]}
            </option>
          ))}
        </Select>
        <TextInput label="水温 °C" value={temp} onChange={(e) => setTemp(e.target.value)} />
      </div>
      <div className="mt-5 flex flex-wrap items-center gap-3">
        <Button variant="copper" onClick={recommend} disabled={busy}>
          {busy ? '推演中…' : '生成推荐'}
        </Button>
        <span className="font-mono text-xs text-mute">规则引擎 · 本地计算 · ¥0</span>
      </div>
      {error ? <ErrorState message={error} onRetry={recommend} /> : null}
      {list && !list.length ? <EmptyState title="没有推荐" hint="换一组水文条件再试。" /> : null}
      {list && list.length > 0 && (
        <ol className="mt-6 space-y-3">
          {list.map((a, i) => (
            <li key={`${a.lure_type}-${i}`} className="panel flex flex-wrap items-start justify-between gap-4 p-4">
              <div>
                <p className="font-mono text-[10px] text-mute">TOP {i + 1}</p>
                <p className="font-display text-2xl italic">
                  {a.label || a.lure_type} · {a.color}
                </p>
                <p className="mt-1 text-sm text-mute">
                  {labelOf(LAYER_LABEL, a.layer)} · {labelOf(RETRIEVE_LABEL, a.retrieve)}
                </p>
                <p className="mt-2 text-sm leading-relaxed text-foam/90">{a.reason}</p>
              </div>
              <p className="font-mono text-3xl text-sonar">{a.score.toFixed(1)}</p>
            </li>
          ))}
        </ol>
      )}
    </div>
  )
}
