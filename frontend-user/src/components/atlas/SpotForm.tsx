import { useState } from 'react'
import type { ClubDTO, CreateSpotBody, DepthDTO } from '../../api/types'
import {
  STRUCTURE_LABEL,
  STRUCTURES,
  VISIBILITIES,
  VISIBILITY_LABEL,
  WATER_LABEL,
  WATER_TYPES,
} from '../../lib/enums'
import { Button } from '../ui/Button'
import { Select, TextArea, TextInput } from '../ui/Field'

interface Props {
  lat: number
  lon: number
  clubs: ClubDTO[]
  busy: boolean
  onCancel: () => void
  onSubmit: (body: CreateSpotBody) => void
}

export function SpotForm({ lat, lon, clubs, busy, onCancel, onSubmit }: Props) {
  const [name, setName] = useState('')
  const [waterType, setWaterType] = useState('RIVER')
  const [structure, setStructure] = useState('EDDY')
  const [visibility, setVisibility] = useState('PRIVATE')
  const [clubId, setClubId] = useState(clubs[0]?.id ?? '')
  const [tidal, setTidal] = useState(true)
  const [bearing, setBearing] = useState('90')
  const [note, setNote] = useState('')
  const [latV, setLatV] = useState(String(lat.toFixed(6)))
  const [lonV, setLonV] = useState(String(lon.toFixed(6)))
  const [d1, setD1] = useState('0')
  const [h1, setH1] = useState('3.5')
  const [errors, setErrors] = useState<Record<string, string>>({})

  function validate() {
    const e: Record<string, string> = {}
    if (!name.trim()) e.name = '名称必填'
    const la = Number(latV)
    const lo = Number(lonV)
    if (Number.isNaN(la) || la < -90 || la > 90) e.lat = '纬度须在 -90 到 90'
    if (Number.isNaN(lo) || lo < -180 || lo > 180) e.lon = '经度须在 -180 到 180'
    if (visibility === 'CLUB' && !clubId) e.club_id = '俱乐部可见性需要选择俱乐部'
    setErrors(e)
    return Object.keys(e).length === 0
  }

  function submit() {
    if (!validate()) return
    const depths: DepthDTO[] = []
    const off = Number(d1)
    const dep = Number(h1)
    if (!Number.isNaN(off) && !Number.isNaN(dep)) depths.push({ offset_m: off, depth_m: dep })
    onSubmit({
      name: name.trim(),
      water_type: waterType,
      structure,
      visibility,
      lat: Number(latV),
      lon: Number(lonV),
      shore_bearing: Number(bearing) || 0,
      tidal,
      note: note.trim(),
      depths,
      club_id: visibility === 'CLUB' ? clubId : undefined,
    })
  }

  return (
    <div className="space-y-3">
      <TextInput label="标点名称" value={name} onChange={(e) => setName(e.target.value)} error={errors.name} />
      <div className="grid grid-cols-1 gap-3 xs:grid-cols-2">
        <Select label="水域" value={waterType} onChange={(e) => setWaterType(e.target.value)}>
          {WATER_TYPES.map((w) => (
            <option key={w} value={w}>
              {WATER_LABEL[w]}
            </option>
          ))}
        </Select>
        <Select label="结构" value={structure} onChange={(e) => setStructure(e.target.value)}>
          {STRUCTURES.map((s) => (
            <option key={s} value={s}>
              {STRUCTURE_LABEL[s]}
            </option>
          ))}
        </Select>
        <Select label="可见性" value={visibility} onChange={(e) => setVisibility(e.target.value)}>
          {VISIBILITIES.map((v) => (
            <option key={v} value={v}>
              {VISIBILITY_LABEL[v]}
            </option>
          ))}
        </Select>
        {visibility === 'CLUB' && (
          <Select label="俱乐部" value={clubId} onChange={(e) => setClubId(e.target.value)} error={errors.club_id}>
            <option value="">选择俱乐部</option>
            {clubs.map((c) => (
              <option key={c.id} value={c.id}>
                {c.name}
              </option>
            ))}
          </Select>
        )}
        <TextInput label="纬度" value={latV} onChange={(e) => setLatV(e.target.value)} error={errors.lat} />
        <TextInput label="经度" value={lonV} onChange={(e) => setLonV(e.target.value)} error={errors.lon} />
        <TextInput label="岸线朝向°" value={bearing} onChange={(e) => setBearing(e.target.value)} />
        <Select label="感潮" value={tidal ? '1' : '0'} onChange={(e) => setTidal(e.target.value === '1')}>
          <option value="1">是</option>
          <option value="0">否</option>
        </Select>
        <TextInput label="剖面偏移 m" value={d1} onChange={(e) => setD1(e.target.value)} />
        <TextInput label="剖面水深 m" value={h1} onChange={(e) => setH1(e.target.value)} />
      </div>
      <TextArea label="备注" value={note} onChange={(e) => setNote(e.target.value)} />
      <div className="flex justify-end gap-2">
        <Button variant="ghost" onClick={onCancel} disabled={busy}>
          取消
        </Button>
        <Button variant="copper" onClick={submit} disabled={busy}>
          {busy ? '落钉中…' : '钉下标点'}
        </Button>
      </div>
    </div>
  )
}
