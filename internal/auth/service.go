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
	store  RefreshStore
	clock  clock.Clock
}

func NewService(users UserRepository, hasher PasswordHasher, tokens TokenIssuer, mailer Mailer, store RefreshStore, c clock.Clock) *Service {
	return &Service{users: users, hasher: hasher, tokens: tokens, mailer: mailer, store: store, clock: c}
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
	if err != nil {
		return User{}, Tokens{}, err
	}
	if err := s.store.Save(ctx, refreshRecord(t, u.ID, id.New())); err != nil {
		return User{}, Tokens{}, err
	}
	return u, t, nil
}

func (s *Service) Refresh(ctx context.Context, refresh string) (Tokens, error) {
	userID, tokenID, err := s.tokens.VerifyRefresh(refresh)
	if err != nil {
		return Tokens{}, ErrInvalidCredentials
	}
	rec, err := s.store.Consume(ctx, tokenID)
	if errors.Is(err, ErrRefreshUsed) {
		if err := s.store.RevokeSession(ctx, tokenID); err != nil {
			return Tokens{}, err
		}
		return Tokens{}, ErrInvalidCredentials
	}
	if isRefreshRejected(err) {
		return Tokens{}, ErrInvalidCredentials
	}
	if err != nil {
		return Tokens{}, err
	}
	if rec.UserID != userID {
		return Tokens{}, ErrInvalidCredentials
	}
	if _, err := s.users.ByID(ctx, userID); err != nil {
		return Tokens{}, ErrInvalidCredentials
	}
	t, err := s.tokens.Issue(userID)
	if err != nil {
		return Tokens{}, err
	}
	err = s.store.Save(ctx, refreshRecord(t, userID, rec.FamilyID))
	if isRefreshRejected(err) {
		return Tokens{}, ErrInvalidCredentials
	}
	if err != nil {
		return Tokens{}, err
	}
	return t, nil
}

func (s *Service) Logout(ctx context.Context, refresh string) error {
	_, tokenID, err := s.tokens.VerifyRefresh(refresh)
	if err != nil {
		return nil
	}
	return s.store.RevokeSession(ctx, tokenID)
}

func refreshRecord(t Tokens, userID, familyID string) RefreshRecord {
	return RefreshRecord{ID: t.RefreshID, FamilyID: familyID, UserID: userID, ExpiresAt: t.RefreshExpires}
}

func isRefreshRejected(err error) bool {
	return errors.Is(err, ErrRefreshUnknown) || errors.Is(err, ErrRefreshUsed) || errors.Is(err, ErrRefreshRevoked)
}

func (s *Service) User(ctx context.Context, userID string) (User, error) {
	return s.users.ByID(ctx, userID)
}

func (s *Service) IDByNickname(ctx context.Context, nickname string) (string, error) {
	u, err := s.users.ByNickname(ctx, strings.TrimSpace(nickname))
	return u.ID, err
}
