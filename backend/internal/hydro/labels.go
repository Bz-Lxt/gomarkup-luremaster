package hydro

var PressureLabels = map[string]string{
	TrendCrashDown: "气压暴降",
	TrendFall:      "气压缓降",
	TrendStable:    "气压平稳",
	TrendRise:      "气压缓升",
	TrendCrashUp:   "气压暴升",
}

var TideLabels = map[string]string{
	TideSlackLow:   "枯潮",
	TideEarlyFlood: "初涨",
	TideThird:      "三分潮",
	TideHalf:       "半潮",
	TideSeventh:    "七分潮",
	TideSlackHigh:  "满潮",
	TideEarlyEbb:   "初落",
	TideRapidEbb:   "急落",
	"INLAND":       "内陆无潮",
}

var MoonLabels = map[string]string{
	MoonNew:     "新月",
	MoonWaxCres: "娥眉月",
	MoonFirst:   "上弦月",
	MoonWaxGib:  "盈凸月",
	MoonFull:    "满月",
	MoonWanGib:  "亏凸月",
	MoonLast:    "下弦月",
	MoonWanCres: "残月",
}

var AspectLabels = map[string]string{
	AspectOnshore:  "迎风岸",
	AspectOffshore: "背风岸",
	AspectCross:    "侧风岸",
}

func LabelOf(kind, code string) string {
	var m map[string]string
	switch kind {
	case "pressure":
		m = PressureLabels
	case "tide":
		m = TideLabels
	case "moon":
		m = MoonLabels
	case "aspect":
		m = AspectLabels
	}
	if m == nil {
		return code
	}
	if s, ok := m[code]; ok {
		return s
	}
	return code
}

func Annotate(snap *Snapshot) {
	if snap == nil {
		return
	}
	for i := range snap.Contributions {
		if snap.Contributions[i].Label == "" {
			snap.Contributions[i].Label = snap.Contributions[i].Node
		}
	}
}
