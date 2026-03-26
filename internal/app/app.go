package app

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/go-chi/chi/v5"
	"golang.org/x/crypto/bcrypt"

	"xray-reality-solo-vpn/internal/config"
	"xray-reality-solo-vpn/internal/runtime"
	"xray-reality-solo-vpn/internal/security"
	setupflow "xray-reality-solo-vpn/internal/setup"
	"xray-reality-solo-vpn/internal/store"
	"xray-reality-solo-vpn/internal/web"
)

type App struct {
	cfg         *config.Config
	store       *store.Store
	runtime     *runtime.Manager
	setup       *setupflow.Manager
	router      http.Handler
	rateLimiter *limiter
	pollOnce    sync.Once
}

type limiter struct {
	mu      sync.Mutex
	buckets map[string]bucket
}

type bucket struct {
	Count   int
	ResetAt time.Time
}

type jsonError struct {
	Error string `json:"error"`
}

func New(cfg *config.Config) (*App, error) {
	s, err := store.Open(cfg)
	if err != nil {
		return nil, err
	}

	rt := runtime.NewManager(cfg, s)
	app := &App{
		cfg:         cfg,
		store:       s,
		runtime:     rt,
		setup:       setupflow.NewManager(cfg),
		rateLimiter: &limiter{buckets: map[string]bucket{}},
	}

	if err := rt.WriteRuntimeArtifacts(context.Background()); err != nil {
		slog.Warn("write initial runtime artifacts", "error", err)
	}

	app.router = app.buildRouter()
	return app, nil
}

func (a *App) Close() error {
	return a.store.Close()
}

func (a *App) Router() http.Handler {
	return a.router
}

func (a *App) StartPolling() {
	a.pollOnce.Do(func() {
		go func() {
			ticker := time.NewTicker(a.cfg.RuntimeRefreshPeriod)
			defer ticker.Stop()
			for {
				_ = a.store.CleanupExpiredSessions(context.Background())
				a.runtime.RefreshTrafficStats(context.Background())
				<-ticker.C
			}
		}()
	})
}

func (a *App) WriteRuntimeArtifacts(ctx context.Context) error {
	return a.runtime.WriteRuntimeArtifacts(ctx)
}

func (a *App) IssueSetupTicket(ctx context.Context, ttl time.Duration) (string, error) {
	initialized, err := a.store.IsInitialized(ctx)
	if err != nil {
		return "", err
	}
	if initialized {
		return "", errors.New("already initialized")
	}
	if ttl > 0 && ttl != a.cfg.SetupTicketTTL {
		a.cfg.SetupTicketTTL = ttl
		a.setup = setupflow.NewManager(a.cfg)
	}
	ticket, err := a.setup.Create()
	if err != nil {
		return "", err
	}
	return ticket.Token, nil
}

func (a *App) buildRouter() http.Handler {
	router := chi.NewRouter()
	router.Use(security.SecurityHeaders)
	router.Use(a.logMiddleware)
	router.Use(a.recoverMiddleware)

	router.Get("/_/setup/{token}", a.handleActivateSetup)

	router.Route("/api", func(r chi.Router) {
		r.Get("/setup/status", a.handleSetupStatus)
		r.Post("/setup/init", a.handleSetupInit)
		r.Get("/bootstrap/status", a.handleSetupStatus)
		r.Post("/bootstrap/init", a.handleSetupInit)

		r.Post("/auth/login", a.handleLogin)
		r.Post("/auth/logout", a.handleLogout)
		r.Get("/session", a.handleSession)
		r.Get("/subscriptions/{token}/mihomo.yaml", a.handleSubscription)

		r.Group(func(authed chi.Router) {
			authed.Use(a.authMiddleware)
			authed.Get("/dashboard", a.handleDashboard)
			authed.Post("/services/sync", a.handleSyncService)
			authed.Get("/clients", a.handleListClients)
			authed.Post("/clients", a.handleCreateClient)
			authed.Patch("/clients/{id}", a.handleUpdateClient)
			authed.Delete("/clients/{id}", a.handleDeleteClient)
			authed.Get("/clients/{id}/share", a.handleShareClient)
			authed.Post("/auth/password", a.handleChangePassword)
			authed.Patch("/account/password", a.handleChangePassword)
		})
	})

	router.Mount("/", web.SPAHandler(a.cfg.WebDistDir))
	return router
}

