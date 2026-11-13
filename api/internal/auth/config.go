package auth

import (
	"errors"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

const CookieName = "marketplace_session"
const SessionTTL = 24 * time.Hour

type Config struct {
	Secret   []byte
	Secure   bool
	SameSite http.SameSite
	Origins  map[string]bool
}

func LoadConfig(getenv func(string) string) (Config, error) {
	c := Config{Secret: []byte(getenv("JWT_SECRET")), Secure: true, SameSite: http.SameSiteLaxMode, Origins: map[string]bool{}}
	if len(c.Secret) < 32 {
		return c, errors.New("JWT_SECRET must contain at least 32 bytes")
	}
	if v := getenv("COOKIE_SECURE"); v != "" {
		b, err := strconv.ParseBool(v)
		if err != nil {
			return c, errors.New("COOKIE_SECURE must be true or false")
		}
		c.Secure = b
	}
	switch strings.ToLower(getenv("COOKIE_SAMESITE")) {
	case "", "lax":
	case "strict":
		c.SameSite = http.SameSiteStrictMode
	case "none":
		c.SameSite = http.SameSiteNoneMode
	default:
		return c, errors.New("COOKIE_SAMESITE must be lax, strict, or none")
	}
	if c.SameSite == http.SameSiteNoneMode && !c.Secure {
		return c, errors.New("SameSite=None requires COOKIE_SECURE=true")
	}
	for _, origin := range strings.Split(getenv("AUTH_ALLOWED_ORIGINS"), ",") {
		origin = strings.TrimSpace(origin)
		if origin == "" {
			continue
		}
		u, err := url.Parse(origin)
		if err != nil || u.Host == "" || (u.Scheme != "http" && u.Scheme != "https") || u.User != nil || u.Path != "" || u.RawQuery != "" || u.Fragment != "" {
			return c, errors.New("AUTH_ALLOWED_ORIGINS must contain exact http(s) origins without paths or wildcards")
		}
		if strings.Contains(u.Host, "*") {
			return c, errors.New("wildcard origins are not allowed")
		}
		c.Origins[origin] = true
	}
	if len(c.Origins) == 0 {
		return c, errors.New("AUTH_ALLOWED_ORIGINS must contain at least one frontend origin")
	}
	return c, nil
}
func (c Config) Cookie(value string, now time.Time) *http.Cookie {
	return &http.Cookie{Name: CookieName, Value: value, Path: "/", HttpOnly: true, Secure: c.Secure, SameSite: c.SameSite, MaxAge: int(SessionTTL.Seconds()), Expires: now.Add(SessionTTL)}
}
func (c Config) ClearCookie() *http.Cookie {
	cookie := c.Cookie("", time.Unix(0, 0))
	cookie.MaxAge = -1
	cookie.Expires = time.Unix(1, 0)
	return cookie
}
