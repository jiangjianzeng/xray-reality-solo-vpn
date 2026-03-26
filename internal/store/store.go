package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"xray-reality-solo-vpn/internal/config"
	"xray-reality-solo-vpn/internal/security"

	_ "modernc.org/sqlite"
)

type Admin struct {
	ID           int64  `json:"id"`
	Username     string `json:"username"`
	PasswordHash string
	CreatedAt    string `json:"created_at"`
}

type Session struct {
	ID         int64
	AdminID    int64
	Username   string
	TokenHash  string
	ExpiresAt  string
	CreatedAt  string
	LastSeenAt string
}

type Client struct {
	ID                int64   `json:"id"`
	Name              string  `json:"name"`
	Slug              string  `json:"slug"`
	UUID              string  `json:"uuid"`
	EmailTag          string  `json:"email_tag"`
	ShortID           string  `json:"short_id"`
	SubscriptionToken string  `json:"subscription_token"`
	Enabled           bool    `json:"enabled"`
	LastSeenAt        *string `json:"last_seen_at"`
	RXBytes           int64   `json:"rx_bytes"`
	TXBytes           int64   `json:"tx_bytes"`
	RXBps             int64   `json:"rx_bps"`
	TXBps             int64   `json:"tx_bps"`
	CreatedAt         string  `json:"created_at"`
	UpdatedAt         string  `json:"updated_at"`
}

type ServiceMetrics struct {
	ServiceState string `json:"service_state"`
	TotalRXBytes int64  `json:"total_rx_bytes"`
	TotalTXBytes int64  `json:"total_tx_bytes"`
	UpdatedAt    string `json:"updated_at"`
}

type ClientUpdate struct {
	Name        *string
	Enabled     *bool
	RotateToken bool
}

type ClientTrafficUpdate struct {
	RXBytes    int64
	TXBytes    int64
	RXBps      int64
	TXBps      int64
	LastSeenAt *string
}

type Store struct {
	db         *sql.DB
	lineDomain string
}

func Open(cfg *config.Config) (*Store, error) {
	dsn := filepath.Join(cfg.DataDir, "manager.db")
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, err
	}

	if _, err := db.Exec(`
		PRAGMA journal_mode = WAL;
		PRAGMA busy_timeout = 5000;
		PRAGMA foreign_keys = ON;
	`); err != nil {
		return nil, err
	}

	store := &Store{db: db, lineDomain: cfg.LineDomain}
	if err := store.migrate(); err != nil {
		return nil, err
	}
	return store, nil
}

func (s *Store) Close() error {
	return s.db.Close()
}

func (s *Store) migrate() error {
	if err := s.maybeMigrateLegacyAdmins(); err != nil {
		return err
	}

	_, err := s.db.Exec(`
		CREATE TABLE IF NOT EXISTS admins (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			username TEXT NOT NULL UNIQUE,
			password_hash TEXT NOT NULL,
			created_at TEXT NOT NULL
		);

		CREATE TABLE IF NOT EXISTS admin_sessions (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			admin_id INTEGER NOT NULL,
			token_hash TEXT NOT NULL UNIQUE,
			created_at TEXT NOT NULL,
			last_seen_at TEXT NOT NULL,
			expires_at TEXT NOT NULL,
			FOREIGN KEY(admin_id) REFERENCES admins(id) ON DELETE CASCADE
		);

		CREATE TABLE IF NOT EXISTS clients (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL,
			slug TEXT NOT NULL UNIQUE,
			uuid TEXT NOT NULL UNIQUE,
			email_tag TEXT NOT NULL UNIQUE,
			short_id TEXT NOT NULL UNIQUE,
			subscription_token TEXT NOT NULL UNIQUE,
			enabled INTEGER NOT NULL DEFAULT 1,
			last_seen_at TEXT,
			rx_bytes INTEGER NOT NULL DEFAULT 0,
			tx_bytes INTEGER NOT NULL DEFAULT 0,
			rx_bps INTEGER NOT NULL DEFAULT 0,
			tx_bps INTEGER NOT NULL DEFAULT 0,
			created_at TEXT NOT NULL,
			updated_at TEXT NOT NULL
		);

		CREATE TABLE IF NOT EXISTS service_metrics (
			id INTEGER PRIMARY KEY CHECK (id = 1),
			service_state TEXT NOT NULL DEFAULT 'unknown',
			total_rx_bytes INTEGER NOT NULL DEFAULT 0,
			total_tx_bytes INTEGER NOT NULL DEFAULT 0,
			updated_at TEXT NOT NULL DEFAULT ''
		);
	`)
	if err != nil {
		return err
	}

	_, err = s.db.Exec(`
		INSERT INTO service_metrics (id, service_state, total_rx_bytes, total_tx_bytes, updated_at)
		VALUES (1, 'unknown', 0, 0, '')
		ON CONFLICT(id) DO NOTHING
	`)
	return err
}

