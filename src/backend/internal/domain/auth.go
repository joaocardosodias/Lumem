package domain

import "time"

type TokenGenerator interface {
	Generate(claims map[string]any, ttl time.Duration) (string, error)
}

type PasswordHasher interface {
	Compare(hashedPassword, password string) error
	Hash(password string) (string, error)
}
