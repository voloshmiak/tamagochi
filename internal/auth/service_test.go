package auth

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/voloshmiak/tamagochi/internal/platform/clock"
)

const (
	testEmail    = "a@b.co"
	testPassword = "secret123"
)

func newTestService(t *testing.T) (*Service, *clock.Manual) {
	t.Helper()
	clk := clock.NewManual(time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC))
	tokens := &JWTIssuer{
		Secret:     []byte("test-secret"),
		AccessTTL:  15 * time.Minute,
		RefreshTTL: 30 * 24 * time.Hour,
		Clock:      clk,
	}
	svc := NewService(Deps{
		Users:         NewMemoryUserRepository(),
		Verifications: NewMemoryVerificationRepository(),
		Hasher:        BcryptHasher{Cost: 4},
		Tokens:        tokens,
		Mailer:        LogMailer{},
		Refresh:       NewMemoryRefreshStore(clk),
		Clock:         clk,
		VerifyURL:     "https://example.test/api/v1/auth/verify",
	})
	in := RegisterInput{Email: testEmail, Password: testPassword, Nickname: "misha"}
	if _, err := svc.Register(context.Background(), in); err != nil {
		t.Fatal(err)
	}
	return svc, clk
}

func login(t *testing.T, svc *Service) Tokens {
	t.Helper()
	_, tok, err := svc.Login(context.Background(), testEmail, testPassword)
	if err != nil {
		t.Fatal(err)
	}
	return tok
}

func TestRefresh_RotatesToken(t *testing.T) {
	svc, _ := newTestService(t)
	ctx := context.Background()
	t1 := login(t, svc)

	t2, err := svc.Refresh(ctx, t1.Refresh)
	if err != nil {
		t.Fatal(err)
	}
	if t2.Refresh == t1.Refresh {
		t.Fatal("refresh token must change on every refresh")
	}
	if _, err := svc.Refresh(ctx, t2.Refresh); err != nil {
		t.Fatalf("the rotated token must work: %v", err)
	}
}

func TestRefresh_ReuseRevokesWholeSession(t *testing.T) {
	svc, _ := newTestService(t)
	ctx := context.Background()
	t1 := login(t, svc)
	t2, err := svc.Refresh(ctx, t1.Refresh)
	if err != nil {
		t.Fatal(err)
	}

	if _, err := svc.Refresh(ctx, t1.Refresh); !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("reusing a spent token: want ErrInvalidCredentials, got %v", err)
	}
	if _, err := svc.Refresh(ctx, t2.Refresh); !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("after reuse the whole session must be dead, got %v", err)
	}
}

func TestLogout_RevokesOnlyThatSession(t *testing.T) {
	svc, _ := newTestService(t)
	ctx := context.Background()
	phone := login(t, svc)
	laptop := login(t, svc)

	if err := svc.Logout(ctx, phone.Refresh); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Refresh(ctx, phone.Refresh); !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("refresh after logout: want ErrInvalidCredentials, got %v", err)
	}
	if _, err := svc.Refresh(ctx, laptop.Refresh); err != nil {
		t.Fatalf("other session must survive logout: %v", err)
	}
}

func TestLogout_GarbageTokenIsNotAnError(t *testing.T) {
	svc, _ := newTestService(t)
	if err := svc.Logout(context.Background(), "not-a-jwt"); err != nil {
		t.Fatalf("logout with a bad token must succeed, got %v", err)
	}
}

func TestRefresh_Expired(t *testing.T) {
	svc, clk := newTestService(t)
	t1 := login(t, svc)

	clk.Advance(30*24*time.Hour + time.Second)
	if _, err := svc.Refresh(context.Background(), t1.Refresh); !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("expired refresh: want ErrInvalidCredentials, got %v", err)
	}
}

func TestRefresh_RejectsAccessToken(t *testing.T) {
	svc, _ := newTestService(t)
	t1 := login(t, svc)

	if _, err := svc.Refresh(context.Background(), t1.Access); !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("access token used as refresh: want ErrInvalidCredentials, got %v", err)
	}
}
