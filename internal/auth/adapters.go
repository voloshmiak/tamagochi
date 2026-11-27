package auth

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"

	"github.com/voloshmiak/tamagochi/internal/platform/clock"
	"github.com/voloshmiak/tamagochi/internal/platform/id"
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

func (j *JWTIssuer) sign(userID, typ, tokenID string, ttl time.Duration) (string, time.Time, error) {
	now := j.Clock.Now()
	exp := now.Add(ttl)
	c := claims{Type: typ, RegisteredClaims: jwt.RegisteredClaims{
		Subject:   userID,
		ID:        tokenID,
		IssuedAt:  jwt.NewNumericDate(now),
		ExpiresAt: jwt.NewNumericDate(exp),
	}}
	s, err := jwt.NewWithClaims(jwt.SigningMethodHS256, c).SignedString(j.Secret)
	return s, exp, err
}

func (j *JWTIssuer) Issue(userID string) (Tokens, error) {
	var t Tokens
	var err error
	if t.Access, t.AccessExpires, err = j.sign(userID, typeAccess, "", j.AccessTTL); err != nil {
		return Tokens{}, err
	}
	t.RefreshID = id.New()
	if t.Refresh, t.RefreshExpires, err = j.sign(userID, typeRefresh, t.RefreshID, j.RefreshTTL); err != nil {
		return Tokens{}, err
	}
	return t, nil
}

func (j *JWTIssuer) verify(token, typ string) (claims, error) {
	var c claims
	_, err := jwt.ParseWithClaims(token, &c, func(*jwt.Token) (any, error) { return j.Secret, nil },
		jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}),
		jwt.WithTimeFunc(j.Clock.Now))
	if err != nil {
		return claims{}, err
	}
	if c.Type != typ {
		return claims{}, errors.New("wrong token type")
	}
	return c, nil
}

func (j *JWTIssuer) VerifyAccess(token string) (string, error) {
	c, err := j.verify(token, typeAccess)
	return c.Subject, err
}

func (j *JWTIssuer) VerifyRefresh(token string) (string, string, error) {
	c, err := j.verify(token, typeRefresh)
	if err != nil {
		return "", "", err
	}
	if c.ID == "" {
		return "", "", errors.New("refresh token without id")
	}
	return c.Subject, c.ID, nil
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

func (r *MemoryUserRepository) MarkEmailVerified(_ context.Context, id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	u, ok := r.users[id]
	if !ok {
		return ErrUserNotFound
	}
	u.EmailVerified = true
	r.users[id] = u
	return nil
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

type MemoryVerificationRepository struct {
	mu     sync.Mutex
	byHash map[string]Verification
}

func NewMemoryVerificationRepository() *MemoryVerificationRepository {
	return &MemoryVerificationRepository{byHash: make(map[string]Verification)}
}

func (r *MemoryVerificationRepository) Save(_ context.Context, v Verification) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.byHash[v.TokenHash] = v
	return nil
}

func (r *MemoryVerificationRepository) Consume(_ context.Context, tokenHash string) (Verification, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	v, ok := r.byHash[tokenHash]
	if !ok {
		return Verification{}, ErrVerificationInvalid
	}
	delete(r.byHash, tokenHash)
	return v, nil
}

type MemoryIdentityRepository struct {
	mu     sync.RWMutex
	byPair map[[2]string]string
}

func NewMemoryIdentityRepository() *MemoryIdentityRepository {
	return &MemoryIdentityRepository{byPair: make(map[[2]string]string)}
}

func (r *MemoryIdentityRepository) UserID(_ context.Context, provider, subject string) (string, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if userID, ok := r.byPair[[2]string{provider, subject}]; ok {
		return userID, nil
	}
	return "", ErrIdentityNotFound
}

func (r *MemoryIdentityRepository) Link(_ context.Context, provider, subject, userID string, _ time.Time) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	key := [2]string{provider, subject}
	if _, ok := r.byPair[key]; !ok {
		r.byPair[key] = userID
	}
	return nil
}
