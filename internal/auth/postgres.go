package auth

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/voloshmiak/tamagochi/internal/platform/pg"
	"github.com/voloshmiak/tamagochi/internal/platform/pg/pgdb"
)

// PgUserRepository stores users in PostgreSQL. Email and nickname uniqueness
// is enforced by unique indexes, so concurrent registrations cannot both win.
type PgUserRepository struct {
	Pool *pgxpool.Pool
}

func (r PgUserRepository) Create(ctx context.Context, u User) error {
	err := pg.Queries(ctx, r.Pool).CreateUser(ctx, pgdb.CreateUserParams{
		ID:            u.ID,
		Email:         u.Email,
		Nickname:      u.Nickname,
		PasswordHash:  u.PasswordHash,
		EmailVerified: u.EmailVerified,
		CreatedAt:     u.CreatedAt,
	})
	switch pg.UniqueViolation(err) {
	case "":
	case "users_email_key":
		return ErrEmailTaken
	case "users_nickname_key":
		return ErrNicknameTaken
	}
	if err != nil {
		return fmt.Errorf("create user: %w", err)
	}
	return nil
}

func (r PgUserRepository) ByID(ctx context.Context, id string) (User, error) {
	return userOf(pg.Queries(ctx, r.Pool).UserByID(ctx, id))
}

func (r PgUserRepository) ByEmail(ctx context.Context, email string) (User, error) {
	return userOf(pg.Queries(ctx, r.Pool).UserByEmail(ctx, email))
}

func (r PgUserRepository) ByNickname(ctx context.Context, nickname string) (User, error) {
	u, err := userOf(pg.Queries(ctx, r.Pool).UserByNickname(ctx, nickname))
	if err != nil {
		return User{}, fmt.Errorf("%w (nickname %q)", err, nickname)
	}
	return u, nil
}

func (r PgUserRepository) MarkEmailVerified(ctx context.Context, id string) error {
	n, err := pg.Queries(ctx, r.Pool).MarkEmailVerified(ctx, id)
	if err != nil {
		return fmt.Errorf("mark email verified: %w", err)
	}
	if n == 0 {
		return ErrUserNotFound
	}
	return nil
}

func userOf(row pgdb.User, err error) (User, error) {
	if pg.IsNoRows(err) {
		return User{}, ErrUserNotFound
	}
	if err != nil {
		return User{}, fmt.Errorf("load user: %w", err)
	}
	return User{
		ID:            row.ID,
		Email:         row.Email,
		Nickname:      row.Nickname,
		PasswordHash:  row.PasswordHash,
		EmailVerified: row.EmailVerified,
		CreatedAt:     row.CreatedAt,
	}, nil
}

// PgVerificationRepository stores pending email verifications in PostgreSQL.
type PgVerificationRepository struct {
	Pool *pgxpool.Pool
}

func (r PgVerificationRepository) Save(ctx context.Context, v Verification) error {
	err := pg.Queries(ctx, r.Pool).CreateEmailVerification(ctx, pgdb.CreateEmailVerificationParams{
		TokenHash: v.TokenHash,
		UserID:    v.UserID,
		ExpiresAt: v.ExpiresAt,
	})
	if err != nil {
		return fmt.Errorf("save verification: %w", err)
	}
	return nil
}

// Consume deletes the verification in the same statement that reads it, so a
// token can be used only once even under concurrent requests.
func (r PgVerificationRepository) Consume(ctx context.Context, tokenHash string) (Verification, error) {
	row, err := pg.Queries(ctx, r.Pool).ConsumeEmailVerification(ctx, tokenHash)
	if pg.IsNoRows(err) {
		return Verification{}, ErrVerificationInvalid
	}
	if err != nil {
		return Verification{}, fmt.Errorf("consume verification: %w", err)
	}
	return Verification{TokenHash: row.TokenHash, UserID: row.UserID, ExpiresAt: row.ExpiresAt}, nil
}
