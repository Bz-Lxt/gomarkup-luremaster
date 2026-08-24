package auth

import "testing"

func TestPasswordAndJWTRoundTrip(t *testing.T) {
	hash, err := HashPassword("LureHunt@2026")
	if err != nil {
		t.Fatal(err)
	}
	if !CheckPassword(hash, "LureHunt@2026") {
		t.Fatal("good password rejected")
	}
	if CheckPassword(hash, "wrong") {
		t.Fatal("accepted wrong password")
	}
	svc := New("test-secret-key-for-jwt")
	access, refresh, err := svc.IssueTokens("user-1")
	if err != nil {
		t.Fatal(err)
	}
	claims, err := svc.ParseAccess(access)
	if err != nil {
		t.Fatal(err)
	}
	if claims.UserID != "user-1" {
		t.Fatalf("uid %s", claims.UserID)
	}
	if _, err := svc.ParseAccess(access + "x"); err == nil {
		t.Fatal("tampered token")
	}
	rc, err := svc.ParseRefresh(refresh)
	if err != nil || rc.UserID != "user-1" {
		t.Fatalf("refresh %v", err)
	}
	if _, err := svc.ParseAccess(refresh); err == nil {
		t.Fatal("refresh accepted as access")
	}
}
