package usecase

import (
	"context"
	"errors"
	"time"

	"github.com/g04/backend/internal/domain"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
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
	jwtSecret     []byte
	tokenDuration time.Duration
}

func NewLoginUseCase(userRepo domain.UserRepository, jwtSecret string, tokenDuration time.Duration) *LoginUseCase {
	return &LoginUseCase{
		userRepo:      userRepo,
		jwtSecret:     []byte(jwtSecret),
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

	err = bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(input.Password))
	if err != nil {
		return nil, domain.ErrInvalidCredentials
	}

	now := time.Now()
	claims := jwt.MapClaims{
		"sub":   user.ID,
		"name":  user.Name,
		"email": user.Email,
		"role":  user.Role,
		"iat":   now.Unix(),
		"exp":   now.Add(uc.tokenDuration).Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString(uc.jwtSecret)
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
