package lure

import (
	"math"
	"sort"
	"strings"
)

type HydroHint struct {
	PressureTrend string
	TideWindow    string
	WaterTempC    float64
	WindBeaufort  int
	Frenzy        bool
}

type HistoryHit struct {
	LureType string
	Color    string
	Layer    string
	Retrieve string
	Caught   bool
}

type Advice struct {
	LureType string  `json:"lure_type"`
	Label    string  `json:"label"`
	Color    string  `json:"color"`
	Layer    string  `json:"layer"`
	Retrieve string  `json:"retrieve"`
	Score    float64 `json:"score"`
	Reason   string  `json:"reason"`
}

var labels = map[string]string{
	"MINNOW": "米诺", "VIB": "VIB", "SOFT": "软虫", "PENCIL": "铅笔",
	"POPPER": "波扒", "SPOON": "亮片", "SPINNER": "复合亮片", "JIG": "铁板",
}

func Recommend(species string, h HydroHint, history []HistoryHit) []Advice {
	priors := rulePriors(species, h)
	counts := map[string]float64{}
	success := map[string]float64{}
	for _, hit := range history {
		counts[hit.LureType]++
		if hit.Caught {
			success[hit.LureType]++
		}
	}
	out := make([]Advice, 0, len(priors))
	for _, p := range priors {
		// Bayesian smoothing: (success+2*priorRate)/(n+2)
		n := counts[p.LureType]
		s := success[p.LureType]
		rate := (s + 2*p.Score/100) / (n + 2)
		score := 0.55*p.Score + 0.45*rate*100
		if h.Frenzy && (p.LureType == "MINNOW" || p.LureType == "PENCIL" || p.LureType == "POPPER") {
			score += 6
		}
		p.Score = math.Round(clamp(score)*10) / 10
		out = append(out, p)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Score > out[j].Score })
	if len(out) > 5 {
		out = out[:5]
	}
	if len(out) < 3 {
		out = append(out, fallback()...)
	}
	if len(out) > 5 {
		out = out[:5]
	}
	return out
}

func rulePriors(species string, h HydroHint) []Advice {
	layer, retrieve := "MID", "TWITCH"
	color := "银白"
	reason := "规则先验"
	switch h.PressureTrend {
	case "CRASH_DOWN":
		layer, retrieve, color = "SHALLOW", "TWITCH", "受伤白"
		reason = "气压暴降，鱼上浮，浅层抽停"
	case "CRASH_UP":
		layer, retrieve, color = "DEEP", "SLOW", "暗色"
		reason = "气压暴升，鱼沉底，慢速底层"
	case "STABLE":
		layer, retrieve = "MID", "STEADY"
		reason = "气压平稳，中层匀收搜索"
	}
	if strings.Contains(h.TideWindow, "THIRD") || h.TideWindow == "EARLY_FLOOD" {
		if layer != "DEEP" {
			layer = "SHALLOW"
		}
		reason += " + 初涨三分潮"
	}
	if h.WaterTempC > 0 && h.WaterTempC < 14 {
		layer, retrieve = "DEEP", "SLOW"
		reason += " + 低温深潜"
	}
	if h.WindBeaufort >= 5 {
		color = "荧光"
		reason += " + 大风浑水高对比"
	}

	base := []Advice{
		{LureType: "MINNOW", Color: color, Layer: layer, Retrieve: retrieve, Score: 78, Reason: reason},
		{LureType: "VIB", Color: "金色", Layer: "DEEP", Retrieve: "STEADY", Score: 70, Reason: "振动搜索中下层结构"},
		{LureType: "SOFT", Color: "青绿", Layer: layer, Retrieve: "SLOW", Score: 68, Reason: "软虫贴结构逗钓"},
		{LureType: "PENCIL", Color: "银白", Layer: "SHALLOW", Retrieve: "WALK", Score: 64, Reason: "水面/亚水面拨水"},
		{LureType: "POPPER", Color: "橙红", Layer: "SHALLOW", Retrieve: "POP", Score: 60, Reason: "波扒引爆表层"},
		{LureType: "JIG", Color: "暗色", Layer: "DEEP", Retrieve: "HOP", Score: 62, Reason: "铁板探深槽"},
		{LureType: "SPOON", Color: "银白", Layer: "MID", Retrieve: "STEADY", Score: 58, Reason: "亮片远投搜索"},
		{LureType: "SPINNER", Color: "金色", Layer: "MID", Retrieve: "STEADY", Score: 56, Reason: "复合亮片扰动"},
	}
	switch strings.ToUpper(species) {
	case "YELLOWCHECK":
		boost(&base, "MINNOW", 10)
		boost(&base, "PENCIL", 8)
	case "MANDARIN":
		boost(&base, "SOFT", 12)
		boost(&base, "VIB", 6)
	case "SNAKEHEAD":
		boost(&base, "SOFT", 10)
		boost(&base, "POPPER", 8)
	case "SEA_BASS":
		boost(&base, "MINNOW", 8)
		boost(&base, "JIG", 8)
	}
	for i := range base {
		base[i].Label = labels[base[i].LureType]
	}
	return base
}

func boost(xs *[]Advice, typ string, add float64) {
	for i := range *xs {
		if (*xs)[i].LureType == typ {
			(*xs)[i].Score += add
			(*xs)[i].Reason += " · 针对目标鱼种加权"
		}
	}
}

func fallback() []Advice {
	return []Advice{
		{LureType: "MINNOW", Label: "米诺", Color: "银白", Layer: "MID", Retrieve: "TWITCH", Score: 60, Reason: "冷启动默认米诺"},
		{LureType: "SOFT", Label: "软虫", Color: "青绿", Layer: "BOTTOM", Retrieve: "SLOW", Score: 55, Reason: "冷启动默认软虫"},
		{LureType: "VIB", Label: "VIB", Color: "金色", Layer: "DEEP", Retrieve: "STEADY", Score: 52, Reason: "冷启动默认 VIB"},
	}
}

func clamp(v float64) float64 {
	if v < 0 {
		return 0
	}
	if v > 100 {
		return 100
	}
	return v
}