func (a *App) handleSetupStatus(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	initialized, err := a.store.IsInitialized(ctx)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	summary, err := a.runtimeSummary(ctx)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	setupAuthorized, setupExpiresAt, err := a.setupAuthorization(r)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	response := map[string]any{
		"initialized":     initialized,
		"setupAuthorized": setupAuthorized && !initialized,
		"setupLocked":     !initialized && !setupAuthorized,
		"setupExpiresAt":  setupExpiresAt,
		"missingRuntime":  a.cfg.MissingRuntime(),
	}
	for key, value := range summary {
		response[key] = value
	}
	writeJSON(w, http.StatusOK, response)
}

func (a *App) handleActivateSetup(w http.ResponseWriter, r *http.Request) {
	initialized, err := a.store.IsInitialized(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if initialized {
		http.Redirect(w, r, "/login?setup=completed", http.StatusFound)
		return
	}

	token := chi.URLParam(r, "token")
	ticket, valid, err := a.setup.Validate(r.Context(), token)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if ticket == nil || !valid {
		http.Redirect(w, r, "/setup?setup=invalid", http.StatusFound)
		return
	}

	expiresAt, err := time.Parse(time.RFC3339, ticket.ExpiresAt)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	ttl := time.Until(expiresAt)
	if ttl < time.Minute {
		ttl = time.Minute
	}
	http.SetCookie(w, security.SetupAuthCookie(security.SessionHash(ticket.Token), security.IsSecureRequest(r, a.cfg.TrustProxy), ttl))
	http.Redirect(w, r, "/setup", http.StatusFound)
}

func (a *App) handleSetupInit(w http.ResponseWriter, r *http.Request) {
	if retryAfter, limited := a.consumeRateLimit(r, "setup", 5); limited {
		w.Header().Set("Retry-After", strconv.Itoa(retryAfter))
		writeError(w, http.StatusTooManyRequests, "Too many attempts, please try again later")
		return
	}

	var payload struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := a.decodeJSON(r, &payload); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid payload")
		return
	}
	if err := security.ValidateUsername(payload.Username); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := security.ValidatePassword(payload.Password); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	ctx := r.Context()
	initialized, err := a.store.IsInitialized(ctx)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if initialized {
		writeError(w, http.StatusConflict, "Setup already completed")
		return
	}

	authorized, _, err := a.setupAuthorization(r)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if !authorized {
		writeError(w, http.StatusLocked, "Setup is locked")
		return
	}

	passwordHash, err := hashPassword(payload.Password)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	admin, err := a.store.CreateAdmin(ctx, payload.Username, passwordHash)
	if err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "unique") {
			writeError(w, http.StatusConflict, "Setup already completed")
			return
		}
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	_ = a.setup.Consume()
	http.SetCookie(w, security.ClearSetupAuthCookie(security.IsSecureRequest(r, a.cfg.TrustProxy)))

	cookieToken, err := a.createSession(ctx, admin.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	http.SetCookie(w, security.SessionCookie(cookieToken, security.IsSecureRequest(r, a.cfg.TrustProxy), a.cfg.AccessTokenTTL))

	restart := a.runtime.RestartXray(ctx)
	writeJSON(w, http.StatusOK, map[string]any{
		"ok":      true,
		"restart": restart,
		"admin": map[string]string{
			"username": admin.Username,
		},
	})
}

func (a *App) handleLogin(w http.ResponseWriter, r *http.Request) {
	if retryAfter, limited := a.consumeRateLimit(r, "login", 10); limited {
		w.Header().Set("Retry-After", strconv.Itoa(retryAfter))
		writeError(w, http.StatusTooManyRequests, "Too many attempts, please try again later")
		return
	}

	var payload struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := a.decodeJSON(r, &payload); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid payload")
		return
	}
	if err := security.ValidateUsername(payload.Username); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid payload")
		return
	}
	if err := security.ValidatePassword(payload.Password); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid payload")
		return
	}

	admin, err := a.store.FindAdminByUsername(r.Context(), payload.Username)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if admin == nil || !checkPassword(payload.Password, admin.PasswordHash) {
		writeError(w, http.StatusUnauthorized, "Invalid username or password")
		return
	}

	cookieToken, err := a.createSession(r.Context(), admin.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	http.SetCookie(w, security.SessionCookie(cookieToken, security.IsSecureRequest(r, a.cfg.TrustProxy), a.cfg.AccessTokenTTL))

	writeJSON(w, http.StatusOK, map[string]any{
		"ok": true,
		"admin": map[string]string{
			"username": admin.Username,
		},
	})
}

