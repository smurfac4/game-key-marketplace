package auth

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"crypto/tls"
	"encoding/hex"
	"errors"
	"fmt"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"math/big"
	"net"
	"net/mail"
	"net/smtp"
	"strings"
	"time"
)

var ErrDisabled = errors.New("sign-in method not configured")

type Mailer interface {
	Send(context.Context, string, string) error
}
type SMTPMailer struct{ Address, Host, User, Password, From string }

func NewSMTP(getenv func(string) string) (Mailer, error) {
	addr := getenv("SMTP_ADDR")
	if addr == "" {
		return nil, nil
	}
	host, _, err := net.SplitHostPort(addr)
	if err != nil {
		return nil, errors.New("invalid SMTP_ADDR")
	}
	from := getenv("SMTP_FROM")
	parsed, err := mail.ParseAddress(from)
	if err != nil || parsed.Address != from {
		return nil, errors.New("invalid SMTP_FROM")
	}
	if getenv("SMTP_USER") == "" || getenv("SMTP_PASSWORD") == "" {
		return nil, errors.New("SMTP authentication required")
	}
	return SMTPMailer{addr, host, getenv("SMTP_USER"), getenv("SMTP_PASSWORD"), from}, nil
}
func (m SMTPMailer) Send(ctx context.Context, to, code string) error {
	conn, err := (&net.Dialer{Timeout: 5 * time.Second}).DialContext(ctx, "tcp", m.Address)
	if err != nil {
		return errors.New("mail delivery failed")
	}
	defer conn.Close()
	deadline := time.Now().Add(8 * time.Second)
	if d, ok := ctx.Deadline(); ok && d.Before(deadline) {
		deadline = d
	}
	conn.SetDeadline(deadline)
	client, err := smtp.NewClient(conn, m.Host)
	if err != nil {
		return errors.New("mail delivery failed")
	}
	defer client.Close()
	if ok, _ := client.Extension("STARTTLS"); !ok {
		return errors.New("SMTP STARTTLS required")
	}
	if err = client.StartTLS(&tls.Config{ServerName: m.Host, MinVersion: tls.VersionTLS12}); err != nil {
		return errors.New("mail delivery failed")
	}
	if err = client.Auth(smtp.PlainAuth("", m.User, m.Password, m.Host)); err != nil {
		return errors.New("mail delivery failed")
	}
	if err = client.Mail(m.From); err != nil {
		return errors.New("mail delivery failed")
	}
	if err = client.Rcpt(to); err != nil {
		return errors.New("mail delivery failed")
	}
	writer, err := client.Data()
	if err != nil {
		return errors.New("mail delivery failed")
	}
	message := "From: " + m.From + "\r\nTo: " + to + "\r\nSubject: Your Keyspace sign-in code\r\nContent-Type: text/plain; charset=UTF-8\r\n\r\nYour sign-in code is " + code + ". It expires in 10 minutes. If you did not request this, ignore this message.\r\n"
	if _, err = writer.Write([]byte(message)); err != nil {
		return errors.New("mail delivery failed")
	}
	if err = writer.Close(); err != nil {
		return errors.New("mail delivery failed")
	}
	return client.Quit()
}
func RandomToken() string {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		panic("secure randomness unavailable")
	}
	return hex.EncodeToString(b)
}
func Digest(value string) string { h := sha256.Sum256([]byte(value)); return hex.EncodeToString(h[:]) }

type Passwordless struct {
	DB      *pgxpool.Pool
	Service *Service
	Mailer  Mailer
}

