package model

import "strings"

var Structures = []string{
	"SNAG", "ROCK", "PIER", "DROPOFF", "BACKWATER", "INLET",
	"WEED", "CAGE", "WRECK", "EDDY",
}

var StructureLabel = map[string]string{
	"SNAG": "枯树", "ROCK": "乱石", "PIER": "桥墩", "DROPOFF": "深浅交界",
	"BACKWATER": "回湾", "INLET": "入水口", "WEED": "草洞", "CAGE": "网箱",
	"WRECK": "沉船", "EDDY": "洄流带",
}

var WaterTypes = []string{"RESERVOIR", "RIVER", "LAKE", "SEA", "POND"}

var Species = []string{
	"MANDARIN", "SNAKEHEAD", "BASS", "PIKEKILLIFISH", "CHUB",
	"SEA_BASS", "YELLOWCHECK", "CATFISH", "OTHER",
}

var SpeciesLabel = map[string]string{
	"MANDARIN": "鳜鱼", "SNAKEHEAD": "黑鱼", "BASS": "鲈鱼",
	"PIKEKILLIFISH": "马口", "CHUB": "军鱼", "SEA_BASS": "海鲈",
	"YELLOWCHECK": "翘嘴", "CATFISH": "鲶鱼", "OTHER": "其他",
}

var LureTypes = []string{
	"MINNOW", "VIB", "SOFT", "PENCIL", "POPPER", "SPOON", "SPINNER", "JIG",
}

var LureLabel = map[string]string{
	"MINNOW": "米诺", "VIB": "VIB", "SOFT": "软虫", "PENCIL": "铅笔",
	"POPPER": "波扒", "SPOON": "亮片", "SPINNER": "复合亮片", "JIG": "铁板",
}

var Visibilities = []string{"PRIVATE", "CLUB", "FRIENDS", "PUBLIC"}

func NormalizeEnum(s string) string {
	return strings.ToUpper(strings.TrimSpace(s))
}

func InSet(s string, set []string) bool {
	s = NormalizeEnum(s)
	for _, v := range set {
		if v == s {
			return true
		}
	}
	return false
}
