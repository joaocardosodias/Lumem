package security

import (
	"time"

	"github.com/g04/backend/internal/domain"
	"github.com/golang-jwt/jwt/v5"
)

var _ domain.TokenGenerator = (*JWTTokenGenerator)(nil)

type JWTTokenGenerator struct {
	secret []byte
}

func NewJWTTokenGenerator(secret string) *JWTTokenGenerator {
	return &JWTTokenGenerator{secret: []byte(secret)}
}

func (g *JWTTokenGenerator) Generate(claims map[string]any, ttl time.Duration) (string, error) {
	now := time.Now()
	mapClaims := jwt.MapClaims{
		"iat": now.Unix(),
		"exp": now.Add(ttl).Unix(),
	}

	for k, v := range claims {
		mapClaims[k] = v
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, mapClaims)
	return token.SignedString(g.secret)
}
