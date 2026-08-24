package spot

import (
	"crypto/sha256"
	"encoding/binary"
	"math"
)

const cellDeg = 0.012 // ≈ 1.3 km latitude

func CanSeeExact(visibility, ownerID, viewerID, clubID string, sameClub, isFriend bool) bool {
	if viewerID != "" && viewerID == ownerID {
		return true
	}
	switch visibility {
	case "PUBLIC":
		return true
	case "CLUB":
		return sameClub && clubID != ""
	case "FRIENDS":
		return isFriend
	default:
		return false
	}
}

func Fuzz(lat, lon float64, spotID string) (float64, float64) {
	glat := math.Floor(lat/cellDeg)*cellDeg + cellDeg/2
	glon := math.Floor(lon/cellDeg)*cellDeg + cellDeg/2
	jlat, jlon := jitter(spotID)
	outLat := glat + jlat*cellDeg*0.18
	outLon := glon + jlon*cellDeg*0.18
	return round5(outLat), round5(outLon)
}

func HaversineKm(lat1, lon1, lat2, lon2 float64) float64 {
	const r = 6371.0
	p1 := lat1 * math.Pi / 180
	p2 := lat2 * math.Pi / 180
	dphi := (lat2 - lat1) * math.Pi / 180
	dl := (lon2 - lon1) * math.Pi / 180
	a := math.Sin(dphi/2)*math.Sin(dphi/2) + math.Cos(p1)*math.Cos(p2)*math.Sin(dl/2)*math.Sin(dl/2)
	return 2 * r * math.Asin(math.Min(1, math.Sqrt(a)))
}

func jitter(id string) (float64, float64) {
	sum := sha256.Sum256([]byte("lure-fuzz:" + id))
	u1 := binary.BigEndian.Uint32(sum[0:4])
	u2 := binary.BigEndian.Uint32(sum[4:8])
	return float64(u1)/math.MaxUint32*2 - 1, float64(u2)/math.MaxUint32*2 - 1
}

func round5(v float64) float64 { return math.Round(v*1e5) / 1e5 }
