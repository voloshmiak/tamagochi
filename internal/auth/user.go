package auth

import (
	"context"
	"fmt"
	"net/mail"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/voloshmiak/tamagochi/internal/platform/apperr"
)

type User struct {
	ID            string
	Email         string
	Nickname      string
	PasswordHash  []byte
	EmailVerified bool
	CreatedAt     time.Time
}

type RegisterInput struct {
	Email    string
	Password string
	Nickname string
}

func (in RegisterInput) normalize() (RegisterInput, error) {
	addr, err := mail.ParseAddress(strings.TrimSpace(in.Email))
	if err != nil {
		return in, fmt.Errorf("%w: invalid email", apperr.ErrInvalid)
	}
	in.Email = strings.ToLower(addr.Address)
	if utf8.RuneCountInString(in.Password) < 8 {
		return in, fmt.Errorf("%w: password must be at least 8 characters", apperr.ErrInvalid)
	}
	in.Nickname = strings.TrimSpace(in.Nickname)
	if n := utf8.RuneCountInString(in.Nickname); n < 3 || n > 20 {
		return in, fmt.Errorf("%w: nickname must be 3-20 characters", apperr.ErrInvalid)
	}
	return in, nil
}

var (
	ErrUserNotFound       = fmt.Errorf("%w: user", apperr.ErrNotFound)
	ErrEmailTaken         = fmt.Errorf("%w: email already registered", apperr.ErrConflict)
	ErrNicknameTaken      = fmt.Errorf("%w: nickname already taken", apperr.ErrConflict)
	ErrInvalidCredentials = fmt.Errorf("%w: invalid email or password", apperr.ErrUnauthorized)
)

type UserRepository interface {
	Create(ctx context.Context, u User) error
	ByID(ctx context.Context, id string) (User, error)
	ByEmail(ctx context.Context, email string) (User, error)
	ByNickname(ctx context.Context, nickname string) (User, error)
}

type PasswordHasher interface {
	Hash(password string) ([]byte, error)
	Compare(hash []byte, password string) error
}

type Tokens struct {
	Access         string
	AccessExpires  time.Time
	Refresh        string
	RefreshExpires time.Time
}

type TokenIssuer interface {
	Issue(userID string) (Tokens, error)
	VerifyAccess(token string) (userID string, err error)
	VerifyRefresh(token string) (userID string, err error)
}

type Mailer interface {
	SendVerification(ctx context.Context, email, token string) error
}
