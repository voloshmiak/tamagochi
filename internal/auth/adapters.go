package auth

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"

	"github.com/voloshmiak/tamagochi/internal/platform/clock"
)

type BcryptHasher struct{ Cost int }

func (h BcryptHasher) Hash(password string) ([]byte, error) {
	return bcrypt.GenerateFromPassword([]byte(password), cmp.Or(h.Cost, bcrypt.DefaultCost))
}

func (BcryptHasher) Compare(hash []byte, password string) error {
	return bcrypt.CompareHashAndPassword(hash, []byte(password))
}

type JWTIssuer struct {
	Secret     []byte
	AccessTTL  time.Duration
	RefreshTTL time.Duration
	Clock      clock.Clock
}

type claims struct {
	Type string `json:"typ"`
	jwt.RegisteredClaims
}

const (
	typeAccess  = "access"
	typeRefresh = "refresh"
)

func (j *JWTIssuer) sign(userID, typ string, ttl time.Duration) (string, time.Time, error) {
	now := j.Clock.Now()
	exp := now.Add(ttl)
	c := claims{Type: typ, RegisteredClaims: jwt.RegisteredClaims{
		Subject:   userID,
		IssuedAt:  jwt.NewNumericDate(now),
		ExpiresAt: jwt.NewNumericDate(exp),
	}}
	s, err := jwt.NewWithClaims(jwt.SigningMethodHS256, c).SignedString(j.Secret)
	return s, exp, err
}

func (j *JWTIssuer) Issue(userID string) (Tokens, error) {
	var t Tokens
	var err error
	if t.Access, t.AccessExpires, err = j.sign(userID, typeAccess, j.AccessTTL); err != nil {
		return Tokens{}, err
	}
	if t.Refresh, t.RefreshExpires, err = j.sign(userID, typeRefresh, j.RefreshTTL); err != nil {
		return Tokens{}, err
	}
	return t, nil
}

func (j *JWTIssuer) verify(token, typ string) (string, error) {
	var c claims
	_, err := jwt.ParseWithClaims(token, &c, func(*jwt.Token) (any, error) { return j.Secret, nil },
		jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}),
		jwt.WithTimeFunc(j.Clock.Now))
	if err != nil {
		return "", err
	}
	if c.Type != typ {
		return "", errors.New("wrong token type")
	}
	return c.Subject, nil
}

func (j *JWTIssuer) VerifyAccess(token string) (string, error) { return j.verify(token, typeAccess) }

func (j *JWTIssuer) VerifyRefresh(token string) (string, error) { return j.verify(token, typeRefresh) }

type LogMailer struct{}

func (LogMailer) SendVerification(ctx context.Context, email, token string) error {
	slog.InfoContext(ctx, "verification email", "to", email, "token", token)
	return nil
}

type MemoryUserRepository struct {
	mu    sync.RWMutex
	users map[string]User
}

func NewMemoryUserRepository() *MemoryUserRepository {
	return &MemoryUserRepository{users: make(map[string]User)}
}

func (r *MemoryUserRepository) Create(_ context.Context, u User) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, existing := range r.users {
		if existing.Email == u.Email {
			return ErrEmailTaken
		}
		if strings.EqualFold(existing.Nickname, u.Nickname) {
			return ErrNicknameTaken
		}
	}
	r.users[u.ID] = u
	return nil
}

func (r *MemoryUserRepository) ByID(_ context.Context, id string) (User, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if u, ok := r.users[id]; ok {
		return u, nil
	}
	return User{}, ErrUserNotFound
}

func (r *MemoryUserRepository) find(match func(User) bool) (User, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, u := range r.users {
		if match(u) {
			return u, nil
		}
	}
	return User{}, ErrUserNotFound
}

func (r *MemoryUserRepository) ByEmail(_ context.Context, email string) (User, error) {
	return r.find(func(u User) bool { return u.Email == email })
}

func (r *MemoryUserRepository) ByNickname(_ context.Context, nickname string) (User, error) {
	u, err := r.find(func(u User) bool { return strings.EqualFold(u.Nickname, nickname) })
	if err != nil {
		return User{}, fmt.Errorf("%w (nickname %q)", err, nickname)
	}
	return u, nil
}
