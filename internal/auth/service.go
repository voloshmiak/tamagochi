package auth

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"log/slog"
	"net/url"
	"strings"
	"time"

	"github.com/voloshmiak/tamagochi/internal/platform/clock"
	"github.com/voloshmiak/tamagochi/internal/platform/id"
)

// VerificationTTL is how long the link from the verification email stays valid.
const VerificationTTL = 48 * time.Hour

type Deps struct {
	Users         UserRepository
	Verifications VerificationRepository
	Hasher        PasswordHasher
	Tokens        TokenIssuer
	Mailer        Mailer
	Refresh       RefreshStore
	Clock         clock.Clock
	// VerifyURL is the absolute URL of the verification endpoint; the token is
	// appended as the "token" query parameter.
	VerifyURL string
}

type Service struct {
	users         UserRepository
	verifications VerificationRepository
	hasher        PasswordHasher
	tokens        TokenIssuer
	mailer        Mailer
	store         RefreshStore
	clock         clock.Clock
	verifyURL     string
}

func NewService(d Deps) *Service {
	return &Service{
		users:         d.Users,
		verifications: d.Verifications,
		hasher:        d.Hasher,
		tokens:        d.Tokens,
		mailer:        d.Mailer,
		store:         d.Refresh,
		clock:         d.Clock,
		verifyURL:     d.VerifyURL,
	}
}

// Register creates the account and sends the email verification link
// (FR-AUTH-01). A failure to send the email does not undo the registration.
func (s *Service) Register(ctx context.Context, in RegisterInput) (User, error) {
	in, err := in.normalize()
	if err != nil {
		return User{}, err
	}
	if _, err := s.users.ByNickname(ctx, in.Nickname); err == nil {
		return User{}, ErrNicknameTaken
	}
	if _, err := s.users.ByEmail(ctx, in.Email); err == nil {
		return User{}, ErrEmailTaken
	}
	hash, err := s.hasher.Hash(in.Password)
	if err != nil {
		return User{}, err
	}
	now := s.clock.Now()
	u := User{ID: id.New(), Email: in.Email, Nickname: in.Nickname, PasswordHash: hash, CreatedAt: now}
	if err := s.users.Create(ctx, u); err != nil {
		return User{}, err
	}
	token := id.New()
	v := Verification{TokenHash: hashToken(token), UserID: u.ID, ExpiresAt: now.Add(VerificationTTL)}
	if err := s.verifications.Save(ctx, v); err != nil {
		return User{}, err
	}
	if err := s.mailer.SendVerification(ctx, u.Email, s.verifyLink(token)); err != nil {
		slog.ErrorContext(ctx, "send verification email", "user", u.ID, "err", err)
	}
	return u, nil
}

// VerifyEmail confirms the address of the user the token was issued to.
// Each token works once.
func (s *Service) VerifyEmail(ctx context.Context, token string) error {
	v, err := s.verifications.Consume(ctx, hashToken(token))
	if err != nil {
		return err
	}
	if !v.ExpiresAt.After(s.clock.Now()) {
		return ErrVerificationInvalid
	}
	return s.users.MarkEmailVerified(ctx, v.UserID)
}

func (s *Service) verifyLink(token string) string {
	return s.verifyURL + "?" + url.Values{"token": {token}}.Encode()
}

// hashToken keeps only a digest of the token in storage, so a leaked table
// does not let anyone verify addresses.
func hashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
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
