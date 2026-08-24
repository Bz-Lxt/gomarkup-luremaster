import { useEffect, useState } from 'react'
import { useNavigate, useSearchParams } from 'react-router-dom'
import { apiGet, apiPost, api, ApiError } from '../api/client'
import type { CatchDTO, CreateCatchBody, SpotDTO } from '../api/types'
import { Button } from '../components/ui/Button'
import { Select, TextArea, TextInput } from '../components/ui/Field'
import { ErrorState, Spinner } from '../components/ui/States'
import { useToast } from '../context/ToastContext'
import {
  LAYERS,
  LAYER_LABEL,
  LURES,
  LURE_LABEL,
  RETRIEVES,
  RETRIEVE_LABEL,
  SPECIES,
  SPECIES_LABEL,
} from '../lib/enums'
import { formatLocalMinute } from '../lib/time'

export function NewCatchPage() {
  const [params] = useSearchParams()
  const navigate = useNavigate()
  const { toast } = useToast()
  const [spots, setSpots] = useState<SpotDTO[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')
  const [busy, setBusy] = useState(false)
  const [dragOver, setDragOver] = useState(false)
  const [file, setFile] = useState<File | null>(null)
  const [preview, setPreview] = useState('')
  const [errors, setErrors] = useState<Record<string, string>>({})

  const [spotId, setSpotId] = useState(params.get('spot') ?? '')
  const [localTime, setLocalTime] = useState(formatLocalMinute())
  const [species, setSpecies] = useState('YELLOWCHECK')
  const [length, setLength] = useState('45')
  const [weight, setWeight] = useState('')
  const [lure, setLure] = useState('MINNOW')
  const [color, setColor] = useState('银白')
  const [retrieve, setRetrieve] = useState('TWITCH')
  const [layer, setLayer] = useState('MID')
  const [depth, setDepth] = useState('2.4')
  const [waterColor, setWaterColor] = useState('清')
  const [turbidity, setTurbidity] = useState('中')
  const [temp, setTemp] = useState('22')
  const [current, setCurrent] = useState('缓')
  const [released, setReleased] = useState(true)
  const [note, setNote] = useState('')

  useEffect(() => {
    apiGet<SpotDTO[]>('/api/v1/spots')
      .then((xs) => {
        setSpots(xs)
        if (!spotId && xs[0]) setSpotId(xs[0].id)
      })
      .catch((e) => setError(e instanceof ApiError ? e.message : '标点加载失败'))
      .finally(() => setLoading(false))
  }, [])

  function takeFile(f?: File | null) {
    if (!f) return
    setFile(f)
    setPreview(URL.createObjectURL(f))
  }

  function validate() {
    const e: Record<string, string> = {}
    if (!spotId) e.spot_id = '请选择标点'
    if (!localTime.trim()) e.local_time = '中鱼时刻必填'
    const len = Number(length)
    if (!len || len <= 0) e.length_cm = '体长必须大于 0'
    setErrors(e)
    return Object.keys(e).length === 0
  }

  async function submit() {
    if (!validate()) return
    setBusy(true)
    try {
      const body: CreateCatchBody = {
        spot_id: spotId,
        local_time: localTime.trim(),
        timezone: 'Asia/Shanghai',
        species,
        length_cm: Number(length),
        weight_kg: weight ? Number(weight) : null,
        lure_type: lure,
        lure_color: color,
        retrieve,
        layer,
        water_depth_m: depth ? Number(depth) : null,
        water_color: waterColor,
        turbidity,
        water_temp_c: temp ? Number(temp) : null,
        current,
        released,
        note,
      }
      const created = await apiPost<CatchDTO>('/api/v1/catches', body)
      if (file) {
        const fd = new FormData()
        fd.append('file', file)
        await api<CatchDTO>(`/api/v1/catches/${created.id}/photo`, { method: 'POST', body: fd })
      }
      toast('战报已入袋，水文绑定中', 'ok')
      navigate('/catches')
    } catch (e) {
      toast(e instanceof ApiError ? e.message : '录入失败', 'err')
    } finally {
      setBusy(false)
    }
  }

  if (loading) return <Spinner />
  if (error) return <ErrorState message={error} onRetry={() => window.location.reload()} />

  return (
    <div className="w-full p-4 md:p-6">
      <p className="font-mono text-[10px] tracking-[0.22em] text-mute">NEW BITE</p>
      <h1 className="mb-6 font-display text-4xl italic">录入中鱼</h1>
      <div className="grid w-full grid-cols-1 gap-4 xs:grid-cols-2">
        <Select label="标点" value={spotId} onChange={(e) => setSpotId(e.target.value)} error={errors.spot_id}>
          <option value="">选择标点</option>
          {spots.map((s) => (
            <option key={s.id} value={s.id}>
              {s.name}
            </option>
          ))}
        </Select>
        <TextInput
          label="中鱼时刻"
          value={localTime}
          onChange={(e) => setLocalTime(e.target.value)}
          error={errors.local_time}
        />
        <Select label="鱼种" value={species} onChange={(e) => setSpecies(e.target.value)}>
          {SPECIES.map((s) => (
            <option key={s} value={s}>
              {SPECIES_LABEL[s]}
            </option>
          ))}
        </Select>
        <TextInput label="体长 cm" value={length} onChange={(e) => setLength(e.target.value)} error={errors.length_cm} />
        <TextInput label="体重 kg" value={weight} onChange={(e) => setWeight(e.target.value)} />
        <Select label="拟饵" value={lure} onChange={(e) => setLure(e.target.value)}>
          {LURES.map((s) => (
            <option key={s} value={s}>
              {LURE_LABEL[s]}
            </option>
          ))}
        </Select>
        <TextInput label="饵色" value={color} onChange={(e) => setColor(e.target.value)} />
        <Select label="手法" value={retrieve} onChange={(e) => setRetrieve(e.target.value)}>
          {RETRIEVES.map((s) => (
            <option key={s} value={s}>
              {RETRIEVE_LABEL[s]}
            </option>
          ))}
        </Select>
        <Select label="泳层" value={layer} onChange={(e) => setLayer(e.target.value)}>
          {LAYERS.map((s) => (
            <option key={s} value={s}>
              {LAYER_LABEL[s]}
            </option>
          ))}
        </Select>
        <TextInput label="水深 m" value={depth} onChange={(e) => setDepth(e.target.value)} />
        <TextInput label="水色" value={waterColor} onChange={(e) => setWaterColor(e.target.value)} />
        <TextInput label="浊度" value={turbidity} onChange={(e) => setTurbidity(e.target.value)} />
        <TextInput label="水温 °C" value={temp} onChange={(e) => setTemp(e.target.value)} />
        <TextInput label="流速" value={current} onChange={(e) => setCurrent(e.target.value)} />
        <Select label="放流" value={released ? '1' : '0'} onChange={(e) => setReleased(e.target.value === '1')}>
          <option value="1">是</option>
          <option value="0">否</option>
        </Select>
      </div>
      <div className="mt-4">
        <TextArea label="备注" value={note} onChange={(e) => setNote(e.target.value)} />
      </div>

      <div
        className={`mt-4 rounded-sm border border-dashed px-4 py-8 text-center ${
          dragOver ? 'border-sonar bg-sonar/10' : 'border-foam/20'
        }`}
        onDragOver={(e) => {
          e.preventDefault()
          setDragOver(true)
        }}
        onDragLeave={() => setDragOver(false)}
        onDrop={(e) => {
          e.preventDefault()
          setDragOver(false)
          takeFile(e.dataTransfer.files[0])
        }}
      >
        <p className="text-sm text-mute">拖拽照片到这里，先创建战报再上传</p>
        <label className="mt-3 inline-block cursor-pointer rounded-sm border border-foam/20 px-3 py-1 text-xs hover:border-sonar">
          选择文件
          <input
            type="file"
            accept="image/png,image/jpeg,image/webp,image/gif"
            className="hidden"
            onChange={(e) => takeFile(e.target.files?.[0])}
          />
        </label>
        {preview ? <img src={preview} alt="预览" className="mx-auto mt-4 max-h-40 rounded-sm" /> : null}
      </div>

      <div className="mt-6 flex justify-end gap-2">
        <Button variant="ghost" onClick={() => navigate('/catches')} disabled={busy}>
          返回战报
        </Button>
        <Button variant="copper" onClick={submit} disabled={busy}>
          {busy ? '入袋中…' : '写入战报'}
        </Button>
      </div>
    </div>
  )
}
