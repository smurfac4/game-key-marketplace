package httpapi

import (
	"context"
	"crypto/subtle"
	"errors"
	"example.com/game-key-marketplace/api/internal/auth"
	"github.com/coreos/go-oidc/v3/oidc"
	"golang.org/x/oauth2"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

type Identity struct {
	Auth           *Auth
	Codes          auth.Passwordless
	Provider       *oidc.Provider
	OAuth          oauth2.Config
	Name, Frontend string
	InitErr        bool
}

func NewIdentity(a *Auth, codes auth.Passwordless) *Identity {
	h := &Identity{Auth: a, Codes: codes, Frontend: os.Getenv("AUTH_WEB_URL")}
	if h.Frontend == "" {
		for origin := range a.Service.Config.Origins {
			h.Frontend = origin
			break
		}
	}
	if !a.Service.Config.Origins[h.Frontend] {
		h.InitErr = true
		return h
	}
	issuer := os.Getenv("OIDC_ISSUER")
	if issuer != "" {
		u, e := url.Parse(issuer)
		if e != nil || u.Host == "" || u.User != nil || (u.Scheme != "https" && !(u.Scheme == "http" && (u.Hostname() == "localhost" || u.Hostname() == "127.0.0.1"))) {
			h.InitErr = true
			return h
		}
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		p, err := oidc.NewProvider(ctx, issuer)
		if err != nil || os.Getenv("OIDC_CLIENT_ID") == "" || os.Getenv("OIDC_CLIENT_SECRET") == "" || os.Getenv("OIDC_REDIRECT_URL") == "" {
			h.InitErr = true
			return h
		}
		h.Provider = p
		h.OAuth = oauth2.Config{ClientID: os.Getenv("OIDC_CLIENT_ID"), ClientSecret: os.Getenv("OIDC_CLIENT_SECRET"), RedirectURL: os.Getenv("OIDC_REDIRECT_URL"), Endpoint: p.Endpoint(), Scopes: []string{oidc.ScopeOpenID, "email", "profile"}}
		h.Name = os.Getenv("OIDC_PROVIDER_NAME")
		if h.Name == "" {
			h.Name = "OpenID Connect"
		}
	}
	return h
}
func (h *Identity) Methods(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	providers := []map[string]string{}
	if h.Provider != nil {
		providers = append(providers, map[string]string{"name": h.Name, "start_url": "/api/auth/oidc/start"})
	}
	writeJSON(w, 200, map[string]any{"data": map[string]any{"password": true, "email_code": h.Codes.Mailer != nil, "providers": providers}})
}
func (h *Identity) Request(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	if h.Auth.limited(w, r) {
		return
	}
	var b struct {
		Email string `json:"email"`
	}
	if !bodyJSON(w, r, &b) {
		return
	}
	id, err := h.Codes.Request(r.Context(), b.Email)
	if errors.Is(err, auth.ErrDisabled) {
		writeError(w, 503, "method_unavailable", "Email sign-in is not configured.", "")
		return
	}
	if err != nil {
		var v *auth.ValidationError
		if errors.As(err, &v) {
			h.Auth.failure(w, r, err)
		} else {
			writeError(w, 503, "mail_unavailable", "Unable to send a sign-in code. Try again later.", "")
		}
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	writeJSON(w, 202, map[string]any{"data": map[string]string{"challenge_id": id, "message": "Check your email for a sign-in code."}})
}
func (h *Identity) Verify(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	if h.Auth.limited(w, r) {
		return
	}
	var b struct {
		ID   string `json:"challenge_id"`
		Code string `json:"code"`
	}
	if !bodyJSON(w, r, &b) {
		return
	}
	user, token, err := h.Codes.Verify(r.Context(), b.ID, b.Code)
	if err != nil {
		h.Auth.failure(w, r, err)
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	http.SetCookie(w, h.Auth.Service.Config.Cookie(token, time.Now()))
	writeJSON(w, 200, map[string]any{"data": user})
}
func safeIdentityNext(raw string) string {
	if len(raw) > 2048 || !strings.HasPrefix(raw, "/") || strings.HasPrefix(raw, "//") || strings.ContainsAny(raw, "\\\r\n") {
		return "/"
	}
	if strings.HasPrefix(raw, "/login") || strings.HasPrefix(raw, "/register") {
		return "/"
	}
	return raw
}
func (h *Identity) Start(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	if h.Provider == nil {
		writeError(w, 503, "method_unavailable", "External sign-in is not configured.", "")
		return
	}
	state, nonce, binding, verifier := auth.RandomToken(), auth.RandomToken(), auth.RandomToken(), oauth2.GenerateVerifier()
	next := safeIdentityNext(r.URL.Query().Get("next"))
	_, err := h.Codes.DB.Exec(r.Context(), `WITH cleanup AS (DELETE FROM oauth_flows WHERE expires_at<=now()) INSERT INTO oauth_flows(state_hash,nonce,verifier,binding_hash,next_path,expires_at) VALUES($1,$2,$3,$4,$5,now()+interval '10 minutes')`, auth.Digest(state), nonce, verifier, auth.Digest(binding), next)
	if err != nil {
		writeError(w, 500, "internal_error", "Unable to start sign-in.", "")
		return
	}
	http.SetCookie(w, &http.Cookie{Name: "keyspace_oidc", Value: binding, Path: "/api/auth/oidc", HttpOnly: true, Secure: h.Auth.Service.Config.Secure, SameSite: http.SameSiteLaxMode, MaxAge: 600})
	w.Header().Set("Cache-Control", "no-store")
	http.Redirect(w, r, h.OAuth.AuthCodeURL(state, oidc.Nonce(nonce), oauth2.S256ChallengeOption(verifier)), 302)
}
func (h *Identity) Callback(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	fail := func() { http.Redirect(w, r, h.Frontend+"/login?identity_error=1", 303) }
	if h.Provider == nil {
		fail()
		return
	}
	binding, err := r.Cookie("keyspace_oidc")
	if err != nil {
		fail()
		return
	}
	state := r.URL.Query().Get("state")
	if len(state) != 64 {
		fail()
		return
	}
	var nonce, verifier, wantBinding, next string
	err = h.Codes.DB.QueryRow(r.Context(), `DELETE FROM oauth_flows WHERE state_hash=$1 AND expires_at>now() RETURNING nonce,verifier,binding_hash,next_path`, auth.Digest(state)).Scan(&nonce, &verifier, &wantBinding, &next)
	if err != nil || subtle.ConstantTimeCompare([]byte(wantBinding), []byte(auth.Digest(binding.Value))) != 1 {
		fail()
		return
	}
	http.SetCookie(w, &http.Cookie{Name: "keyspace_oidc", Value: "", Path: "/api/auth/oidc", HttpOnly: true, Secure: h.Auth.Service.Config.Secure, SameSite: http.SameSiteLaxMode, MaxAge: -1})
	token, err := h.OAuth.Exchange(r.Context(), r.URL.Query().Get("code"), oauth2.VerifierOption(verifier))
	if err != nil {
		fail()
		return
	}
	raw, ok := token.Extra("id_token").(string)
	if !ok {
		fail()
		return
	}
	verified, err := h.Provider.Verifier(&oidc.Config{ClientID: h.OAuth.ClientID}).Verify(r.Context(), raw)
	if err != nil || verified.Nonce != nonce {
		fail()
		return
	}
	var claims struct {
		Email         string `json:"email"`
		EmailVerified bool   `json:"email_verified"`
	}
	if err = verified.Claims(&claims); err != nil || !claims.EmailVerified || verified.Subject == "" {
		fail()
		return
	}
	email := strings.ToLower(strings.TrimSpace(claims.Email))
	if _, err = auth.Validate(auth.Credentials{Email: email, Password: "identity-validation-only"}); err != nil {
		fail()
		return
	}
	_, session, err := h.Codes.ExternalUser(r.Context(), verified.Issuer, verified.Subject, email)
	if err != nil {
		fail()
		return
	}
	http.SetCookie(w, h.Auth.Service.Config.Cookie(session, time.Now()))
	destination, err := url.Parse(h.Frontend + safeIdentityNext(next))
	if err != nil {
		fail()
		return
	}
	http.Redirect(w, r, destination.String(), 303)
}
