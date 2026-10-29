package auth

import (
	"context"
	"errors"
	"strings"

	"github.com/voloshmiak/tamagochi/internal/platform/clock"
	"github.com/voloshmiak/tamagochi/internal/platform/id"
)

type Service struct {
	users  UserRepository
	hasher PasswordHasher
	tokens TokenIssuer
	mailer Mailer
	clock  clock.Clock
}

func NewService(users UserRepository, hasher PasswordHasher, tokens TokenIssuer, mailer Mailer, c clock.Clock) *Service {
	return &Service{users: users, hasher: hasher, tokens: tokens, mailer: mailer, clock: c}
}

func (s *Service) Register(ctx context.Context, in RegisterInput) (User, error) {
	in, err := in.normalize()
	if err != nil {
		return User{}, err
	}
	if _, err := s.users.ByNickname(ctx, in.Nickname); err == nil {
		return User{}, ErrNicknameTaken
	}
	hash, err := s.hasher.Hash(in.Password)
	if err != nil {
		return User{}, err
	}
	u := User{ID: id.New(), Email: in.Email, Nickname: in.Nickname, PasswordHash: hash, CreatedAt: s.clock.Now()}
	if err := s.users.Create(ctx, u); err != nil {
		return User{}, err
	}
	if err := s.mailer.SendVerification(ctx, u.Email, id.New()); err != nil {
		return User{}, err
	}
	return u, nil
}

func (s *Service) Login(ctx context.Context, email, password string) (User, Tokens, error) {
	u, err := s.users.ByEmail(ctx, strings.ToLower(strings.TrimSpace(email)))
	if errors.Is(err, ErrUserNotFound) {
		return User{}, Tokens{}, ErrInvalidCredentials
	}
	if err != nil {
		return User{}, Tokens{}, err
	}
	if err := s.hasher.Compare(u.PasswordHash, password); err != nil {
		return User{}, Tokens{}, ErrInvalidCredentials
	}
	t, err := s.tokens.Issue(u.ID)
	return u, t, err
}

func (s *Service) Refresh(ctx context.Context, refresh string) (Tokens, error) {
	userID, err := s.tokens.VerifyRefresh(refresh)
	if err != nil {
		return Tokens{}, ErrInvalidCredentials
	}
	if _, err := s.users.ByID(ctx, userID); err != nil {
		return Tokens{}, ErrInvalidCredentials
	}
	return s.tokens.Issue(userID)
}

func (s *Service) User(ctx context.Context, userID string) (User, error) {
	return s.users.ByID(ctx, userID)
}

func (s *Service) IDByNickname(ctx context.Context, nickname string) (string, error) {
	u, err := s.users.ByNickname(ctx, strings.TrimSpace(nickname))
	return u.ID, err
}