func (s *Store) maybeMigrateLegacyAdmins() error {
	rows, err := s.db.Query(`PRAGMA table_info(admins)`)
	if err != nil {
		if strings.Contains(err.Error(), "no such table") {
			return nil
		}
		return err
	}
	defer rows.Close()

	hasUsername := false
	hasEmail := false
	for rows.Next() {
		var (
			cid        int
			name       string
			typeName   string
			notNull    int
			defaultVal sql.NullString
			primaryKey int
		)
		if err := rows.Scan(&cid, &name, &typeName, &notNull, &defaultVal, &primaryKey); err != nil {
			return err
		}
		if name == "username" {
			hasUsername = true
		}
		if name == "email" {
			hasEmail = true
		}
	}
	if hasUsername || !hasEmail {
		return nil
	}

	_, err = s.db.Exec(`
		ALTER TABLE admins RENAME TO admins_legacy;
		CREATE TABLE admins (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			username TEXT NOT NULL UNIQUE,
			password_hash TEXT NOT NULL,
			created_at TEXT NOT NULL
		);
		INSERT INTO admins (id, username, password_hash, created_at)
		SELECT id, email, password_hash, created_at FROM admins_legacy;
		DROP TABLE admins_legacy;
	`)
	return err
}

func (s *Store) IsInitialized(ctx context.Context) (bool, error) {
	var count int
	if err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM admins`).Scan(&count); err != nil {
		return false, err
	}
	return count > 0, nil
}

func (s *Store) CreateAdmin(ctx context.Context, username, passwordHash string) (*Admin, error) {
	now := nowISO()
	result, err := s.db.ExecContext(ctx, `
		INSERT INTO admins (username, password_hash, created_at)
		VALUES (?, ?, ?)
	`, strings.ToLower(username), passwordHash, now)
	if err != nil {
		return nil, err
	}
	id, err := result.LastInsertId()
	if err != nil {
		return nil, err
	}
	return s.GetAdminByID(ctx, id)
}

func (s *Store) GetAdminByID(ctx context.Context, id int64) (*Admin, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT id, username, password_hash, created_at
		FROM admins
		WHERE id = ?
	`, id)
	return scanAdmin(row)
}

func (s *Store) FindAdminByUsername(ctx context.Context, username string) (*Admin, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT id, username, password_hash, created_at
		FROM admins
		WHERE username = ?
	`, strings.ToLower(strings.TrimSpace(username)))
	return scanAdmin(row)
}

func (s *Store) UpdateAdminPassword(ctx context.Context, adminID int64, passwordHash string) error {
	_, err := s.db.ExecContext(ctx, `
		UPDATE admins
		SET password_hash = ?
		WHERE id = ?
	`, passwordHash, adminID)
	return err
}

func (s *Store) DeleteSessionsByAdmin(ctx context.Context, adminID int64) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM admin_sessions WHERE admin_id = ?`, adminID)
	return err
}

