import { useCallback, useEffect, useRef, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import maplibregl from 'maplibre-gl'
import 'maplibre-gl/dist/maplibre-gl.css'
import { apiGet, apiPost, ApiError } from '../api/client'
import type { CatchDTO, ClubDTO, CreateSpotBody, HydroDTO, SpotDTO } from '../api/types'
import { AtlasHUD } from '../components/atlas/AtlasHUD'
import { SpotDrawer } from '../components/atlas/SpotDrawer'
import { SpotForm } from '../components/atlas/SpotForm'
import { Modal } from '../components/ui/Modal'
import { ErrorState, Spinner } from '../components/ui/States'
import { Button } from '../components/ui/Button'
import { useToast } from '../context/ToastContext'
import { PRESSURE_LABEL, STRUCTURE_COLOR, STRUCTURES, TIDE_LABEL, labelOf } from '../lib/enums'

type StyleKey = 'osm' | 'dark' | 'sat'

const STYLES: Record<StyleKey, string> = {
  osm: 'https://tile.openstreetmap.org/{z}/{x}/{y}.png',
  dark: 'https://basemaps.cartocdn.com/dark_all/{z}/{x}/{y}.png',
  sat: 'https://server.arcgisonline.com/ArcGIS/rest/services/World_Imagery/MapServer/tile/{z}/{y}/{x}',
}

function rasterStyle(key: StyleKey): maplibregl.StyleSpecification {
  return {
    version: 8,
    sources: {
      raster: {
        type: 'raster',
        tiles: [STYLES[key]],
        tileSize: 256,
        attribution: '© OpenStreetMap / Carto / Esri',
      },
    },
    layers: [
      { id: 'bg', type: 'background', paint: { 'background-color': '#07120F' } },
      { id: 'raster', type: 'raster', source: 'raster' },
    ],
  }
}

export function AtlasPage() {
  const wrap = useRef<HTMLDivElement>(null)
  const mapRef = useRef<maplibregl.Map | null>(null)
  const markers = useRef<maplibregl.Marker[]>([])
  const { toast } = useToast()
  const navigate = useNavigate()

  const [spots, setSpots] = useState<SpotDTO[]>([])
  const [clubs, setClubs] = useState<ClubDTO[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')
  const [style, setStyle] = useState<StyleKey>('dark')
  const [styleRev, setStyleRev] = useState(0)
  const [active, setActive] = useState<Set<string>>(() => new Set(STRUCTURES))
  const [selected, setSelected] = useState<SpotDTO | null>(null)
  const [draft, setDraft] = useState<{ lat: number; lon: number } | null>(null)
  const [busy, setBusy] = useState(false)
  const [hydroMini, setHydroMini] = useState<HydroDTO | null>(null)

  const load = useCallback(async () => {
    setLoading(true)
    setError('')
    try {
      const [list, clubList, catches] = await Promise.all([
        apiGet<SpotDTO[]>('/api/v1/spots'),
        apiGet<ClubDTO[]>('/api/v1/clubs').catch(() => [] as ClubDTO[]),
        apiGet<CatchDTO[]>('/api/v1/catches').catch(() => [] as CatchDTO[]),
      ])
      setSpots(list)
      setClubs(clubList)
      const latest = catches[0]
      if (latest) {
        const detail = await apiGet<CatchDTO>(`/api/v1/catches/${latest.id}`).catch(() => latest)
        setHydroMini(detail.hydro)
      }
    } catch (e) {
      setError(e instanceof ApiError ? e.message : '标点装载失败')
    } finally {
      setLoading(false)
    }
  }, [])

  useEffect(() => {
    load()
  }, [load])

  useEffect(() => {
    if (!wrap.current || mapRef.current) return
    const map = new maplibregl.Map({
      container: wrap.current,
      style: rasterStyle(style),
      center: [120.692, 30.378],
      zoom: 9,
      attributionControl: { compact: true },
    })
    map.addControl(new maplibregl.NavigationControl({ showCompass: true }), 'bottom-right')
    map.on('click', (e) => {
      const t = e.originalEvent.target as HTMLElement
      if (t.closest('.lm-pin')) return
      setSelected(null)
      setDraft({ lat: e.lngLat.lat, lon: e.lngLat.lng })
    })
    mapRef.current = map
    return () => {
      map.remove()
      mapRef.current = null
    }
  }, [])

  const styleReady = useRef(false)
  useEffect(() => {
    const map = mapRef.current
    if (!map) return
    if (!styleReady.current) {
      styleReady.current = true
      return
    }
    const onLoad = () => setStyleRev((n) => n + 1)
    map.once('style.load', onLoad)
    map.setStyle(rasterStyle(style))
    return () => {
      map.off('style.load', onLoad)
    }
  }, [style])

  useEffect(() => {
    const map = mapRef.current
    if (!map) return
    markers.current.forEach((m) => m.remove())
    markers.current = []
    spots
      .filter((s) => active.has(s.structure))
      .forEach((s) => {
        const el = document.createElement('button')
        el.type = 'button'
        el.className = 'lm-pin'
        el.style.cssText = `
          width:14px;height:14px;border-radius:50%;
          background:${STRUCTURE_COLOR[s.structure] ?? '#C6F04A'};
          box-shadow:0 0 0 3px rgba(7,18,15,0.55), 0 0 10px ${STRUCTURE_COLOR[s.structure] ?? '#C6F04A'};
          border:1px solid #E8DCC4; cursor:pointer; transform:scale(1); transition:transform 160ms ease;
        `
        el.title = s.name
        el.onmouseenter = () => {
          el.style.transform = 'scale(1.45)'
        }
        el.onmouseleave = () => {
          el.style.transform = 'scale(1)'
        }
        el.onclick = (ev) => {
          ev.stopPropagation()
          setDraft(null)
          setSelected(s)
        }
        const mk = new maplibregl.Marker({ element: el }).setLngLat([s.lon, s.lat]).addTo(map)
        markers.current.push(mk)
      })
  }, [spots, active, styleRev])

  async function createSpot(body: CreateSpotBody) {
    setBusy(true)
    try {
      const created = await apiPost<SpotDTO>('/api/v1/spots', body)
      setSpots((xs) => [created, ...xs])
      setDraft(null)
      setSelected(created)
      toast('标点已钉下', 'ok')
      mapRef.current?.flyTo({ center: [created.lon, created.lat], zoom: 12 })
    } catch (e) {
      toast(e instanceof ApiError ? e.message : '创建失败', 'err')
    } finally {
      setBusy(false)
    }
  }

  const visible = spots.filter((s) => active.has(s.structure))

  return (
    <div className="relative h-[calc(100vh-52px)] w-full md:h-screen">
      <div ref={wrap} className="absolute inset-0" />
      {loading && (
        <div className="absolute inset-0 z-20 bg-ink/50">
          <Spinner label="展开海图…" />
        </div>
      )}
      {error && !loading && (
        <div className="absolute inset-x-0 top-16 z-20">
          <ErrorState message={error} onRetry={load} />
        </div>
      )}

      <AtlasHUD
        count={visible.length}
        pressure={labelOf(PRESSURE_LABEL, hydroMini?.pressure_trend ?? '')}
        tide={labelOf(TIDE_LABEL, hydroMini?.tide_window ?? '')}
        score={hydroMini?.bite_score}
        active={active}
        onToggle={(k) => {
          setActive((prev) => {
            const next = new Set(prev)
            if (next.has(k)) next.delete(k)
            else next.add(k)
            return next
          })
        }}
        onAll={() => setActive(new Set(STRUCTURES))}
      />

      <div className="absolute left-3 top-28 z-20 flex gap-1">
        {(['dark', 'sat', 'osm'] as StyleKey[]).map((k) => (
          <Button key={k} variant={style === k ? 'sonar' : 'ghost'} className="px-2 py-1 text-xs" onClick={() => setStyle(k)}>
            {k === 'dark' ? '暗色' : k === 'sat' ? '卫星' : '海图'}
          </Button>
        ))}
      </div>

      <SpotDrawer
        spot={selected}
        onClose={() => setSelected(null)}
        onRecord={(id) => navigate(`/catches/new?spot=${id}`)}
      />

      <Modal
        open={!!draft}
        title="钉一枚新标点"
        onClose={() => setDraft(null)}
        hideFooter
      >
        {draft && (
          <SpotForm
            lat={draft.lat}
            lon={draft.lon}
            clubs={clubs}
            busy={busy}
            onCancel={() => setDraft(null)}
            onSubmit={createSpot}
          />
        )}
      </Modal>
    </div>
  )
}
