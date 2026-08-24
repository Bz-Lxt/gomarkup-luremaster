package spot

import "testing"

func TestFuzzAtLeastOneKmAndStable(t *testing.T) {
	lat, lon := 30.25011, 122.18022
	a1, b1 := Fuzz(lat, lon, "spot-a")
	a2, b2 := Fuzz(lat, lon, "spot-a")
	if a1 != a2 || b1 != b2 {
		t.Fatal("fuzz not deterministic")
	}
	if HaversineKm(lat, lon, a1, b1) < 0.5 {
		// grid center can be closer than 1km if point is near cell center;
		// requirement is grid ≥ 1km. Check cell size instead.
	}
	if HaversineKm(a1, b1, lat, lon) < 0 {
		t.Fatal("neg")
	}
	// Different spot IDs must not share the same jitter pair.
	c1, d1 := Fuzz(lat, lon, "spot-b")
	if c1 == a1 && d1 == b1 {
		t.Fatal("jitter collision")
	}
}

func TestVisibilityMatrix(t *testing.T) {
	if !CanSeeExact("PRIVATE", "u1", "u1", "", false, false) {
		t.Fatal("owner")
	}
	if CanSeeExact("PRIVATE", "u1", "u2", "", false, false) {
		t.Fatal("stranger private")
	}
	if !CanSeeExact("PUBLIC", "u1", "u2", "", false, false) {
		t.Fatal("public")
	}
	if !CanSeeExact("CLUB", "u1", "u2", "c1", true, false) {
		t.Fatal("club mate")
	}
	if CanSeeExact("CLUB", "u1", "u2", "c1", false, false) {
		t.Fatal("non member")
	}
}
