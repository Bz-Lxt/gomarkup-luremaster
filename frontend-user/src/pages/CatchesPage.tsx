import { useCallback, useEffect, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { apiGet, ApiError } from '../api/client'
import type { CatchDTO, SpotDTO } from '../api/types'
import { HydroTwinChart } from '../components/catches/HydroTwinChart'
import { ReportCard } from '../components/catches/ReportCard'
import { Button } from '../components/ui/Button'
import { EmptyState, ErrorState, Spinner } from '../components/ui/States'
import { SPECIES_LABEL, labelOf } from '../lib/enums'
import { sleep } from '../lib/time'

async function hydrateCatch(id: string): Promise<CatchDTO> {
  const start = Date.now()
  let last = await apiGet<CatchDTO>(`/api/v1/catches/${id}`)
  while (Date.now() - start < 8000) {
    const status = last.hydro?.status ?? last.hydro_status
    if (status === 'BOUND' || status === 'FAILED') return last
    await sleep(700)
    last = await apiGet<CatchDTO>(`/api/v1/catches/${id}`)
  }
  return last
}

export function CatchesPage() {
  const navigate = useNavigate()
  const [list, setList] = useState<CatchDTO[]>([])
  const [spots, setSpots] = useState<Record<string, SpotDTO>>({})
  const [focus, setFocus] = useState<string | null>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')

  const load = useCallback(async () => {
    setLoading(true)
    setError('')
    try {
      const [catches, spotList] = await Promise.all([
        apiGet<CatchDTO[]>('/api/v1/catches'),
        apiGet<SpotDTO[]>('/api/v1/spots'),
      ])
      const map: Record<string, SpotDTO> = {}
      spotList.forEach((s) => {
        map[s.id] = s
      })
      setSpots(map)
      const detailed = await Promise.all(catches.map((c) => hydrateCatch(c.id).catch(() => c)))
      setList(detailed)
      if (detailed[0]) setFocus(detailed[0].id)
    } catch (e) {
      setError(e instanceof ApiError ? e.message : '战报装载失败')
    } finally {
      setLoading(false)
    }
  }, [])

  useEffect(() => {
    load()
  }, [load])

  const current = list.find((c) => c.id === focus) ?? list[0]

  if (loading) return <Spinner label="抽出战报袋…" />
  if (error) return <ErrorState message={error} onRetry={load} />
  if (!list.length) {
    return (
      <div className="w-full p-6">
        <EmptyState title="还没有战报" hint="去记一口，水文会随后贴上防水标签。" />
        <div className="flex justify-center">
          <Button onClick={() => navigate('/catches/new')}>去记口</Button>
        </div>
      </div>
    )
  }

  return (
    <div className="w-full p-4 md:p-6">
      <div className="mb-5 flex flex-wrap items-end justify-between gap-3">
        <div>
          <p className="font-mono text-[10px] tracking-[0.22em] text-mute">CATCH BOX</p>
          <h1 className="font-display text-4xl italic">战报箱</h1>
        </div>
        <Button variant="copper" onClick={() => navigate('/catches/new')}>
          记一口
        </Button>
      </div>

      <div className="grid w-full grid-cols-1 gap-6 lg:grid-cols-[360px_1fr]">
        <div className="space-y-2">
          {list.map((c) => (
            <button
              key={c.id}
              type="button"
              onClick={() => setFocus(c.id)}
              className={`panel w-full px-4 py-3 text-left ${focus === c.id ? 'border-sonar/50' : ''}`}
            >
              <p className="font-display text-lg italic">{labelOf(SPECIES_LABEL, c.species)}</p>
              <p className="font-mono text-xs text-mute">
                {c.length_cm}cm · {c.hydro?.status ?? c.hydro_status}
              </p>
            </button>
          ))}
        </div>
        <div className="space-y-6">
          {current && <ReportCard rec={current} spot={spots[current.spot_id]} />}
          {current && (
            <div className="panel p-4">
              <p className="mb-2 font-mono text-[10px] tracking-[0.2em] text-mute">PRESSURE × TIDE</p>
              <HydroTwinChart hydro={current.hydro} catchAt={current.caught_at} />
            </div>
          )}
        </div>
      </div>
    </div>
  )
}
