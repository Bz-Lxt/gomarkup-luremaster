import { useEffect, useRef } from 'react'
import * as echarts from 'echarts'
import type { HydroDTO } from '../../api/types'
import { EmptyState } from '../ui/States'

interface Props {
  hydro?: HydroDTO | null
  catchAt?: string
}

export function HydroTwinChart({ hydro, catchAt }: Props) {
  const ref = useRef<HTMLDivElement>(null)
  const hourly = hydro?.series?.hourly ?? []
  const tides = hydro?.series?.tides ?? hydro?.series?.tide ?? []
  const hasSeries = hourly.length > 0 || tides.length > 0
  const seriesKey = JSON.stringify(hydro?.series ?? null)

  useEffect(() => {
    if (!ref.current || !hasSeries) return
    const chart = echarts.init(ref.current, undefined, { renderer: 'canvas' })
    const hours = hourly.map((p) => p.at)
    const tideTimes = tides.map((p) => p.at)
    const axis = hours.length >= tideTimes.length ? hours : tideTimes
    const mark = catchAt
      ? {
          symbol: 'none' as const,
          data: [{ xAxis: catchAt, label: { formatter: '中鱼', color: '#C6F04A' }, lineStyle: { color: '#C6F04A' } }],
        }
      : undefined

    chart.setOption({
      backgroundColor: 'transparent',
      animationDuration: 600,
      tooltip: { trigger: 'axis' },
      legend: { data: ['气压 hPa', '潮高 m'], textStyle: { color: '#E8DCC4' } },
      brush: { xAxisIndex: 0, toolbox: ['lineX', 'clear'] },
      toolbox: { feature: { brush: { type: ['lineX', 'clear'] } }, iconStyle: { borderColor: '#8A9A8E' } },
      grid: { left: 48, right: 48, top: 48, bottom: 40 },
      xAxis: {
        type: 'category',
        data: axis,
        axisLabel: { color: '#8A9A8E', formatter: (v: string) => (v || '').slice(11, 16) },
        axisLine: { lineStyle: { color: '#8A9A8E' } },
      },
      yAxis: [
        {
          type: 'value',
          name: 'hPa',
          nameTextStyle: { color: '#F2C14E' },
          axisLabel: { color: '#F2C14E' },
          splitLine: { lineStyle: { color: 'rgba(232,220,196,0.06)' } },
        },
        {
          type: 'value',
          name: 'm',
          nameTextStyle: { color: '#5EC8D8' },
          axisLabel: { color: '#5EC8D8' },
          splitLine: { show: false },
        },
      ],
      series: [
        {
          name: '气压 hPa',
          type: 'line',
          smooth: true,
          showSymbol: false,
          itemStyle: { color: '#F2C14E' },
          data: hourly.map((p) => [p.at, p.pressure_hpa ?? null]),
          markLine: mark,
        },
        {
          name: '潮高 m',
          type: 'line',
          yAxisIndex: 1,
          smooth: true,
          showSymbol: false,
          itemStyle: { color: '#5EC8D8' },
          areaStyle: { color: 'rgba(94,200,216,0.12)' },
          data: tides.map((p) => [p.at, p.height_m ?? null]),
        },
      ],
    })

    const ro = new ResizeObserver(() => chart.resize())
    ro.observe(ref.current)
    return () => {
      ro.disconnect()
      chart.dispose()
    }
  }, [hasSeries, seriesKey, catchAt])

  if (!hasSeries) {
    return <EmptyState title="水文曲线未绑定" hint="等待气压与潮汐序列写入快照" />
  }

  return <div ref={ref} className="h-[320px] w-full" />
}
