package usecase_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/g04/backend/internal/adapter/security"
	"github.com/g04/backend/internal/domain"
	"github.com/g04/backend/internal/usecase"
	"golang.org/x/crypto/bcrypt"
)

type mockUserRepo struct {
	users map[string]*domain.User
}

func newMockUserRepo() *mockUserRepo {
	return &mockUserRepo{users: make(map[string]*domain.User)}
}

func (m *mockUserRepo) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	u, ok := m.users[email]
	if !ok {
		return nil, domain.ErrUserNotFound
	}
	return u, nil
}

func (m *mockUserRepo) GetByID(ctx context.Context, id string) (*domain.User, error) {
	for _, u := range m.users {
		if u.ID == id {
			return u, nil
		}
	}
	return nil, domain.ErrUserNotFound
}

func (m *mockUserRepo) Create(ctx context.Context, user *domain.User) error {
	m.users[user.Email] = user
	return nil
}

func TestLoginUseCase_Success(t *testing.T) {
	repo := newMockUserRepo()
	hasher := security.NewBcryptPasswordHasher(bcrypt.MinCost)
	tokenGen := security.NewJWTTokenGenerator("secret-key")

	hash, err := hasher.Hash("operador123")
	if err != nil {
		t.Fatalf("falha ao gerar hash: %v", err)
	}

	repo.users["operador@motiva.com.br"] = &domain.User{
		ID:           "usr_01",
		Name:         "Operador Teste",
		Email:        "operador@motiva.com.br",
		PasswordHash: hash,
		Role:         "operator",
		IsActive:     true,
	}

	uc := usecase.NewLoginUseCase(repo, hasher, tokenGen, time.Hour)

	input := usecase.LoginInputDTO{
		Email:    "operador@motiva.com.br",
		Password: "operador123",
	}

	output, err := uc.Execute(context.Background(), input)
	if err != nil {
		t.Fatalf("esperava sucesso, mas obteve erro: %v", err)
	}

	if output.Token == "" {
		t.Errorf("token nao deveria ser vazio")
	}

	if output.User.Email != "operador@motiva.com.br" {
		t.Errorf("esperava email operador@motiva.com.br, obteve %s", output.User.Email)
	}
}

func TestLoginUseCase_WrongPassword(t *testing.T) {
	repo := newMockUserRepo()
	hasher := security.NewBcryptPasswordHasher(bcrypt.MinCost)
	tokenGen := security.NewJWTTokenGenerator("secret-key")

	hash, _ := hasher.Hash("operador123")
	repo.users["operador@motiva.com.br"] = &domain.User{
		ID:           "usr_01",
		Email:        "operador@motiva.com.br",
		PasswordHash: hash,
		Role:         "operator",
		IsActive:     true,
	}

	uc := usecase.NewLoginUseCase(repo, hasher, tokenGen, time.Hour)

	input := usecase.LoginInputDTO{
		Email:    "operador@motiva.com.br",
		Password: "senha-errada",
	}

	_, err := uc.Execute(context.Background(), input)
	if !errors.Is(err, domain.ErrInvalidCredentials) {
		t.Fatalf("esperava ErrInvalidCredentials, obteve %v", err)
	}
}

func TestLoginUseCase_UserNotFound(t *testing.T) {
	repo := newMockUserRepo()
	hasher := security.NewBcryptPasswordHasher(bcrypt.MinCost)
	tokenGen := security.NewJWTTokenGenerator("secret-key")

	uc := usecase.NewLoginUseCase(repo, hasher, tokenGen, time.Hour)

	input := usecase.LoginInputDTO{
		Email:    "naoexiste@motiva.com.br",
		Password: "qualquer-senha",
	}

	_, err := uc.Execute(context.Background(), input)
	if !errors.Is(err, domain.ErrInvalidCredentials) {
		t.Fatalf("esperava ErrInvalidCredentials para usuario inexistente, obteve %v", err)
	}
}

func TestLoginUseCase_InactiveUser(t *testing.T) {
	repo := newMockUserRepo()
	hasher := security.NewBcryptPasswordHasher(bcrypt.MinCost)
	tokenGen := security.NewJWTTokenGenerator("secret-key")

	hash, _ := hasher.Hash("operador123")
	repo.users["inativo@motiva.com.br"] = &domain.User{
		ID:           "usr_02",
		Email:        "inativo@motiva.com.br",
		PasswordHash: hash,
		Role:         "operator",
		IsActive:     false,
	}

	uc := usecase.NewLoginUseCase(repo, hasher, tokenGen, time.Hour)

	input := usecase.LoginInputDTO{
		Email:    "inativo@motiva.com.br",
		Password: "operador123",
	}

	_, err := uc.Execute(context.Background(), input)
	if !errors.Is(err, domain.ErrUserInactive) {
		t.Fatalf("esperava ErrUserInactive, obteve %v", err)
	}
}

func TestLoginUseCase_EmptyFields(t *testing.T) {
	repo := newMockUserRepo()
	hasher := security.NewBcryptPasswordHasher(bcrypt.MinCost)
	tokenGen := security.NewJWTTokenGenerator("secret-key")

	uc := usecase.NewLoginUseCase(repo, hasher, tokenGen, time.Hour)

	input := usecase.LoginInputDTO{
		Email:    "",
		Password: "",
	}

	_, err := uc.Execute(context.Background(), input)
	if !errors.Is(err, domain.ErrEmailPasswordRequired) {
		t.Fatalf("esperava ErrEmailPasswordRequired, mas obteve %v", err)
	}
}

type errTokenGen struct{}

func (e *errTokenGen) Generate(claims map[string]any, duration time.Duration) (string, error) {
	return "", errors.New("jwt signing failed")
}

type errUserRepo struct {
	mockUserRepo
	err error
}

func (r *errUserRepo) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	if r.err != nil {
		return nil, r.err
	}
	return r.mockUserRepo.GetByEmail(ctx, email)
}

func TestLoginUseCase_DBError(t *testing.T) {
	repo := &errUserRepo{err: errors.New("db connection failure")}
	hasher := security.NewBcryptPasswordHasher(bcrypt.MinCost)
	tokenGen := security.NewJWTTokenGenerator("secret-key")

	uc := usecase.NewLoginUseCase(repo, hasher, tokenGen, time.Hour)
	_, err := uc.Execute(context.Background(), usecase.LoginInputDTO{Email: "a@b.com", Password: "pass"})
	if err == nil || err.Error() != "db connection failure" {
		t.Fatalf("expected db connection failure, got %v", err)
	}
}

func TestLoginUseCase_TokenGenError(t *testing.T) {
	repo := newMockUserRepo()
	hasher := security.NewBcryptPasswordHasher(bcrypt.MinCost)
	hash, _ := hasher.Hash("pass")
	repo.users["valid@test.com"] = &domain.User{
		ID:           "u1",
		Email:        "valid@test.com",
		PasswordHash: hash,
		IsActive:     true,
	}

	uc := usecase.NewLoginUseCase(repo, hasher, &errTokenGen{}, time.Hour)
	_, err := uc.Execute(context.Background(), usecase.LoginInputDTO{Email: "valid@test.com", Password: "pass"})
	if err == nil || err.Error() != "jwt signing failed" {
		t.Fatalf("expected jwt signing failed, got %v", err)
	}
}
