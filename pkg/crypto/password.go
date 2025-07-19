package crypto

import "golang.org/x/crypto/bcrypt"

// HashPassword hashes the raw password using bcrypt with cost 12.
func HashPassword(raw string) (string, error) {
	hashed, err := bcrypt.GenerateFromPassword([]byte(raw), 12)
	if err != nil {
		return "", err
	}
	return string(hashed), nil
}

// ComparePassword compares a bcrypt hashed password with its possible plaintext equivalent.
// Returns an error if they do not match.
func ComparePassword(hash, raw string) error {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(raw))
}
