package hydro

import (
	"math"
	"strconv"
	"time"

	"luremaster/internal/timeutil"
)

type dagNode struct {
	ID     string
	Label  string
	Base   float64
	Reason string
}

type dagEdge struct {
	From, To string
	When     func(ctx scoreCtx) bool
	Bonus    float64
	Reason   string
}

type scoreCtx struct {
	Trend, Tide, Aspect, Moon, Period string
	Beaufort                          int
	WaterTemp                         float64
}

func ScoreBite(snap Snapshot, waterTemp *float64, at time.Time) (float64, bool, []Contribution) {
	wt := 20.0
	if waterTemp != nil {
		wt = *waterTemp
	} else {
		wt = snap.AirTempC - 1.5
	}
	ctx := scoreCtx{
		Trend:     snap.PressureTrend,
		Tide:      snap.TideWindow,
		Aspect:    snap.ShoreAspect,
		Moon:      snap.MoonPhase,
		Period:    dayPeriod(at),
		Beaufort:  snap.Beaufort,
		WaterTemp: wt,
	}
	nodes := []dagNode{
		{ID: "pressure", Label: "气压趋势", Base: pressureBase(ctx.Trend), Reason: pressureReason(ctx.Trend)},
		{ID: "tide", Label: "潮汐窗口", Base: tideBase(ctx.Tide), Reason: tideReason(ctx.Tide)},
		{ID: "wind", Label: "风向风力", Base: windBase(ctx.Beaufort, ctx.Aspect), Reason: windReason(ctx.Beaufort, ctx.Aspect)},
		{ID: "moon", Label: "月相", Base: moonBase(ctx.Moon), Reason: moonReason(ctx.Moon)},
		{ID: "temp", Label: "水温", Base: tempBase(ctx.WaterTemp), Reason: tempReason(ctx.WaterTemp)},
		{ID: "period", Label: "时段", Base: periodBase(ctx.Period), Reason: periodReason(ctx.Period)},
	}
	edges := []dagEdge{
		{From: "pressure", To: "tide", Bonus: 12, Reason: "暴降叠三分潮，开口加速", When: func(c scoreCtx) bool {
			return c.Trend == TrendCrashDown && (c.Tide == TideThird || c.Tide == TideEarlyFlood)
		}},
		{From: "pressure", To: "period", Bonus: 8, Reason: "低压配合晨昏", When: func(c scoreCtx) bool {
			return c.Trend == TrendCrashDown && (c.Period == "DAWN" || c.Period == "DUSK")
		}},
		{From: "moon", To: "period", Bonus: 10, Reason: "满月夜光增强夜口", When: func(c scoreCtx) bool {
			return c.Moon == MoonFull && c.Period == "NIGHT"
		}},
		{From: "tide", To: "wind", Bonus: 6, Reason: "涨潮迎风岸饵鱼堆积", When: func(c scoreCtx) bool {
			return (c.Tide == TideThird || c.Tide == TideSeventh) && c.Aspect == AspectOnshore && c.Beaufort >= 2 && c.Beaufort <= 5
		}},
		{From: "temp", To: "period", Bonus: 5, Reason: "适温叠黄金时段", When: func(c scoreCtx) bool {
			return c.WaterTemp >= 16 && c.WaterTemp <= 26 && (c.Period == "DAWN" || c.Period == "DUSK")
		}},
		{From: "pressure", To: "temp", Bonus: -8, Reason: "暴升叠加过热，鱼沉底少开口", When: func(c scoreCtx) bool {
			return c.Trend == TrendCrashUp && c.WaterTemp >= 28
		}},
	}

	bonus := map[string]float64{}
	reasons := map[string][]string{}
	for _, e := range edges {
		if e.When(ctx) {
			bonus[e.From] += e.Bonus
			reasons[e.From] = append(reasons[e.From], e.Reason)
		}
	}

	weights := map[string]float64{
		"pressure": 0.24, "tide": 0.22, "wind": 0.14,
		"moon": 0.12, "temp": 0.14, "period": 0.14,
	}
	var contrib []Contribution
	var sum float64
	for _, n := range nodes {
		sc := clamp(n.Base+bonus[n.ID], 0, 100)
		reason := n.Reason
		if extra := reasons[n.ID]; len(extra) > 0 {
			reason = reason + "；" + extra[0]
		}
		contrib = append(contrib, Contribution{
			Node: n.ID, Label: n.Label, Base: n.Base, Bonus: bonus[n.ID], Score: sc, Reason: reason,
		})
		sum += sc * weights[n.ID]
	}
	score := math.Round(sum*10) / 10
	return score, score >= 75, contrib
}

