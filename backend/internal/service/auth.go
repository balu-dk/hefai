package service

import (
	"context"
	"strings"

	"github.com/google/uuid"

	"github.com/balu-dk/hefai/backend/internal/auth"
	"github.com/balu-dk/hefai/backend/internal/domain"
)

type UserRepo interface {
	Create(ctx context.Context, email, displayName, passwordHash string) (*domain.User, error)
	GetByID(ctx context.Context, id uuid.UUID) (*domain.User, error)
	GetByEmail(ctx context.Context, email string) (*domain.User, error)
}

type Auth struct {
	users  UserRepo
	tokens *auth.TokenIssuer
}

func NewAuth(users UserRepo, tokens *auth.TokenIssuer) *Auth {
	return &Auth{users: users, tokens: tokens}
}

type Credentials struct {
	Email       string `json:"email"`
	DisplayName string `json:"displayName"`
	Password    string `json:"password"`
}

type AuthResult struct {
	Token string       `json:"token"`
	User  *domain.User `json:"user"`
}

func (s *Auth) Register(ctx context.Context, in Credentials) (*AuthResult, error) {
	in.Email = strings.TrimSpace(strings.ToLower(in.Email))
	in.DisplayName = strings.TrimSpace(in.DisplayName)
	if in.Email == "" || !strings.Contains(in.Email, "@") {
		return nil, domain.Validation("gyldig e-mail kræves")
	}
	if in.DisplayName == "" {
		in.DisplayName = in.Email
	}
	if len(in.Password) < 8 {
		return nil, domain.Validation("adgangskode skal være mindst 8 tegn")
	}

	hash, err := auth.HashPassword(in.Password)
	if err != nil {
		return nil, err
	}
	user, err := s.users.Create(ctx, in.Email, in.DisplayName, hash)
	if err != nil {
		if err == domain.ErrConflict {
			return nil, domain.Validation("e-mailen er allerede registreret")
		}
		return nil, err
	}
	return s.issue(user)
}

func (s *Auth) Login(ctx context.Context, email, password string) (*AuthResult, error) {
	user, err := s.users.GetByEmail(ctx, strings.TrimSpace(email))
	if err != nil {
		if err == domain.ErrNotFound {
			return nil, domain.ErrUnauthorized
		}
		return nil, err
	}
	if !auth.CheckPassword(user.PasswordHash, password) {
		return nil, domain.ErrUnauthorized
	}
	return s.issue(user)
}

func (s *Auth) GetUser(ctx context.Context, id uuid.UUID) (*domain.User, error) {
	return s.users.GetByID(ctx, id)
}

func (s *Auth) issue(user *domain.User) (*AuthResult, error) {
	token, err := s.tokens.Issue(user.ID)
	if err != nil {
		return nil, err
	}
	return &AuthResult{Token: token, User: user}, nil
}
