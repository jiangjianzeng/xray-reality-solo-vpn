package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	RootDir              string
	AppHost              string
	AppPort              int
	Timezone             string
	PanelDomain          string
	PanelBaseURL         string
	LineDomain           string
	LineServerAddress    string
	LinePublicPort       int
	XrayListenPort       int
	XrayAPIListenPort    int
	XrayRealityTarget    string
	XrayRealityNames     []string
	XrayPrivateKey       string
	XrayPublicKey        string
	SessionSecret        string
	TrustProxy           bool
	DataDir              string
	GeneratedDir         string
	SetupTicketFile      string
	SetupTicketPath      string
	SetupTicketTTL       time.Duration
	XrayServiceName      string
	XrayExecutable       string
	XrayServerPort       int
	AccessTokenTTL       time.Duration
	RateLimitWindow      time.Duration
	WebDistDir           string
	MaxRequestBodyBytes  int64
	RuntimeRefreshPeriod time.Duration
}

func Load() (*Config, error) {
	rootDir, err := os.Getwd()
	if err != nil {
		return nil, err
	}

	cfg := &Config{
		RootDir:              rootDir,
		AppHost:              stringFromEnv("APP_HOST", "127.0.0.1"),
		AppPort:              intFromEnv("APP_PORT", 3000),
		Timezone:             stringFromEnv("TZ", "UTC"),
		PanelDomain:          stringFromEnv("PANEL_DOMAIN", "panel.example.com"),
		PanelBaseURL:         stringFromEnv("PANEL_BASE_URL", "https://panel.example.com"),
		LineDomain:           stringFromEnv("LINE_DOMAIN", "line.example.com"),
		LinePublicPort:       intFromEnv("LINE_PUBLIC_PORT", intFromEnv("XRAY_SERVER_PORT", 443)),
		XrayListenPort:       intFromEnv("XRAY_LISTEN_PORT", 2443),
		XrayAPIListenPort:    intFromEnv("XRAY_API_LISTEN_PORT", 10085),
		XrayRealityTarget:    stringFromEnv("XRAY_REALITY_TARGET", "www.cloudflare.com:443"),
		XrayPrivateKey:       os.Getenv("XRAY_PRIVATE_KEY"),
		XrayPublicKey:        os.Getenv("XRAY_PUBLIC_KEY"),
		SessionSecret:        os.Getenv("SESSION_SECRET"),
		TrustProxy:           boolFromEnv("TRUST_PROXY", true),
		XrayServiceName:      stringFromEnv("XRAY_SERVICE_NAME", "xray"),
		XrayExecutable:       stringFromEnv("XRAY_EXECUTABLE", "xray"),
		AccessTokenTTL:       7 * 24 * time.Hour,
		RateLimitWindow:      15 * time.Minute,
		MaxRequestBodyBytes:  16 * 1024,
		RuntimeRefreshPeriod: 15 * time.Second,
		SetupTicketTTL:       time.Duration(intFromEnv("SETUP_TTL_MINUTES", 30)) * time.Minute,
	}

	cfg.LineServerAddress = stringFromEnv("LINE_SERVER_ADDRESS", cfg.LineDomain)
	cfg.XrayRealityNames = splitCSV(stringFromEnv("XRAY_REALITY_SERVER_NAMES", "www.cloudflare.com"))
	cfg.DataDir = resolvePath(rootDir, stringFromEnv("DATA_DIR", "./data"))
	cfg.GeneratedDir = resolvePath(rootDir, stringFromEnv("GENERATED_DIR", "./generated"))
	cfg.SetupTicketFile = resolvePath(rootDir, stringFromEnv("SETUP_TICKET_FILE", filepath.Join(cfg.DataDir, "setup-ticket.json")))
	cfg.SetupTicketPath = cfg.SetupTicketFile
	cfg.XrayServerPort = cfg.LinePublicPort
	cfg.WebDistDir = filepath.Join(rootDir, "web", "dist")

	if err := os.MkdirAll(cfg.DataDir, 0o755); err != nil {
		return nil, err
	}
	if err := os.MkdirAll(cfg.GeneratedDir, 0o755); err != nil {
		return nil, err
	}

	if cfg.PanelBaseURL == "" {
		return nil, errors.New("PANEL_BASE_URL is required")
	}
	if len(cfg.XrayRealityNames) == 0 {
		return nil, errors.New("XRAY_REALITY_SERVER_NAMES must not be empty")
	}

	return cfg, nil
}

func (c *Config) ListenAddr() string {
	if c.AppHost == "" {
		return fmt.Sprintf(":%d", c.AppPort)
	}
	return fmt.Sprintf("%s:%d", c.AppHost, c.AppPort)
}

func (c *Config) MissingRuntime() []string {
	missing := make([]string, 0, 3)
	if c.XrayPrivateKey == "" {
		missing = append(missing, "XRAY_PRIVATE_KEY")
	}
	if c.XrayPublicKey == "" {
		missing = append(missing, "XRAY_PUBLIC_KEY")
	}
	if c.SessionSecret == "" {
		missing = append(missing, "SESSION_SECRET")
	}
	return missing
}

func stringFromEnv(key, fallback string) string {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	return value
}

func intFromEnv(key string, fallback int) int {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return fallback
	}
	return parsed
}

func boolFromEnv(key string, fallback bool) bool {
	value := strings.TrimSpace(strings.ToLower(os.Getenv(key)))
	if value == "" {
		return fallback
	}
	switch value {
	case "1", "true", "yes", "on":
		return true
	case "0", "false", "no", "off":
		return false
	default:
		return fallback
	}
}

func splitCSV(input string) []string {
	parts := strings.Split(input, ",")
	result := make([]string, 0, len(parts))
	for _, item := range parts {
		trimmed := strings.TrimSpace(item)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}

func resolvePath(rootDir, value string) string {
	if filepath.IsAbs(value) {
		return filepath.Clean(value)
	}
	return filepath.Clean(filepath.Join(rootDir, value))
}
