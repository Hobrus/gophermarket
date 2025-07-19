package crypto

import "testing"

func TestComparePasswordSuccess(t *testing.T) {
	hash, err := HashPassword("secret")
	if err != nil {
		t.Fatalf("hash error: %v", err)
	}
	if err := ComparePassword(hash, "secret"); err != nil {
		t.Errorf("compare failed: %v", err)
	}
}

func TestComparePasswordFail(t *testing.T) {
	hash, err := HashPassword("secret")
	if err != nil {
		t.Fatalf("hash error: %v", err)
	}
	if err := ComparePassword(hash, "other"); err == nil {
		t.Error("expected compare error")
	}
}