func (p Passwordless) codeHash(id, code string) string {
	mac := hmac.New(sha256.New, p.Service.Config.Secret)
	mac.Write([]byte(id + ":" + code))
	return hex.EncodeToString(mac.Sum(nil))
}
func (p Passwordless) Request(ctx context.Context, email string) (string, error) {
	if p.Mailer == nil {
		return "", ErrDisabled
	}
	email = strings.ToLower(strings.TrimSpace(email))
	parsed, err := mail.ParseAddress(email)
	if err != nil || parsed.Address != email || len(email) > 254 {
		return "", &ValidationError{"email", "must be a valid email address"}
	}
	n, err := rand.Int(rand.Reader, big.NewInt(1000000))
	if err != nil {
		return "", err
	}
	code := fmt.Sprintf("%06d", n.Int64())
	id := RandomToken()
	_, err = p.DB.Exec(ctx, `WITH cleanup AS (DELETE FROM email_challenges WHERE expires_at<=now()) INSERT INTO email_challenges(id,email,code_hash,expires_at) VALUES($1,$2,$3,now()+interval '10 minutes')`, id, email, p.codeHash(id, code))
	if err != nil {
		return "", err
	}
	if err = p.Mailer.Send(ctx, email, code); err != nil {
		cleanup, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		p.DB.Exec(cleanup, `DELETE FROM email_challenges WHERE id=$1`, id)
		return "", errors.New("mail delivery unavailable")
	}
	return id, nil
}
func (p Passwordless) Verify(ctx context.Context, id, code string) (User, string, error) {
	if len(id) != 64 || len(code) != 6 || strings.Trim(code, "0123456789") != "" {
		return User{}, "", ErrUnauthorized
	}
	tx, err := p.DB.Begin(ctx)
	if err != nil {
		return User{}, "", err
	}
	defer tx.Rollback(context.Background())
	var email, hash string
	var attempts int
	err = tx.QueryRow(ctx, `SELECT email,code_hash,attempts FROM email_challenges WHERE id=$1 AND expires_at>now() FOR UPDATE`, id).Scan(&email, &hash, &attempts)
	if errors.Is(err, pgx.ErrNoRows) {
		return User{}, "", ErrUnauthorized
	}
	if err != nil {
		return User{}, "", err
	}
	if attempts >= 5 {
		return User{}, "", ErrUnauthorized
	}
	if subtle.ConstantTimeCompare([]byte(hash), []byte(p.codeHash(id, code))) != 1 {
		if _, err = tx.Exec(ctx, `UPDATE email_challenges SET attempts=attempts+1 WHERE id=$1`, id); err != nil {
			return User{}, "", err
		}
		if err = tx.Commit(ctx); err != nil {
			return User{}, "", err
		}
		return User{}, "", ErrUnauthorized
	}
	var user User
	err = tx.QueryRow(ctx, `INSERT INTO users(email,password_hash) VALUES($1,'passwordless-only') ON CONFLICT(lower(email)) DO UPDATE SET email=users.email RETURNING id,email,created_at`, email).Scan(&user.ID, &user.Email, &user.CreatedAt)
	if err != nil {
		return User{}, "", err
	}
	if _, err = tx.Exec(ctx, `DELETE FROM email_challenges WHERE id=$1`, id); err != nil {
		return User{}, "", err
	}
	if err = tx.Commit(ctx); err != nil {
		return User{}, "", err
	}
	return p.Service.StartSession(ctx, user)
}
func (p Passwordless) ExternalUser(ctx context.Context, issuer, subject, email string) (User, string, error) {
	tx, err := p.DB.Begin(ctx)
	if err != nil {
		return User{}, "", err
	}
	defer tx.Rollback(context.Background())
	var u User
	err = tx.QueryRow(ctx, `SELECT u.id,u.email,u.created_at FROM external_identities i JOIN users u ON u.id=i.user_id WHERE i.issuer=$1 AND i.subject=$2`, issuer, subject).Scan(&u.ID, &u.Email, &u.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		err = tx.QueryRow(ctx, `INSERT INTO users(email,password_hash) VALUES($1,'passwordless-only') ON CONFLICT(lower(email)) DO UPDATE SET email=users.email RETURNING id,email,created_at`, email).Scan(&u.ID, &u.Email, &u.CreatedAt)
		if err != nil {
			return User{}, "", err
		}
		if _, err = tx.Exec(ctx, `INSERT INTO external_identities(issuer,subject,user_id) VALUES($1,$2,$3) ON CONFLICT DO NOTHING`, issuer, subject, u.ID); err != nil {
			return User{}, "", err
		}
	} else if err != nil {
		return User{}, "", err
	}
	err = tx.QueryRow(ctx, `SELECT u.id,u.email,u.created_at FROM external_identities i JOIN users u ON u.id=i.user_id WHERE i.issuer=$1 AND i.subject=$2`, issuer, subject).Scan(&u.ID, &u.Email, &u.CreatedAt)
	if err != nil {
		return User{}, "", err
	}
	if err = tx.Commit(ctx); err != nil {
		return User{}, "", err
	}
	return p.Service.StartSession(ctx, u)
}
