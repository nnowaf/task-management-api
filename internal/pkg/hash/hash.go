package hash

import "golang.org/x/crypto/bcrypt"

// Password hashes a plaintext password using bcrypt at the default cost.
func Password(plain string) (string, error) {
	b, err := bcrypt.GenerateFromPassword([]byte(plain), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// Check reports whether the plaintext password matches the stored bcrypt hash.
func Check(hashed, plain string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hashed), []byte(plain)) == nil
}