func (a *App) handleLogout(w http.ResponseWriter, r *http.Request) {
	if token := sessionTokenFromRequest(r); token != "" {
		_ = a.store.DeleteSessionByHash(r.Context(), security.SessionHash(token))
	}
	http.SetCookie(w, security.ClearSessionCookie(security.IsSecureRequest(r, a.cfg.TrustProxy)))
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func (a *App) handleSession(w http.ResponseWriter, r *http.Request) {
	token := sessionTokenFromRequest(r)
	if token == "" {
		writeJSON(w, http.StatusOK, map[string]any{"authenticated": false})
		return
	}
	session, err := a.store.GetSessionByHash(r.Context(), security.SessionHash(token))
	if err != nil || session == nil {
		writeJSON(w, http.StatusOK, map[string]any{"authenticated": false})
		return
	}
	expiresAt, err := time.Parse(time.RFC3339, session.ExpiresAt)
	if err != nil || time.Now().UTC().After(expiresAt) {
		_ = a.store.DeleteSessionByHash(r.Context(), session.TokenHash)
		writeJSON(w, http.StatusOK, map[string]any{"authenticated": false})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"authenticated": true,
		"admin": map[string]string{
			"username": session.Username,
		},
	})
}

func (a *App) handleDashboard(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	status := a.runtime.ReadServiceStatus(ctx)
	summary, err := a.runtimeSummary(ctx)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	metrics, err := a.store.GetServiceMetrics(ctx)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"status":  status,
		"summary": summary,
		"traffic": map[string]any{
			"totalRxBytes": metrics.TotalRXBytes,
			"totalTxBytes": metrics.TotalTXBytes,
			"totalRxHuman": runtime.BytesToHuman(metrics.TotalRXBytes),
			"totalTxHuman": runtime.BytesToHuman(metrics.TotalTXBytes),
		},
	})
}

func (a *App) handleSyncService(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	restart := a.runtime.RestartXray(ctx)
	status := a.runtime.RefreshTrafficStats(ctx)
	writeJSON(w, http.StatusOK, map[string]any{
		"ok":      restart.State != "error",
		"restart": restart,
		"status":  status,
	})
}

func (a *App) handleListClients(w http.ResponseWriter, r *http.Request) {
	clients, err := a.store.ListClients(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	payload := make([]map[string]any, 0, len(clients))
	for _, client := range clients {
		payload = append(payload, a.clientPayload(client))
	}
	writeJSON(w, http.StatusOK, map[string]any{"clients": payload})
}

func (a *App) handleCreateClient(w http.ResponseWriter, r *http.Request) {
	var payload struct {
		Name string `json:"name"`
	}
	if err := a.decodeJSON(r, &payload); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid payload")
		return
	}
	name := security.NormalizeClientName(payload.Name)
	if err := security.ValidateClientName(name); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	client, err := a.store.CreateClient(r.Context(), name)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	restart := a.runtime.RestartXray(r.Context())
	writeJSON(w, http.StatusCreated, map[string]any{
		"client":  a.clientPayload(*client),
		"restart": restart,
	})
}