func (s *Store) CreateSession(ctx context.Context, adminID int64, tokenHash string, expiresAt time.Time) error {
	now := nowISO()
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO admin_sessions (admin_id, token_hash, created_at, last_seen_at, expires_at)
		VALUES (?, ?, ?, ?, ?)
	`, adminID, tokenHash, now, now, expiresAt.UTC().Format(time.RFC3339))
	return err
}

func (s *Store) GetSessionByHash(ctx context.Context, tokenHash string) (*Session, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT
			s.id,
			s.admin_id,
			a.username,
			s.token_hash,
			s.expires_at,
			s.created_at,
			s.last_seen_at
		FROM admin_sessions s
		INNER JOIN admins a ON a.id = s.admin_id
		WHERE s.token_hash = ?
	`, tokenHash)
	session := &Session{}
	if err := row.Scan(
		&session.ID,
		&session.AdminID,
		&session.Username,
		&session.TokenHash,
		&session.ExpiresAt,
		&session.CreatedAt,
		&session.LastSeenAt,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return session, nil
}

func (s *Store) TouchSession(ctx context.Context, tokenHash string) error {
	_, err := s.db.ExecContext(ctx, `
		UPDATE admin_sessions
		SET last_seen_at = ?
		WHERE token_hash = ?
	`, nowISO(), tokenHash)
	return err
}

func (s *Store) DeleteSessionByHash(ctx context.Context, tokenHash string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM admin_sessions WHERE token_hash = ?`, tokenHash)
	return err
}

func (s *Store) CleanupExpiredSessions(ctx context.Context) error {
	_, err := s.db.ExecContext(ctx, `
		DELETE FROM admin_sessions
		WHERE expires_at <= ?
	`, nowISO())
	return err
}

func (s *Store) ListClients(ctx context.Context) ([]Client, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT
			id,
			name,
			slug,
			uuid,
			email_tag,
			short_id,
			subscription_token,
			enabled,
			last_seen_at,
			rx_bytes,
			tx_bytes,
			rx_bps,
			tx_bps,
			created_at,
			updated_at
		FROM clients
		ORDER BY created_at DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	clients := make([]Client, 0)
	for rows.Next() {
		client, err := scanClient(rows)
		if err != nil {
			return nil, err
		}
		clients = append(clients, *client)
	}
	return clients, rows.Err()
}

func (s *Store) GetClientByID(ctx context.Context, id int64) (*Client, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT
			id,
			name,
			slug,
			uuid,
			email_tag,
			short_id,
			subscription_token,
			enabled,
			last_seen_at,
			rx_bytes,
			tx_bytes,
			rx_bps,
			tx_bps,
			created_at,
			updated_at
		FROM clients
		WHERE id = ?
	`, id)
	return scanClient(row)
}

func (s *Store) GetClientByToken(ctx context.Context, token string) (*Client, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT
			id,
			name,
			slug,
			uuid,
			email_tag,
			short_id,
			subscription_token,
			enabled,
			last_seen_at,
			rx_bytes,
			tx_bytes,
			rx_bps,
			tx_bps,
			created_at,
			updated_at
		FROM clients
		WHERE subscription_token = ?
	`, token)
	return scanClient(row)
}

func (s *Store) CreateClient(ctx context.Context, name string) (*Client, error) {
	baseSlug := slugify(name)
	slug := baseSlug
	counter := 1
	for {
		var exists int
		err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM clients WHERE slug = ?`, slug).Scan(&exists)
		if err != nil {
			return nil, err
		}
		if exists == 0 {
			break
		}
		counter++
		slug = fmt.Sprintf("%s-%d", baseSlug, counter)
	}

	uuid, err := security.NewUUID()
	if err != nil {
		return nil, err
	}
	shortID, err := security.RandomHex(16)
	if err != nil {
		return nil, err
	}
	subscriptionToken, err := security.RandomHex(48)
	if err != nil {
		return nil, err
	}

	now := nowISO()
	emailTag := fmt.Sprintf("%s@%s", slug, s.lineDomain)
	result, err := s.db.ExecContext(ctx, `
		INSERT INTO clients (
			name,
			slug,
			uuid,
			email_tag,
			short_id,
			subscription_token,
			enabled,
			created_at,
			updated_at
		) VALUES (?, ?, ?, ?, ?, ?, 1, ?, ?)
	`, name, slug, uuid, emailTag, shortID, subscriptionToken, now, now)
	if err != nil {
		return nil, err
	}
	id, err := result.LastInsertId()
	if err != nil {
		return nil, err
	}
	return s.GetClientByID(ctx, id)
}

