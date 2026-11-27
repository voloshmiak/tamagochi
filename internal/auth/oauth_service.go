package auth

import (
	"context"
	"errors"
	"fmt"
	"math/rand/v2"
	"net/mail"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/voloshmiak/tamagochi/internal/platform/id"
)

const (
	nicknameMin      = 3
	nicknameBaseMax  = 15
	nicknameAttempts = 20
)

func (s *Service) LoginWithIdentity(ctx context.Context, ident Identity) (User, Tokens, bool, error) {
	if ident.Provider == "" || ident.Subject == "" {
		return User{}, Tokens{}, false, fmt.Errorf("%w: empty identity", ErrProviderFailed)
	}
	var (
		u       User
		created bool
	)
	err := s.tx.WithinTx(ctx, func(ctx context.Context) error {
		var err error
		u, created, err = s.userForIdentity(ctx, ident)
		return err
	})
	if err != nil {
		return User{}, Tokens{}, false, err
	}
	t, err := s.startSession(ctx, u.ID)
	if err != nil {
		return User{}, Tokens{}, false, err
	}
	return u, t, created, nil
}

func (s *Service) userForIdentity(ctx context.Context, ident Identity) (User, bool, error) {
	userID, err := s.identities.UserID(ctx, ident.Provider, ident.Subject)
	if err == nil {
		u, err := s.users.ByID(ctx, userID)
		return u, false, err
	}
	if !errors.Is(err, ErrIdentityNotFound) {
		return User{}, false, err
	}
	addr, perr := mail.ParseAddress(ident.Email)
	if !ident.EmailVerified || perr != nil {
		return User{}, false, ErrNoVerifiedEmail
	}
	email := strings.ToLower(addr.Address)
	now := s.clock.Now()

	u, err := s.users.ByEmail(ctx, email)
	switch {
	case err == nil:
		if !u.EmailVerified {
			if err := s.users.MarkEmailVerified(ctx, u.ID); err != nil {
				return User{}, false, err
			}
			u.EmailVerified = true
		}
		return u, false, s.identities.Link(ctx, ident.Provider, ident.Subject, u.ID, now)
	case !errors.Is(err, ErrUserNotFound):
		return User{}, false, err
	}

	nickname, err := s.freeNickname(ctx, nicknameBase(ident, email))
	if err != nil {
		return User{}, false, err
	}
	u = User{ID: id.New(), Email: email, Nickname: nickname, PasswordHash: []byte{}, EmailVerified: true, CreatedAt: now}
	if err := s.users.Create(ctx, u); err != nil {
		return User{}, false, err
	}
	return u, true, s.identities.Link(ctx, ident.Provider, ident.Subject, u.ID, now)
}

func (s *Service) freeNickname(ctx context.Context, base string) (string, error) {
	candidate := base
	for range nicknameAttempts {
		_, err := s.users.ByNickname(ctx, candidate)
		if errors.Is(err, ErrUserNotFound) {
			return candidate, nil
		}
		if err != nil {
			return "", err
		}
		candidate = fmt.Sprintf("%s%04d", base, rand.IntN(10000))
	}
	return "", fmt.Errorf("could not find a free nickname for %q", base)
}

func nicknameBase(ident Identity, email string) string {
	for _, src := range []string{ident.Login, ident.Name, strings.SplitN(email, "@", 2)[0]} {
		if n := sanitizeNickname(src); utf8.RuneCountInString(n) >= nicknameMin {
			return n
		}
	}
	return "player"
}

func sanitizeNickname(s string) string {
	var b strings.Builder
	n := 0
	for _, r := range s {
		if n == nicknameBaseMax {
			break
		}
		if unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_' || r == '-' || r == '.' {
			b.WriteRune(r)
			n++
		}
	}
	return b.String()
}