func (a *App) handleUpdateClient(w http.ResponseWriter, r *http.Request) {
	clientID, err := parseID(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid client id")
		return
	}

	var payload struct {
		Name        *string `json:"name"`
		Enabled     *bool   `json:"enabled"`
		RotateToken bool    `json:"rotateToken"`
	}
	if err := a.decodeJSON(r, &payload); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid payload")
		return
	}

	var nextName *string
	if payload.Name != nil {
		normalized := security.NormalizeClientName(*payload.Name)
		if err := security.ValidateClientName(normalized); err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		nextName = &normalized
	}

	client, err := a.store.UpdateClient(r.Context(), clientID, store.ClientUpdate{
		Name:        nextName,
		Enabled:     payload.Enabled,
		RotateToken: payload.RotateToken,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if client == nil {
		writeError(w, http.StatusNotFound, "Client not found")
		return
	}

	restart := a.runtime.RestartXray(r.Context())
	writeJSON(w, http.StatusOK, map[string]any{
		"client":  a.clientPayload(*client),
		"restart": restart,
	})
}

func (a *App) handleDeleteClient(w http.ResponseWriter, r *http.Request) {
	clientID, err := parseID(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid client id")
		return
	}

	client, err := a.store.GetClientByID(r.Context(), clientID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if client == nil {
		writeError(w, http.StatusNotFound, "Client not found")
		return
	}

	if err := a.store.DeleteClient(r.Context(), clientID); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	restart := a.runtime.RestartXray(r.Context())
	writeJSON(w, http.StatusOK, map[string]any{
		"ok":      true,
		"restart": restart,
	})
}

func (a *App) handleShareClient(w http.ResponseWriter, r *http.Request) {
	clientID, err := parseID(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid client id")
		return
	}
	client, err := a.store.GetClientByID(r.Context(), clientID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if client == nil {
		writeError(w, http.StatusNotFound, "Client not found")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"shareLink": a.runtime.BuildClientShareLink(*client),
		"mihomo":    a.runtime.BuildMihomoConfig(*client),
	})
}

func (a *App) handleSubscription(w http.ResponseWriter, r *http.Request) {
	client, err := a.store.GetClientByToken(r.Context(), chi.URLParam(r, "token"))
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if client == nil || !client.Enabled {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		http.Error(w, "Not found", http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "text/yaml; charset=utf-8")
	_, _ = w.Write([]byte(a.runtime.BuildMihomoConfig(*client)))
}

func (a *App) handleChangePassword(w http.ResponseWriter, r *http.Request) {
	session, ok := sessionFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	var payload struct {
		CurrentPassword string `json:"currentPassword"`
		NewPassword     string `json:"newPassword"`
	}
	if err := a.decodeJSON(r, &payload); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid payload")
		return
	}
	if err := security.ValidatePassword(payload.CurrentPassword); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid payload")
		return
	}
	if err := security.ValidatePassword(payload.NewPassword); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	admin, err := a.store.GetAdminByID(r.Context(), session.AdminID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if admin == nil || !checkPassword(payload.CurrentPassword, admin.PasswordHash) {
		writeError(w, http.StatusUnauthorized, "Current password is incorrect")
		return
	}

	passwordHash, err := hashPassword(payload.NewPassword)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if err := a.store.UpdateAdminPassword(r.Context(), admin.ID, passwordHash); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if err := a.store.DeleteSessionsByAdmin(r.Context(), admin.ID); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	cookieToken, err := a.createSession(r.Context(), admin.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	http.SetCookie(w, security.SessionCookie(cookieToken, security.IsSecureRequest(r, a.cfg.TrustProxy), a.cfg.AccessTokenTTL))
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func (a *App) authMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token := sessionTokenFromRequest(r)
		if token == "" {
			writeError(w, http.StatusUnauthorized, "Unauthorized")
			return
		}

		session, err := a.store.GetSessionByHash(r.Context(), security.SessionHash(token))
		if err != nil || session == nil {
			writeError(w, http.StatusUnauthorized, "Unauthorized")
			return
		}
		expiresAt, err := time.Parse(time.RFC3339, session.ExpiresAt)
		if err != nil || time.Now().UTC().After(expiresAt) {
			_ = a.store.DeleteSessionByHash(r.Context(), session.TokenHash)
			writeError(w, http.StatusUnauthorized, "Unauthorized")
			return
		}

		_ = a.store.TouchSession(r.Context(), session.TokenHash)
		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), sessionContextKey{}, *session)))
	})
}

func (a *App) runtimeSummary(ctx context.Context) (map[string]any, error) {
	clients, err := a.store.ListClients(ctx)
	if err != nil {
		return nil, err
	}
	metrics, err := a.store.GetServiceMetrics(ctx)
	if err != nil {
		return nil, err
	}

	active := 0
	for _, client := range clients {
		if client.Enabled {
			active++
		}
	}
	settings := a.runtime.Settings()
	return map[string]any{
		"panelBaseUrl":      settings.PanelBaseURL,
		"panelDomain":       settings.PanelDomain,
		"lineDomain":        settings.LineDomain,
		"lineServerAddress": settings.LineServerAddress,
		"xrayTarget":        settings.XrayTarget,
		"xrayServerNames":   settings.XrayServerNames,
		"serviceState":      metrics.ServiceState,
		"totalRxBytes":      metrics.TotalRXBytes,
		"totalTxBytes":      metrics.TotalTXBytes,
		"totalRxHuman":      runtime.BytesToHuman(metrics.TotalRXBytes),
		"totalTxHuman":      runtime.BytesToHuman(metrics.TotalTXBytes),
		"refreshIntervalMs": int(a.cfg.RuntimeRefreshPeriod / time.Millisecond),
		"clientCount":       len(clients),
		"activeClientCount": active,
	}, nil
}

