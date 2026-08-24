export interface APIErrorBody {
  code: string
  message: string
}

export interface APIOk<T> {
  ok: true
  data: T
}

export interface APIFail {
  ok: false
  error: APIErrorBody
}

export type APIEnvelope<T> = APIOk<T> | APIFail

export interface UserDTO {
  id: string
  email: string
  username: string
  nickname: string
  avatar_url: string
  home_water: string
  lure_pref: string
  credit_score: number
  created_at: string
}

export interface TokenUserData {
  access_token: string
  refresh_token: string
  user: UserDTO
}

export interface DepthDTO {
  offset_m: number
  depth_m: number
}

export interface SpotDTO {
  id: string
  owner_id: string
  club_id: string | null
  name: string
  water_type: string
  structure: string
  visibility: string
  lat: number
  lon: number
  shore_bearing: number
  tidal: boolean
  note: string
  fuzzed: boolean
  depths: DepthDTO[]
  created_at: string
}

export interface HydroContribution {
  node: string
  label: string
  base: number
  bonus: number
  score: number
  reason: string
}

export interface HydroHourly {
  at: string
  pressure_hpa?: number
  temp_c?: number
  wind_dir_deg?: number
  wind_ms?: number
}

export interface HydroTide {
  at: string
  height_m?: number
}

export interface HydroSeries {
  hourly?: HydroHourly[]
  tides?: HydroTide[]
  tide?: HydroTide[]
}

export interface HydroDTO {
  status: string
  bound_at?: string
  bind_error_sec?: number
  pressure_hpa?: number
  pressure_delta_3h?: number
  pressure_trend?: string
  air_temp_c?: number
  wind_dir_deg?: number
  wind_dir_label?: string
  wind_speed_ms?: number
  beaufort?: number
  shore_aspect?: string
  tide_height_m?: number
  tide_phase_pct?: number
  tide_window?: string
  moon_phase?: string
  moon_illum_pct?: number
  bite_score?: number
  frenzy?: boolean
  contributions?: HydroContribution[]
  series?: HydroSeries
}

export interface CatchDTO {
  id: string
  user_id: string
  spot_id: string
  caught_at: string
  timezone: string
  local_time: string
  species: string
  length_cm: number
  weight_kg: number | null
  lure_type: string
  lure_weight_g: number | null
  lure_color: string
  retrieve: string
  layer: string
  water_depth_m: number | null
  water_color: string
  turbidity: string
  water_temp_c: number | null
  current: string
  released: boolean
  note: string
  photo_key: string
  photo_url: string
  hydro_status: string
  hydro: HydroDTO
  created_at: string
}

export interface CreateCatchBody {
  spot_id: string
  local_time: string
  timezone: string
  species: string
  length_cm: number
  weight_kg?: number | null
  lure_type: string
  lure_color: string
  retrieve: string
  layer: string
  water_depth_m?: number | null
  water_color: string
  turbidity: string
  water_temp_c?: number | null
  current: string
  released: boolean
  note: string
}

export interface CreateSpotBody {
  name: string
  water_type: string
  structure: string
  visibility: string
  lat: number
  lon: number
  shore_bearing: number
  tidal: boolean
  note: string
  depths: DepthDTO[]
  club_id?: string
}

export interface ClubDTO {
  id: string
  name: string
  owner_id: string
  description: string
  members: number
  created_at: string
}

export interface SlotDTO {
  id: string
  activity_id: string
  label: string
  status: string
  holder_id: string
  lock_expires_at: string
  version: number
}

export interface ActivityDTO {
  id: string
  host_id: string
  club_id: string | null
  spot_id: string
  title: string
  kind: string
  status: string
  starts_at: string
  ends_at: string
  meet_lat: number
  meet_lon: number
  meet_radius_m: number
  fee_amount: number
  fee_note: string
  slots: SlotDTO[]
  created_at: string
}

export interface CheckinDTO {
  activity_id: string
  user_id: string
  lat: number
  lon: number
  distance_m: number
  created_at: string
}

export interface AdviceDTO {
  lure_type: string
  label: string
  color: string
  layer: string
  retrieve: string
  score: number
  reason: string
}

export interface TopLureDTO {
  lure_type: string
  count: number
}

export interface TopSpotDTO {
  spot_id: string
  name: string
  count: number
}

export interface StatsDTO {
  total_catches: number
  released_count: number
  release_rate: number
  max_length_cm: number
  max_species: string
  streak_days: number
  top_lures: TopLureDTO[]
  top_spots: TopSpotDTO[]
}
