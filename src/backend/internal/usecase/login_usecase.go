package usecase

import (
	"context"
	"errors"
	"time"

	"github.com/g04/backend/internal/domain"
	"github.com/g04/backend/internal/usecase/port"
)

type LoginInputDTO struct {
	Email    string
	Password string
}

type UserSummaryDTO struct {
	ID    string
	Name  string
	Email string
	Role  string
}

type LoginOutputDTO struct {
	Token string
	User  UserSummaryDTO
}

type LoginUseCase struct {
	userRepo      port.UserRepository
	hasher        port.PasswordHasher
	tokenGen      port.TokenGenerator
	tokenDuration time.Duration
}

func NewLoginUseCase(
	userRepo port.UserRepository,
	hasher port.PasswordHasher,
	tokenGen port.TokenGenerator,
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
		return nil, domain.ErrEmailPasswordRequired
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
