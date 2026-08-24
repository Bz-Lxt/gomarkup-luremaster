export const STRUCTURES = [
  'SNAG',
  'ROCK',
  'PIER',
  'DROPOFF',
  'BACKWATER',
  'INLET',
  'WEED',
  'CAGE',
  'WRECK',
  'EDDY',
] as const

export const STRUCTURE_LABEL: Record<string, string> = {
  SNAG: '枯树',
  ROCK: '乱石',
  PIER: '桥墩',
  DROPOFF: '深浅交界',
  BACKWATER: '回湾',
  INLET: '入水口',
  WEED: '草洞',
  CAGE: '网箱',
  WRECK: '沉船',
  EDDY: '洄流带',
}

export const STRUCTURE_COLOR: Record<string, string> = {
  SNAG: '#8B6914',
  ROCK: '#7A8B99',
  PIER: '#C9A227',
  DROPOFF: '#5EC8D8',
  BACKWATER: '#6BAF8A',
  INLET: '#4A90D9',
  WEED: '#3F7D4E',
  CAGE: '#B5651D',
  WRECK: '#5C4A3A',
  EDDY: '#D4783A',
}

export const SPECIES = [
  'YELLOWCHECK',
  'MANDARIN',
  'BASS',
  'SNAKEHEAD',
  'PIKEKILLIFISH',
  'CHUB',
  'SEA_BASS',
  'CATFISH',
  'OTHER',
] as const

export const SPECIES_LABEL: Record<string, string> = {
  YELLOWCHECK: '翘嘴',
  MANDARIN: '鳜鱼',
  BASS: '鲈鱼',
  SNAKEHEAD: '黑鱼',
  PIKEKILLIFISH: '马口',
  CHUB: '军鱼',
  SEA_BASS: '海鲈',
  CATFISH: '鲶鱼',
  OTHER: '其他',
}

export const LURES = ['MINNOW', 'VIB', 'SOFT', 'PENCIL', 'POPPER', 'SPOON', 'SPINNER', 'JIG'] as const

export const LURE_LABEL: Record<string, string> = {
  MINNOW: '米诺',
  VIB: 'VIB',
  SOFT: '软虫',
  PENCIL: '铅笔',
  POPPER: '波扒',
  SPOON: '亮片',
  SPINNER: '复合亮片',
  JIG: '铁板',
}

export const VISIBILITIES = ['PRIVATE', 'CLUB', 'FRIENDS', 'PUBLIC'] as const

export const VISIBILITY_LABEL: Record<string, string> = {
  PRIVATE: '仅自己',
  CLUB: '俱乐部',
  FRIENDS: '互关',
  PUBLIC: '公开',
}

export const WATER_TYPES = ['RESERVOIR', 'RIVER', 'LAKE', 'SEA', 'POND'] as const

export const WATER_LABEL: Record<string, string> = {
  RESERVOIR: '水库',
  RIVER: '江河',
  LAKE: '湖泊',
  SEA: '海钓',
  POND: '野塘',
}

export const PRESSURE_TRENDS = ['CRASH_DOWN', 'FALL', 'STABLE', 'RISE', 'CRASH_UP'] as const

export const PRESSURE_LABEL: Record<string, string> = {
  CRASH_DOWN: '暴降',
  FALL: '缓降',
  STABLE: '平稳',
  RISE: '缓升',
  CRASH_UP: '暴升',
}

export const TIDE_WINDOWS = [
  'SLACK_LOW',
  'EARLY_FLOOD',
  'THIRD',
  'HALF',
  'SEVENTH',
  'SLACK_HIGH',
  'EARLY_EBB',
  'RAPID_EBB',
] as const

export const TIDE_LABEL: Record<string, string> = {
  SLACK_LOW: '枯潮',
  EARLY_FLOOD: '初涨',
  THIRD: '三分潮',
  HALF: '半潮',
  SEVENTH: '七分潮',
  SLACK_HIGH: '满潮',
  EARLY_EBB: '初落',
  RAPID_EBB: '急落',
}

export const MOON_LABEL: Record<string, string> = {
  NEW: '新月',
  WAXING_CRESCENT: '娥眉',
  FIRST_QUARTER: '上弦',
  WAXING_GIBBOUS: '盈凸',
  FULL: '满月',
  WANING_GIBBOUS: '亏凸',
  LAST_QUARTER: '下弦',
  WANING_CRESCENT: '残月',
}

export const RETRIEVES = ['TWITCH', 'STEADY', 'SLOW', 'WALK', 'POP', 'HOP'] as const

export const RETRIEVE_LABEL: Record<string, string> = {
  TWITCH: '抽停',
  STEADY: '匀收',
  SLOW: '慢摇',
  WALK: '狗步',
  POP: '波扒',
  HOP: '跳底',
}

export const LAYERS = ['SHALLOW', 'MID', 'DEEP', 'BOTTOM'] as const

export const LAYER_LABEL: Record<string, string> = {
  SHALLOW: '浅层',
  MID: '中层',
  DEEP: '深层',
  BOTTOM: '底层',
}

export const ACTIVITY_KIND_LABEL: Record<string, string> = {
  CLUB_CHARTER: '俱乐部包船',
  WILD_MEETUP: '同城野钓',
  HOT_SLOT: '热门放位',
}

export const SLOT_STATUS_LABEL: Record<string, string> = {
  OPEN: '可抢',
  LOCKED: '锁定中',
  CONFIRMED: '已确认',
  CHECKED_IN: '已打卡',
}

export function labelOf(map: Record<string, string>, key: string, fallback = key) {
  return map[key] ?? fallback
}