func (s *Store) UpdateClient(ctx context.Context, id int64, update ClientUpdate) (*Client, error) {
	client, err := s.GetClientByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if client == nil {
		return nil, nil
	}

	name := client.Name
	if update.Name != nil {
		name = *update.Name
	}
	enabled := client.Enabled
	if update.Enabled != nil {
		enabled = *update.Enabled
	}
	token := client.SubscriptionToken
	if update.RotateToken {
		token, err = security.RandomHex(48)
		if err != nil {
			return nil, err
		}
	}

	_, err = s.db.ExecContext(ctx, `
		UPDATE clients
		SET name = ?, enabled = ?, subscription_token = ?, updated_at = ?
		WHERE id = ?
	`, name, boolToInt(enabled), token, nowISO(), id)
	if err != nil {
		return nil, err
	}
	return s.GetClientByID(ctx, id)
}

func (s *Store) DeleteClient(ctx context.Context, id int64) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM clients WHERE id = ?`, id)
	return err
}

func (s *Store) UpdateClientTraffic(ctx context.Context, id int64, update ClientTrafficUpdate) error {
	_, err := s.db.ExecContext(ctx, `
		UPDATE clients
		SET rx_bytes = ?, tx_bytes = ?, rx_bps = ?, tx_bps = ?, last_seen_at = ?, updated_at = ?
		WHERE id = ?
	`, update.RXBytes, update.TXBytes, update.RXBps, update.TXBps, nullableString(update.LastSeenAt), nowISO(), id)
	return err
}

func (s *Store) UpdateServiceMetrics(ctx context.Context, state string, totalRX, totalTX int64) error {
	_, err := s.db.ExecContext(ctx, `
		UPDATE service_metrics
		SET service_state = ?, total_rx_bytes = ?, total_tx_bytes = ?, updated_at = ?
		WHERE id = 1
	`, state, totalRX, totalTX, nowISO())
	return err
}

func (s *Store) GetServiceMetrics(ctx context.Context) (*ServiceMetrics, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT service_state, total_rx_bytes, total_tx_bytes, updated_at
		FROM service_metrics
		WHERE id = 1
	`)
	metrics := &ServiceMetrics{}
	if err := row.Scan(&metrics.ServiceState, &metrics.TotalRXBytes, &metrics.TotalTXBytes, &metrics.UpdatedAt); err != nil {
		return nil, err
	}
	return metrics, nil
}

func scanAdmin(row interface{ Scan(dest ...any) error }) (*Admin, error) {
	admin := &Admin{}
	if err := row.Scan(&admin.ID, &admin.Username, &admin.PasswordHash, &admin.CreatedAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return admin, nil
}

func scanClient(row interface{ Scan(dest ...any) error }) (*Client, error) {
	client := &Client{}
	var enabled int
	var lastSeen sql.NullString
	if err := row.Scan(
		&client.ID,
		&client.Name,
		&client.Slug,
		&client.UUID,
		&client.EmailTag,
		&client.ShortID,
		&client.SubscriptionToken,
		&enabled,
		&lastSeen,
		&client.RXBytes,
		&client.TXBytes,
		&client.RXBps,
		&client.TXBps,
		&client.CreatedAt,
		&client.UpdatedAt,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	client.Enabled = enabled == 1
	if lastSeen.Valid {
		client.LastSeenAt = &lastSeen.String
	}
	return client, nil
}

func nowISO() string {
	return time.Now().UTC().Format(time.RFC3339)
}

func slugify(input string) string {
	lower := strings.ToLower(input)
	var b strings.Builder
	lastDash := false
	for _, r := range lower {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
			lastDash = false
			continue
		}
		if !lastDash {
			b.WriteByte('-')
			lastDash = true
		}
	}
	slug := strings.Trim(b.String(), "-")
	if len(slug) > 32 {
		slug = slug[:32]
		slug = strings.Trim(slug, "-")
	}
	if slug == "" {
		randomPart, _ := security.RandomHex(4)
		return "client-" + randomPart
	}
	return slug
}

func boolToInt(value bool) int {
	if value {
		return 1
	}
	return 0
}

func nullableString(value *string) any {
	if value == nil {
		return nil
	}
	return *value
}
