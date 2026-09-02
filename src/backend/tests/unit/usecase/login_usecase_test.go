package usecase_test

import (
	"context"
	"errors"
	"testing"
	"time"

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
	hash, err := bcrypt.GenerateFromPassword([]byte("operador123"), bcrypt.MinCost)
	if err != nil {
		t.Fatalf("falha ao gerar hash: %v", err)
	}

	repo.users["operador@motiva.com.br"] = &domain.User{
		ID:           "usr_01",
		Name:         "Operador Teste",
		Email:        "operador@motiva.com.br",
		PasswordHash: string(hash),
		Role:         "operator",
		IsActive:     true,
	}

	uc := usecase.NewLoginUseCase(repo, "secret-key", time.Hour)

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
	hash, _ := bcrypt.GenerateFromPassword([]byte("operador123"), bcrypt.MinCost)
	repo.users["operador@motiva.com.br"] = &domain.User{
		ID:           "usr_01",
		Email:        "operador@motiva.com.br",
		PasswordHash: string(hash),
		Role:         "operator",
		IsActive:     true,
	}

	uc := usecase.NewLoginUseCase(repo, "secret-key", time.Hour)

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
	uc := usecase.NewLoginUseCase(repo, "secret-key", time.Hour)

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
	hash, _ := bcrypt.GenerateFromPassword([]byte("operador123"), bcrypt.MinCost)
	repo.users["inativo@motiva.com.br"] = &domain.User{
		ID:           "usr_02",
		Email:        "inativo@motiva.com.br",
		PasswordHash: string(hash),
		Role:         "operator",
		IsActive:     false,
	}

	uc := usecase.NewLoginUseCase(repo, "secret-key", time.Hour)

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
	uc := usecase.NewLoginUseCase(repo, "secret-key", time.Hour)

	input := usecase.LoginInputDTO{
		Email:    "",
		Password: "",
	}

	_, err := uc.Execute(context.Background(), input)
	if err == nil {
		t.Fatalf("esperava erro de campos obrigatorios, mas obteve sucesso")
	}
}
