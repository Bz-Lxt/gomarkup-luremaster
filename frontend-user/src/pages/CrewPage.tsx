import { useCallback, useState, useEffect } from 'react'
import { apiGet, apiPost, ApiError } from '../api/client'
import type { ActivityDTO, CheckinDTO, SlotDTO } from '../api/types'
import { Button } from '../components/ui/Button'
import { TextInput } from '../components/ui/Field'
import { Modal } from '../components/ui/Modal'
import { EmptyState, ErrorState, Spinner } from '../components/ui/States'
import { useAuth } from '../context/AuthContext'
import { useToast } from '../context/ToastContext'
import { ACTIVITY_KIND_LABEL, SLOT_STATUS_LABEL, labelOf } from '../lib/enums'
import { formatDateTime } from '../lib/time'

export function CrewPage() {
  const { user } = useAuth()
  const { toast } = useToast()
  const [list, setList] = useState<ActivityDTO[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')
  const [busy, setBusy] = useState<string | null>(null)
  const [checkinAct, setCheckinAct] = useState<ActivityDTO | null>(null)
  const [manual, setManual] = useState(false)
  const [lat, setLat] = useState('')
  const [lon, setLon] = useState('')
  const [latErr, setLatErr] = useState('')
  const [lonErr, setLonErr] = useState('')

  const load = useCallback(async () => {
    setLoading(true)
    setError('')
    try {
      setList(await apiGet<ActivityDTO[]>('/api/v1/activities'))
    } catch (e) {
      setError(e instanceof ApiError ? e.message : '活动装载失败')
    } finally {
      setLoading(false)
    }
  }, [])

  useEffect(() => {
    load()
  }, [load])

  async function slotOp(actId: string, sid: string, op: 'claim' | 'confirm' | 'release') {
    setBusy(sid + op)
    try {
      const updated = await apiPost<SlotDTO>(`/api/v1/activities/${actId}/slots/${sid}/${op}`)
      setList((xs) =>
        xs.map((a) =>
          a.id === actId ? { ...a, slots: a.slots.map((s) => (s.id === sid ? { ...s, ...updated } : s)) } : a,
        ),
      )
      toast(op === 'claim' ? '已抢位' : op === 'confirm' ? '已确认' : '已释放', 'ok')
    } catch (e) {
      toast(e instanceof ApiError ? e.message : '操作失败', 'err')
    } finally {
      setBusy(null)
    }
  }

  function startCheckin(act: ActivityDTO) {
    setCheckinAct(act)
    setManual(false)
    setLat('')
    setLon('')
    if (!navigator.geolocation) {
      setManual(true)
      toast('浏览器不支持定位，请手填经纬度', 'info')
      return
    }
    navigator.geolocation.getCurrentPosition(
      (pos) => {
        void doCheckin(act.id, pos.coords.latitude, pos.coords.longitude)
      },
      () => {
        setManual(true)
        toast('定位失败，改用手填经纬度', 'info')
      },
      { enableHighAccuracy: true, timeout: 8000 },
    )
  }

  async function doCheckin(id: string, la: number, lo: number) {
    setBusy('checkin')
    try {
      await apiPost<CheckinDTO>(`/api/v1/activities/${id}/checkin`, { lat: la, lon: lo })
      toast('打卡成功', 'ok')
      setCheckinAct(null)
      await load()
    } catch (e) {
      toast(e instanceof ApiError ? e.message : '打卡失败', 'err')
    } finally {
      setBusy(null)
    }
  }

  function submitManual() {
    const la = Number(lat)
    const lo = Number(lon)
    let ok = true
    if (Number.isNaN(la) || la < -90 || la > 90) {
      setLatErr('纬度须在 -90 到 90')
      ok = false
    } else setLatErr('')
    if (Number.isNaN(lo) || lo < -180 || lo > 180) {
      setLonErr('经度须在 -180 到 180')
      ok = false
    } else setLonErr('')
    if (!ok || !checkinAct) return
    void doCheckin(checkinAct.id, la, lo)
  }

  if (loading) return <Spinner label="清点席位…" />
  if (error) return <ErrorState message={error} onRetry={load} />
  if (!list.length) return <EmptyState title="暂无开放活动" hint="等船长放位，或自己发起一场。" />

  return (
    <div className="w-full p-4 md:p-6">
      <p className="font-mono text-[10px] tracking-[0.22em] text-mute">CREW LOCK</p>
      <h1 className="mb-6 font-display text-4xl italic">抢位组队</h1>
      <div className="space-y-4">
        {list.map((act) => (
          <section key={act.id} className="panel p-5">
            <div className="flex flex-wrap items-start justify-between gap-3">
              <div>
                <h2 className="font-display text-2xl italic">{act.title}</h2>
                <p className="text-xs text-mute">
                  {labelOf(ACTIVITY_KIND_LABEL, act.kind)} · {act.status} · {formatDateTime(act.starts_at)} →{' '}
                  {formatDateTime(act.ends_at)}
                </p>
                <p className="mt-1 font-mono text-xs text-tide">
                  集合 {act.meet_lat.toFixed(4)}, {act.meet_lon.toFixed(4)} · 半径 {act.meet_radius_m}m
                </p>
              </div>
              <Button variant="tide" onClick={() => startCheckin(act)} disabled={busy === 'checkin'}>
                集合打卡
              </Button>
            </div>
            <div className="mt-4 grid grid-cols-1 gap-2 xs:grid-cols-3">
              {act.slots.map((s) => {
                const mine = user && s.holder_id === user.id
                return (
                  <div key={s.id} className="rounded-sm border border-foam/10 bg-ink/40 p-3">
                    <p className="font-display text-xl italic">{s.label}</p>
                    <p className="font-mono text-xs text-mute">{labelOf(SLOT_STATUS_LABEL, s.status)}</p>
                    <div className="mt-3 flex flex-wrap gap-1">
                      <Button
                        variant="sonar"
                        className="px-2 py-1 text-xs"
                        disabled={!!busy || s.status !== 'OPEN'}
                        onClick={() => slotOp(act.id, s.id, 'claim')}
                      >
                        抢位
                      </Button>
                      <Button
                        variant="copper"
                        className="px-2 py-1 text-xs"
                        disabled={!!busy || !mine || s.status !== 'LOCKED'}
                        onClick={() => slotOp(act.id, s.id, 'confirm')}
                      >
                        确认
                      </Button>
                      <Button
                        variant="ghost"
                        className="px-2 py-1 text-xs"
                        disabled={!!busy || !mine}
                        onClick={() => slotOp(act.id, s.id, 'release')}
                      >
                        释放
                      </Button>
                    </div>
                  </div>
                )
              })}
            </div>
          </section>
        ))}
      </div>

      <Modal
        open={!!checkinAct && manual}
        title="手填打卡坐标"
        onClose={() => setCheckinAct(null)}
        confirmLabel="打卡"
        busy={busy === 'checkin'}
        onConfirm={submitManual}
      >
        <div className="space-y-3">
          <p className="text-xs text-mute">定位失败时，按集合点附近填写经纬度。</p>
          <TextInput label="纬度" value={lat} onChange={(e) => setLat(e.target.value)} error={latErr} />
          <TextInput label="经度" value={lon} onChange={(e) => setLon(e.target.value)} error={lonErr} />
        </div>
      </Modal>
    </div>
  )
}
