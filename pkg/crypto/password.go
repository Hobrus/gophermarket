package crypto

import "golang.org/x/crypto/bcrypt"

const passwordCost = 12

// HashPassword hashes raw password using bcrypt with cost 12.
func HashPassword(raw string) (string, error) {
	b, err := bcrypt.GenerateFromPassword([]byte(raw), passwordCost)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// ComparePassword compares hashed password with raw password.
// It returns an error if they do not match.
func ComparePassword(hash, raw string) error {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(raw))
}
