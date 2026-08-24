import { useCallback, useEffect, useState } from 'react'
import { apiGet, ApiError } from '../api/client'
import type { StatsDTO } from '../api/types'
import { EmptyState, ErrorState, Spinner } from '../components/ui/States'
import { LURE_LABEL, SPECIES_LABEL, labelOf } from '../lib/enums'

export function StatsPage() {
  const [stats, setStats] = useState<StatsDTO | null>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')

  const load = useCallback(async () => {
    setLoading(true)
    setError('')
    try {
      setStats(await apiGet<StatsDTO>('/api/v1/me/stats'))
    } catch (e) {
      setError(e instanceof ApiError ? e.message : '战绩装载失败')
    } finally {
      setLoading(false)
    }
  }, [])

  useEffect(() => {
    load()
  }, [load])

  if (loading) return <Spinner label="清算战绩…" />
  if (error) return <ErrorState message={error} onRetry={load} />
  if (!stats || stats.total_catches === 0) {
    return <EmptyState title="战绩簿还是空白" hint="第一口鱼会把数字点亮。" />
  }

  return (
    <div className="w-full p-4 md:p-6">
      <p className="font-mono text-[10px] tracking-[0.22em] text-mute">LOGBOOK</p>
      <h1 className="mb-6 font-display text-4xl italic">战绩</h1>
      <div className="grid w-full grid-cols-2 gap-3 md:grid-cols-4">
        <Stat n={String(stats.total_catches)} k="总中鱼" />
        <Stat n={String(stats.released_count)} k="放流" />
        <Stat n={`${(stats.release_rate * 100).toFixed(0)}%`} k="放流率" />
        <Stat n={String(stats.streak_days)} k="连续出钓" />
        <Stat n={`${stats.max_length_cm.toFixed(0)}cm`} k="最大体长" />
        <Stat n={labelOf(SPECIES_LABEL, stats.max_species)} k="最大鱼种" />
      </div>
      <div className="mt-6 grid w-full grid-cols-1 gap-4 md:grid-cols-2">
        <section className="panel p-5">
          <h2 className="font-display text-xl italic">最高效拟饵</h2>
          <ul className="mt-3 space-y-2">
            {stats.top_lures.map((x) => (
              <li key={x.lure_type} className="flex justify-between font-mono text-sm">
                <span>{labelOf(LURE_LABEL, x.lure_type)}</span>
                <span className="text-sonar">{x.count}</span>
              </li>
            ))}
          </ul>
        </section>
        <section className="panel p-5">
          <h2 className="font-display text-xl italic">命中标点</h2>
          <ul className="mt-3 space-y-2">
            {stats.top_spots.map((x) => (
              <li key={x.spot_id} className="flex justify-between font-mono text-sm">
                <span>{x.name}</span>
                <span className="text-tide">{x.count}</span>
              </li>
            ))}
          </ul>
        </section>
      </div>
    </div>
  )
}

function Stat({ n, k }: { n: string; k: string }) {
  return (
    <div className="panel p-4">
      <p className="font-display text-4xl italic text-sonar">{n}</p>
      <p className="mt-1 text-xs text-mute">{k}</p>
    </div>
  )
}
