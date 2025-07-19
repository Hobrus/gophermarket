package crypto

import "testing"

func TestHashAndComparePassword_Success(t *testing.T) {
	raw := "secret"
	hash, err := HashPassword(raw)
	if err != nil {
		t.Fatalf("unexpected error hashing: %v", err)
	}
	if err := ComparePassword(hash, raw); err != nil {
		t.Errorf("passwords should match: %v", err)
	}
}

func TestHashAndComparePassword_Fail(t *testing.T) {
	raw := "secret"
	hash, err := HashPassword(raw)
	if err != nil {
		t.Fatalf("unexpected error hashing: %v", err)
	}
	if err := ComparePassword(hash, "wrong"); err == nil {
		t.Errorf("expected mismatch error, got nil")
	}
}