func dayPeriod(at time.Time) string {
	h := at.In(timeutil.Beijing).Hour()
	switch {
	case h >= 4 && h < 7:
		return "DAWN"
	case h >= 17 && h < 20:
		return "DUSK"
	case h >= 20 || h < 4:
		return "NIGHT"
	default:
		return "DAY"
	}
}

func pressureBase(t string) float64 {
	switch t {
	case TrendCrashDown:
		return 90
	case TrendFall:
		return 72
	case TrendStable:
		return 52
	case TrendRise:
		return 40
	case TrendCrashUp:
		return 24
	default:
		return 50
	}
}

func pressureReason(t string) string {
	switch t {
	case TrendCrashDown:
		return "气压暴降，鱼抢食补能"
	case TrendFall:
		return "气压缓降，开口偏积极"
	case TrendStable:
		return "气压平稳，常规巡游"
	case TrendRise:
		return "气压缓升，鱼趋保守"
	default:
		return "气压暴升，鱼沉底少动"
	}
}

func tideBase(t string) float64 {
	switch t {
	case TideThird:
		return 88
	case TideEarlyFlood:
		return 80
	case TideSeventh:
		return 76
	case TideHalf:
		return 70
	case TideEarlyEbb:
		return 68
	case TideRapidEbb:
		return 58
	case TideSlackHigh:
		return 50
	case TideSlackLow:
		return 42
	default:
		return 55
	}
}

func tideReason(t string) string {
	switch t {
	case TideThird:
		return "三分潮初涨，饵鱼随流"
	case TideEarlyFlood:
		return "初涨洗岸，结构点开始进鱼"
	case TideSeventh:
		return "七分潮，水位推高结构"
	case TideSlackLow:
		return "枯潮，鱼多聚深槽"
	case TideSlackHigh:
		return "满潮平流，搜索成本升高"
	default:
		return "潮汐过渡窗口"
	}
}

func windBase(b int, aspect string) float64 {
	base := 48.0
	switch {
	case b >= 2 && b <= 4:
		base = 74
	case b == 5:
		base = 62
	case b <= 1:
		base = 40
	default:
		base = 28
	}
	if aspect == AspectOnshore && b >= 2 && b <= 5 {
		base += 6
	}
	if aspect == AspectOffshore {
		base -= 4
	}
	return clamp(base, 0, 100)
}

func windReason(b int, aspect string) string {
	return "风力" + itoa(b) + "级 · " + aspectLabel(aspect)
}

func moonBase(m string) float64 {
	switch m {
	case MoonNew, MoonFull:
		return 78
	case MoonWaxGib, MoonWanGib:
		return 64
	case MoonFirst, MoonLast:
		return 56
	default:
		return 50
	}
}

func moonReason(m string) string {
	if m == MoonFull || m == MoonNew {
		return "朔望大潮窗口，夜口增强"
	}
	return "月相过渡，影响有限"
}

func tempBase(c float64) float64 {
	switch {
	case c >= 18 && c <= 26:
		return 80
	case c >= 14 && c < 18:
		return 64
	case c > 26 && c <= 30:
		return 58
	default:
		return 38
	}
}

func tempReason(c float64) string {
	return "推定水温约 " + ftoa(c) + "℃"
}

func periodBase(p string) float64 {
	switch p {
	case "DAWN":
		return 86
	case "DUSK":
		return 84
	case "NIGHT":
		return 58
	default:
		return 42
	}
}

func periodReason(p string) string {
	switch p {
	case "DAWN":
		return "晨光低照度，猎食窗口"
	case "DUSK":
		return "黄昏低照度，猎食窗口"
	case "NIGHT":
		return "夜间，视种而定"
	default:
		return "日间，鱼多停在结构阴影"
	}
}

func aspectLabel(a string) string {
	switch a {
	case AspectOnshore:
		return "迎风岸"
	case AspectOffshore:
		return "背风岸"
	default:
		return "侧风岸"
	}
}

func clamp(v, lo, hi float64) float64 {
	return math.Min(hi, math.Max(lo, v))
}

func itoa(i int) string { return strconv.Itoa(i) }

func ftoa(f float64) string {
	return strconv.FormatFloat(math.Round(f*10)/10, 'f', 1, 64)
}
