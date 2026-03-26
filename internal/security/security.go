package security

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"net/http"
	"regexp"
	"strings"
	"time"
)

var usernamePattern = regexp.MustCompile(`^[a-zA-Z0-9_.-]+$`)

func NormalizeClientName(input string) string {
	return strings.Join(strings.Fields(strings.TrimSpace(input)), " ")
}

func ValidateClientName(name string) error {
	if len(name) < 2 || len(name) > 48 {
		return errors.New("client name must be 2-48 characters")
	}
	for _, r := range name {
		if r < 32 || r == 127 {
			return errors.New("client name contains control characters")
		}
	}
	return nil
}

func IsValidClientName(name string) bool {
	return ValidateClientName(name) == nil
}

func ValidateUsername(username string) error {
	username = strings.TrimSpace(username)
	if len(username) < 3 || len(username) > 32 {
		return errors.New("username must be 3-32 characters")
	}
	if !usernamePattern.MatchString(username) {
		return errors.New("username may only use letters, numbers, dot, underscore, or dash")
	}
	return nil
}

func ValidatePassword(password string) error {
	if len(password) < 8 || len(password) > 128 {
		return errors.New("password must be 8-128 characters")
	}
	return nil
}

func YAMLQuote(value string) string {
	return "'" + strings.ReplaceAll(value, "'", "''") + "'"
}

func SessionCookie(token string, secure bool, ttl time.Duration) *http.Cookie {
	return &http.Cookie{
		Name:     "session",
		Value:    token,
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
		Secure:   secure,
		Path:     "/",
		MaxAge:   int(ttl.Seconds()),
	}
}

func ClearSessionCookie(secure bool) *http.Cookie {
	return &http.Cookie{
		Name:     "session",
		Value:    "",
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
		Secure:   secure,
		Path:     "/",
		MaxAge:   -1,
	}
}

func SetupAuthCookie(tokenHash string, secure bool, ttl time.Duration) *http.Cookie {
	return &http.Cookie{
		Name:     "setup_auth",
		Value:    tokenHash,
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
		Secure:   secure,
		Path:     "/",
		MaxAge:   int(ttl.Seconds()),
	}
}

func ClearSetupAuthCookie(secure bool) *http.Cookie {
	return &http.Cookie{
		Name:     "setup_auth",
		Value:    "",
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
		Secure:   secure,
		Path:     "/",
		MaxAge:   -1,
	}
}

func IsSecureRequest(r *http.Request, trustProxy bool) bool {
	if r.TLS != nil {
		return true
	}
	if trustProxy {
		forwardedProto := r.Header.Get("X-Forwarded-Proto")
		if forwardedProto != "" {
			return strings.TrimSpace(strings.Split(forwardedProto, ",")[0]) == "https"
		}
	}
	return false
}

func SessionHash(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

func RandomHex(length int) (string, error) {
	if length <= 0 {
		return "", errors.New("length must be positive")
	}
	buf := make([]byte, (length+1)/2)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf)[:length], nil
}

func RandomToken(bytes int) (string, error) {
	buf := make([]byte, bytes)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf), nil
}

func NewUUID() (string, error) {
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	buf[6] = (buf[6] & 0x0f) | 0x40
	buf[8] = (buf[8] & 0x3f) | 0x80

	hexText := hex.EncodeToString(buf)
	return strings.Join([]string{
		hexText[0:8],
		hexText[8:12],
		hexText[12:16],
		hexText[16:20],
		hexText[20:32],
	}, "-"), nil
}

func SecurityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Referrer-Policy", "no-referrer")
		w.Header().Set("Permissions-Policy", "geolocation=(), microphone=(), camera=()")
		w.Header().Set(
			"Content-Security-Policy",
			"default-src 'self'; connect-src 'self'; img-src 'self' data:; font-src 'self'; style-src 'self' 'unsafe-inline'; script-src 'self'; object-src 'none'; base-uri 'self'; form-action 'self'; frame-ancestors 'none'",
		)
		next.ServeHTTP(w, r)
	})
}
