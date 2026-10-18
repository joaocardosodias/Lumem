package adapter_test

import (
	"strings"
	"testing"
	"time"

	"github.com/g04/backend/internal/adapter/security"
)

func TestBcryptPasswordHasher(t *testing.T) {
	hasherDefault := security.NewBcryptPasswordHasher(0)
	hasherCustom := security.NewBcryptPasswordHasher(4)

	password := "senhaForte123!"
	hashed, err := hasherCustom.Hash(password)
	if err != nil {
		t.Fatalf("falha ao gerar hash: %v", err)
	}

	if err := hasherDefault.Compare(hashed, password); err != nil {
		t.Fatalf("esperava sucesso na comparacao da senha correta: %v", err)
	}

	if err := hasherDefault.Compare(hashed, "senhaErrada"); err == nil {
		t.Fatalf("esperava erro na comparacao com senha incorreta")
	}

	longPassword := strings.Repeat("a", 100)
	_, errLong := hasherDefault.Hash(longPassword)
	if errLong == nil {
		t.Fatalf("esperava erro ao fazer hash de senha maior que 72 bytes")
	}
}

func TestJWTTokenGenerator(t *testing.T) {
	gen := security.NewJWTTokenGenerator("minha-chave-secreta-jwt")
	token, err := gen.Generate(map[string]any{
		"sub":   "user-1",
		"email": "user@teste.com",
		"role":  "operator",
	}, time.Hour)

	if err != nil {
		t.Fatalf("falha ao gerar token: %v", err)
	}

	if token == "" {
		t.Fatalf("token gerado nao pode ser vazio")
	}
}