func (a *App) clientPayload(client store.Client) map[string]any {
	subscriptionURL := fmt.Sprintf("%s/api/subscriptions/%s/mihomo.yaml", strings.TrimRight(a.cfg.PanelBaseURL, "/"), client.SubscriptionToken)
	return map[string]any{
		"id":                    client.ID,
		"name":                  client.Name,
		"slug":                  client.Slug,
		"uuid":                  client.UUID,
		"email_tag":             client.EmailTag,
		"short_id":              client.ShortID,
		"subscription_token":    client.SubscriptionToken,
		"enabled":               client.Enabled,
		"last_seen_at":          client.LastSeenAt,
		"rx_bytes":              client.RXBytes,
		"tx_bytes":              client.TXBytes,
		"rx_bps":                client.RXBps,
		"tx_bps":                client.TXBps,
		"created_at":            client.CreatedAt,
		"updated_at":            client.UpdatedAt,
		"shareLink":             a.runtime.BuildClientShareLink(client),
		"mihomoSubscriptionUrl": subscriptionURL,
		"rxHuman":               runtime.BytesToHuman(client.RXBytes),
		"txHuman":               runtime.BytesToHuman(client.TXBytes),
		"rxBpsHuman":            runtime.BytesToHuman(client.RXBps) + "/s",
		"txBpsHuman":            runtime.BytesToHuman(client.TXBps) + "/s",
	}
}

func (a *App) createSession(ctx context.Context, adminID int64) (string, error) {
	rawToken, err := security.RandomToken(32)
	if err != nil {
		return "", err
	}
	if err := a.store.CreateSession(ctx, adminID, security.SessionHash(rawToken), time.Now().UTC().Add(a.cfg.AccessTokenTTL)); err != nil {
		return "", err
	}
	return rawToken, nil
}

func (a *App) setupAuthorization(r *http.Request) (bool, *string, error) {
	cookie, err := r.Cookie("setup_auth")
	if err != nil {
		return false, nil, nil
	}
	ticket, err := a.setup.Load()
	if err != nil || ticket == nil {
		return false, nil, err
	}
	expiresAt := ticket.ExpiresAt
	if security.SessionHash(ticket.Token) != cookie.Value {
		return false, &expiresAt, nil
	}
	_, valid, err := a.setup.Validate(r.Context(), ticket.Token)
	return valid, &expiresAt, err
}

func (a *App) decodeJSON(r *http.Request, target any) error {
	r.Body = http.MaxBytesReader(nil, r.Body, a.cfg.MaxRequestBodyBytes)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	return decoder.Decode(target)
}

func (a *App) consumeRateLimit(r *http.Request, scope string, limit int) (retryAfter int, limited bool) {
	a.rateLimiter.mu.Lock()
	defer a.rateLimiter.mu.Unlock()

	key := scope + ":" + clientIP(r)
	now := time.Now()
	entry, ok := a.rateLimiter.buckets[key]
	if !ok || now.After(entry.ResetAt) {
		a.rateLimiter.buckets[key] = bucket{Count: 1, ResetAt: now.Add(a.cfg.RateLimitWindow)}
		return 0, false
	}
	entry.Count++
	a.rateLimiter.buckets[key] = entry
	if entry.Count > limit {
		return int(time.Until(entry.ResetAt).Seconds()), true
	}
	return 0, false
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, jsonError{Error: message})
}

func hashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), 12)
	if err != nil {
		return "", err
	}
	return string(bytes), nil
}

func checkPassword(password, hash string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) == nil
}

func parseID(value string) (int64, error) {
	id, err := strconv.ParseInt(value, 10, 64)
	if err != nil || id <= 0 {
		return 0, errors.New("invalid id")
	}
	return id, nil
}

func clientIP(r *http.Request) string {
	if host, _, err := net.SplitHostPort(r.RemoteAddr); err == nil {
		return host
	}
	return r.RemoteAddr
}

type sessionContextKey struct{}

func sessionFromContext(ctx context.Context) (store.Session, bool) {
	value, ok := ctx.Value(sessionContextKey{}).(store.Session)
	return value, ok
}

func sessionTokenFromRequest(r *http.Request) string {
	cookie, err := r.Cookie("session")
	if err != nil {
		return ""
	}
	return cookie.Value
}

func (a *App) logMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		recorder := &statusRecorder{ResponseWriter: w, statusCode: http.StatusOK}
		next.ServeHTTP(recorder, r)
		slog.Info("http request",
			"method", r.Method,
			"path", r.URL.Path,
			"status", recorder.statusCode,
			"duration", time.Since(start).String(),
		)
	})
}

func (a *App) recoverMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rec := recover(); rec != nil {
				slog.Error("panic", "error", rec)
				writeError(w, http.StatusInternalServerError, "Internal server error")
			}
		}()
		next.ServeHTTP(w, r)
	})
}

type statusRecorder struct {
	http.ResponseWriter
	statusCode int
}

func (r *statusRecorder) WriteHeader(statusCode int) {
	r.statusCode = statusCode
	r.ResponseWriter.WriteHeader(statusCode)
}
