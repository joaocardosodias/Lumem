package usecase

import (
	"context"
	"errors"
	"time"

	"github.com/g04/backend/internal/domain"
)

type LoginInputDTO struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type UserSummaryDTO struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
	Role  string `json:"role"`
}

type LoginOutputDTO struct {
	Token string         `json:"token"`
	User  UserSummaryDTO `json:"user"`
}

type LoginUseCase struct {
	userRepo      domain.UserRepository
	hasher        domain.PasswordHasher
	tokenGen      domain.TokenGenerator
	tokenDuration time.Duration
}

func NewLoginUseCase(
	userRepo domain.UserRepository,
	hasher domain.PasswordHasher,
	tokenGen domain.TokenGenerator,
	tokenDuration time.Duration,
) *LoginUseCase {
	return &LoginUseCase{
		userRepo:      userRepo,
		hasher:        hasher,
		tokenGen:      tokenGen,
		tokenDuration: tokenDuration,
	}
}

func (uc *LoginUseCase) Execute(ctx context.Context, input LoginInputDTO) (*LoginOutputDTO, error) {
	if input.Email == "" || input.Password == "" {
		return nil, errors.New("e-mail e senha são obrigatórios")
	}

	user, err := uc.userRepo.GetByEmail(ctx, input.Email)
	if err != nil {
		if errors.Is(err, domain.ErrUserNotFound) {
			return nil, domain.ErrInvalidCredentials
		}
		return nil, err
	}

	if !user.IsActive {
		return nil, domain.ErrUserInactive
	}

	err = uc.hasher.Compare(user.PasswordHash, input.Password)
	if err != nil {
		return nil, domain.ErrInvalidCredentials
	}

	claims := map[string]any{
		"sub":   user.ID,
		"name":  user.Name,
		"email": user.Email,
		"role":  user.Role,
	}

	tokenString, err := uc.tokenGen.Generate(claims, uc.tokenDuration)
	if err != nil {
		return nil, err
	}

	return &LoginOutputDTO{
		Token: tokenString,
		User: UserSummaryDTO{
			ID:    user.ID,
			Name:  user.Name,
			Email: user.Email,
			Role:  user.Role,
		},
	}, nil
}
