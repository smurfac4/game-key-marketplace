package auth

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"net/mail"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

var (
	ErrUnauthorized = errors.New("invalid credentials or session")
	ErrEmailTaken   = errors.New("email already registered")
)

type ValidationError struct{ Field, Message string }

func (e *ValidationError) Error() string { return e.Field + ": " + e.Message }

type Credentials struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}
type User struct {
	ID        int64     `json:"id"`
	Email     string    `json:"email"`
	CreatedAt time.Time `json:"created_at"`
}
type Account struct {
	User
	PasswordHash string
}
type Store interface {
	CreateUser(context.Context, string, string) (User, error)
	FindByEmail(context.Context, string) (Account, error)
	CreateSession(context.Context, string, int64, time.Time) error
	SessionUser(context.Context, string, int64) (User, error)
	DeleteSession(context.Context, string) error
}
type Service struct {
	Store     Store
	Config    Config
	now       func() time.Time
	dummyHash []byte
}

func New(store Store, config Config) (*Service, error) {

	hash, err := bcrypt.GenerateFromPassword([]byte("not-a-real-account-password"), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}
	return &Service{Store: store, Config: config, now: time.Now, dummyHash: hash}, nil
}
func Validate(c Credentials) (Credentials, error) {
	c.Email = strings.ToLower(strings.TrimSpace(c.Email))
	parsed, err := mail.ParseAddress(c.Email)
	if err != nil || parsed.Address != c.Email || len(c.Email) > 254 || !strings.Contains(c.Email, "@") {
		return c, &ValidationError{"email", "must be a valid email address"}
	}
	if !utf8.ValidString(c.Password) || utf8.RuneCountInString(c.Password) < 12 || len(c.Password) > 72 {
		return c, &ValidationError{"password", "must contain at least 12 characters and at most 72 UTF-8 bytes"}
	}
	return c, nil
}
func (s *Service) Register(ctx context.Context, c Credentials) (User, error) {
	c, err := Validate(c)
	if err != nil {
		return User{}, err
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(c.Password), bcrypt.DefaultCost)
	if err != nil {
		return User{}, err
	}
	return s.Store.CreateUser(ctx, c.Email, string(hash))
}
func (s *Service) Login(ctx context.Context, c Credentials) (User, string, error) {
	c, err := Validate(c)
	if err != nil {
		return User{}, "", err
	}
	account, err := s.Store.FindByEmail(ctx, c.Email)
	if err != nil {
		if !errors.Is(err, ErrUnauthorized) {
			return User{}, "", err
		}
		_ = bcrypt.CompareHashAndPassword(s.dummyHash, []byte(c.Password))
		return User{}, "", ErrUnauthorized
	}
	if bcrypt.CompareHashAndPassword([]byte(account.PasswordHash), []byte(c.Password)) != nil {
		return User{}, "", ErrUnauthorized
	}
	now := s.now().UTC().Truncate(time.Second)
	idBytes := make([]byte, 32)
	if _, err = rand.Read(idBytes); err != nil {
		return User{}, "", err
	}
	id := hex.EncodeToString(idBytes)
	claims := jwt.RegisteredClaims{Issuer: "marketplace-api", Audience: jwt.ClaimStrings{"marketplace-web"}, Subject: strconv.FormatInt(account.ID, 10), ID: id, IssuedAt: jwt.NewNumericDate(now), ExpiresAt: jwt.NewNumericDate(now.Add(SessionTTL))}
	token, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(s.Config.Secret)
	if err != nil {
		return User{}, "", err
	}
	if err = s.Store.CreateSession(ctx, id, account.ID, now.Add(SessionTTL)); err != nil {
		return User{}, "", err
	}
	return account.User, token, nil
}
func (s *Service) claims(raw string) (*jwt.RegisteredClaims, error) {
	claims := &jwt.RegisteredClaims{}
	token, err := jwt.ParseWithClaims(raw, claims, func(*jwt.Token) (any, error) { return s.Config.Secret, nil }, jwt.WithValidMethods([]string{"HS256"}), jwt.WithIssuer("marketplace-api"), jwt.WithAudience("marketplace-web"), jwt.WithExpirationRequired(), jwt.WithIssuedAt(), jwt.WithTimeFunc(s.now))
	if err != nil || !token.Valid || claims.ID == "" || claims.IssuedAt == nil {
		return nil, ErrUnauthorized
	}
	id, err := strconv.ParseInt(claims.Subject, 10, 64)
	if err != nil || id <= 0 {
		return nil, ErrUnauthorized
	}
	return claims, nil
}
func (s *Service) Authenticate(ctx context.Context, raw string) (User, error) {
	claims, err := s.claims(raw)
	if err != nil {
		return User{}, err
	}
	id, _ := strconv.ParseInt(claims.Subject, 10, 64)
	return s.Store.SessionUser(ctx, claims.ID, id)
}
func (s *Service) Logout(ctx context.Context, raw string) error {
	claims, err := s.claims(raw)
	if err != nil {
		return nil
	}
	return s.Store.DeleteSession(ctx, claims.ID)
}
