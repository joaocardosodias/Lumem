package domain

import (
	"errors"
	"time"
)

var (
	ErrUserNotFound          = errors.New("usuário não encontrado")
	ErrInvalidCredentials    = errors.New("e-mail ou senha inválidos")
	ErrUserInactive          = errors.New("usuário inativo")
	ErrEmailAlreadyExists    = errors.New("e-mail já cadastrado")
	ErrEmailPasswordRequired = errors.New("e-mail e senha sao obrigatorios")
)

type User struct {
	ID           string
	Name         string
	Email        string
	PasswordHash string
	Role         string
	IsActive     bool
	CreatedAt    time.Time
	UpdatedAt    time.Time
}
