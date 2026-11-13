package auth

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

type DB interface {
	QueryRow(context.Context, string, ...any) pgx.Row
	Exec(context.Context, string, ...any) (pgconn.CommandTag, error)
}
type Repository struct{ DB DB }

func (r Repository) CreateUser(ctx context.Context, email, hash string) (User, error) {
	var u User
	err := r.DB.QueryRow(ctx, `INSERT INTO users(email,password_hash) VALUES ($1,$2) RETURNING id,email,created_at`, email, hash).Scan(&u.ID, &u.Email, &u.CreatedAt)
	var pgerr *pgconn.PgError
	if errors.As(err, &pgerr) && pgerr.Code == "23505" {
		return User{}, ErrEmailTaken
	}
	return u, err
}
func (r Repository) FindByEmail(ctx context.Context, email string) (Account, error) {
	var a Account
	err := r.DB.QueryRow(ctx, `SELECT id,email,created_at,password_hash FROM users WHERE lower(email)=$1`, email).Scan(&a.ID, &a.Email, &a.CreatedAt, &a.PasswordHash)
	if errors.Is(err, pgx.ErrNoRows) {
		return a, ErrUnauthorized
	}
	return a, err
}
func (r Repository) CreateSession(ctx context.Context, id string, userID int64, expires time.Time) error {
	_, err := r.DB.Exec(ctx, `WITH cleanup AS (DELETE FROM auth_sessions WHERE user_id=$2 AND expires_at<=now())
 INSERT INTO auth_sessions(id,user_id,expires_at) VALUES ($1,$2,$3)`, id, userID, expires)
	return err
}
func (r Repository) SessionUser(ctx context.Context, id string, userID int64) (User, error) {
	var u User
	err := r.DB.QueryRow(ctx, `SELECT u.id,u.email,u.created_at FROM auth_sessions s JOIN users u ON u.id=s.user_id
 WHERE s.id=$1 AND s.user_id=$2 AND s.expires_at>now()`, id, userID).Scan(&u.ID, &u.Email, &u.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return u, ErrUnauthorized
	}
	return u, err
}
func (r Repository) DeleteSession(ctx context.Context, id string) error {
	_, err := r.DB.Exec(ctx, "DELETE FROM auth_sessions WHERE id=$1", id)
	return err
}
