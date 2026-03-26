package runtime

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"xray-reality-solo-vpn/internal/config"
	"xray-reality-solo-vpn/internal/security"
	"xray-reality-solo-vpn/internal/store"
)

type Manager struct {
	cfg        *config.Config
	store      *store.Store
	stateMu    sync.Mutex
	statsState map[int64]statSnapshot
}

type statSnapshot struct {
	RXBytes int64
	TXBytes int64
	At      time.Time
}

type Status struct {
	State   string `json:"state"`
	Message string `json:"message"`
}

type RefreshResult struct {
	State        string `json:"state"`
	Message      string `json:"message"`
	TotalRXBytes int64  `json:"totalRxBytes"`
	TotalTXBytes int64  `json:"totalTxBytes"`
}

type RuntimeSettings struct {
	PanelDomain       string   `json:"panelDomain"`
	PanelBaseURL      string   `json:"panelBaseUrl"`
	LineDomain        string   `json:"lineDomain"`
	LineServerAddress string   `json:"lineServerAddress"`
	XrayTarget        string   `json:"xrayTarget"`
	XrayServerNames   []string `json:"xrayServerNames"`
}

func NewManager(cfg *config.Config, store *store.Store) *Manager {
	return &Manager{
		cfg:        cfg,
		store:      store,
		statsState: map[int64]statSnapshot{},
	}
}

func (m *Manager) Settings() RuntimeSettings {
	return RuntimeSettings{
		PanelDomain:       m.cfg.PanelDomain,
		PanelBaseURL:      m.cfg.PanelBaseURL,
		LineDomain:        m.cfg.LineDomain,
		LineServerAddress: m.cfg.LineServerAddress,
		XrayTarget:        m.cfg.XrayRealityTarget,
		XrayServerNames:   append([]string(nil), m.cfg.XrayRealityNames...),
	}
}

func BytesToHuman(bytes int64) string {
	units := []string{"B", "KB", "MB", "GB", "TB"}
	value := float64(bytes)
	unit := 0
	for value >= 1024 && unit < len(units)-1 {
		value /= 1024
		unit++
	}
	if unit == 0 {
		return fmt.Sprintf("%.0f %s", value, units[unit])
	}
	return fmt.Sprintf("%.2f %s", value, units[unit])
}

func (m *Manager) BuildClientShareLink(client store.Client) string {
	sni := ""
	if len(m.cfg.XrayRealityNames) > 0 {
		sni = m.cfg.XrayRealityNames[0]
	}
	return fmt.Sprintf(
		"vless://%s@%s:%d?encryption=none&security=reality&sni=%s&fp=chrome&pbk=%s&sid=%s&type=tcp&headerType=none#%s",
		client.UUID,
		m.cfg.LineServerAddress,
		m.cfg.LinePublicPort,
		escape(sni),
		escape(m.cfg.XrayPublicKey),
		client.ShortID,
		escape(client.Name),
	)
}

func (m *Manager) BuildMihomoConfig(client store.Client) string {
	serverName := ""
	if len(m.cfg.XrayRealityNames) > 0 {
		serverName = m.cfg.XrayRealityNames[0]
	}
	return fmt.Sprintf(`port: 7890
socks-port: 7891
allow-lan: false
mode: rule
log-level: info
ipv6: true

proxies:
  - name: %s
    type: vless
    server: %s
    port: %d
    uuid: %s
    network: tcp
    tls: true
    udp: false
    servername: %s
    client-fingerprint: %s
    reality-opts:
      public-key: %s
      short-id: %s

proxy-groups:
  - name: %s
    type: select
    proxies:
      - %s
      - %s

rules:
  - %s
  - %s
`,
		security.YAMLQuote(client.Name),
		security.YAMLQuote(m.cfg.LineServerAddress),
		m.cfg.LinePublicPort,
		security.YAMLQuote(client.UUID),
		security.YAMLQuote(serverName),
		security.YAMLQuote("chrome"),
		security.YAMLQuote(m.cfg.XrayPublicKey),
		security.YAMLQuote(client.ShortID),
		security.YAMLQuote("PROXY"),
		security.YAMLQuote(client.Name),
		security.YAMLQuote("DIRECT"),
		security.YAMLQuote("GEOIP,CN,DIRECT"),
		security.YAMLQuote("MATCH,PROXY"),
	)
}

func (m *Manager) WriteRuntimeArtifacts(ctx context.Context) error {
	clients, err := m.store.ListClients(ctx)
	if err != nil {
		return err
	}

	data, err := json.MarshalIndent(renderRealityServerConfig(m.cfg, clients), "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')

	target := filepath.Join(m.cfg.GeneratedDir, "server.json")
	tmp := target + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, target)
}

func (m *Manager) RestartXray(ctx context.Context) Status {
	if err := m.WriteRuntimeArtifacts(ctx); err != nil {
		return Status{State: "error", Message: err.Error()}
	}

	output, err := m.execCommand(ctx, "systemctl", "restart", m.cfg.XrayServiceName)
	if err != nil {
		return Status{State: "error", Message: trimMessage(err.Error())}
	}
	if strings.TrimSpace(output) == "" {
		return Status{State: "running", Message: "Xray restarted"}
	}
	return Status{State: "running", Message: strings.TrimSpace(output)}
}

func (m *Manager) ReadServiceStatus(ctx context.Context) Status {
	output, err := m.execCommand(ctx, "systemctl", "is-active", m.cfg.XrayServiceName)
	if err != nil {
		message := strings.TrimSpace(output)
		if message == "" {
			message = trimMessage(err.Error())
		}
		return Status{State: "unknown", Message: message}
	}
	state := strings.ToLower(strings.TrimSpace(output))
	if state == "" {
		state = "unknown"
	}
	return Status{State: state, Message: strings.TrimSpace(output)}
}

func (m *Manager) RefreshTrafficStats(ctx context.Context) RefreshResult {
	serviceStatus := m.ReadServiceStatus(ctx)
	clients, err := m.store.ListClients(ctx)
	if err != nil {
		return RefreshResult{State: "degraded", Message: err.Error()}
	}
	if serviceStatus.State != "active" && serviceStatus.State != "running" {
		_ = m.store.UpdateServiceMetrics(ctx, serviceStatus.State, 0, 0)
		return RefreshResult{State: serviceStatus.State, Message: serviceStatus.Message}
	}

	output, err := m.execCommand(
		ctx,
		m.cfg.XrayExecutable,
		"api",
		"statsquery",
		fmt.Sprintf("--server=127.0.0.1:%d", m.cfg.XrayAPIListenPort),
		"-pattern",
		"user>>>",
	)
	if err != nil {
		_ = m.store.UpdateServiceMetrics(ctx, "degraded", 0, 0)
		return RefreshResult{State: "degraded", Message: trimMessage(err.Error())}
	}

	var parsed struct {
		Stat []struct {
			Name  string `json:"name"`
			Value int64  `json:"value,string"`
		} `json:"stat"`
	}
	if err := json.Unmarshal([]byte(output), &parsed); err != nil {
		_ = m.store.UpdateServiceMetrics(ctx, "degraded", 0, 0)
		return RefreshResult{State: "degraded", Message: "invalid stats payload"}
	}

	now := time.Now().UTC()
	var totalRX, totalTX int64
	for _, client := range clients {
		txBytes := extractCounter(parsed.Stat, fmt.Sprintf("user>>>%s>>>traffic>>>uplink", client.EmailTag))
		rxBytes := extractCounter(parsed.Stat, fmt.Sprintf("user>>>%s>>>traffic>>>downlink", client.EmailTag))
		totalTX += txBytes
		totalRX += rxBytes

		m.stateMu.Lock()
		previous, ok := m.statsState[client.ID]
		if !ok {
			previous = statSnapshot{RXBytes: rxBytes, TXBytes: txBytes, At: now}
		}
		elapsed := now.Sub(previous.At).Seconds()
		if elapsed < 1 {
			elapsed = 1
		}
		rxDelta := rxBytes - previous.RXBytes
		txDelta := txBytes - previous.TXBytes
		if rxDelta < 0 {
			rxDelta = 0
		}
		if txDelta < 0 {
			txDelta = 0
		}
		m.statsState[client.ID] = statSnapshot{RXBytes: rxBytes, TXBytes: txBytes, At: now}
		m.stateMu.Unlock()

		var lastSeen *string
		if rxDelta > 0 || txDelta > 0 {
			value := now.Format(time.RFC3339)
			lastSeen = &value
		} else {
			lastSeen = client.LastSeenAt
		}

		_ = m.store.UpdateClientTraffic(ctx, client.ID, store.ClientTrafficUpdate{
			RXBytes:    rxBytes,
			TXBytes:    txBytes,
			RXBps:      int64(float64(rxDelta) / elapsed),
			TXBps:      int64(float64(txDelta) / elapsed),
			LastSeenAt: lastSeen,
		})
	}

	_ = m.store.UpdateServiceMetrics(ctx, "running", totalRX, totalTX)
	return RefreshResult{
		State:        "running",
		Message:      serviceStatus.Message,
		TotalRXBytes: totalRX,
		TotalTXBytes: totalTX,
	}
}

func renderRealityServerConfig(cfg *config.Config, clients []store.Client) map[string]any {
	enabledClients := make([]map[string]string, 0)
	shortIDs := make([]string, 0)
	for _, client := range clients {
		if !client.Enabled {
			continue
		}
		enabledClients = append(enabledClients, map[string]string{
			"id":    client.UUID,
			"email": client.EmailTag,
		})
		shortIDs = append(shortIDs, client.ShortID)
	}
	if len(shortIDs) == 0 {
		shortIDs = []string{"0123456789abcdef"}
	}

	return map[string]any{
		"log":   map[string]any{"loglevel": "warning"},
		"stats": map[string]any{},
		"api": map[string]any{
			"tag":      "api",
			"services": []string{"StatsService"},
		},
		"policy": map[string]any{
			"levels": map[string]any{
				"0": map[string]bool{
					"statsUserUplink":   true,
					"statsUserDownlink": true,
				},
			},
			"system": map[string]bool{
				"statsInboundUplink":    true,
				"statsInboundDownlink":  true,
				"statsOutboundUplink":   true,
				"statsOutboundDownlink": true,
			},
		},
		"inbounds": []any{
			map[string]any{
				"listen":   "127.0.0.1",
				"port":     cfg.XrayListenPort,
				"protocol": "vless",
				"settings": map[string]any{
					"clients":    enabledClients,
					"decryption": "none",
				},
				"streamSettings": map[string]any{
					"network":  "raw",
					"security": "reality",
					"realitySettings": map[string]any{
						"show":        false,
						"dest":        cfg.XrayRealityTarget,
						"xver":        0,
						"serverNames": cfg.XrayRealityNames,
						"privateKey":  cfg.XrayPrivateKey,
						"shortIds":    shortIDs,
					},
				},
			},
			map[string]any{
				"listen":   "127.0.0.1",
				"port":     cfg.XrayAPIListenPort,
				"protocol": "dokodemo-door",
				"settings": map[string]any{
					"address": "127.0.0.1",
				},
				"tag": "api",
			},
		},
		"outbounds": []any{
			map[string]string{"protocol": "freedom", "tag": "direct"},
			map[string]string{"protocol": "blackhole", "tag": "block"},
		},
		"routing": map[string]any{
			"rules": []any{
				map[string]any{
					"type":        "field",
					"inboundTag":  []string{"api"},
					"outboundTag": "api",
				},
			},
		},
	}
}

func extractCounter(stats []struct {
	Name  string `json:"name"`
	Value int64  `json:"value,string"`
}, name string) int64 {
	for _, item := range stats {
		if item.Name == name {
			return item.Value
		}
	}
	return 0
}

func trimMessage(message string) string {
	message = strings.TrimSpace(message)
	if message == "" {
		return "runtime command failed"
	}
	return message
}

func escape(value string) string {
	return strings.ReplaceAll(url.QueryEscape(value), "+", "%20")
}

func (m *Manager) execCommand(ctx context.Context, name string, args ...string) (string, error) {
	commandCtx, cancel := context.WithTimeout(ctx, 8*time.Second)
	defer cancel()

	cmd := exec.CommandContext(commandCtx, name, args...)
	output, err := cmd.CombinedOutput()
	text := strings.TrimSpace(string(output))
	if err != nil {
		if text == "" {
			return "", err
		}
		return text, fmt.Errorf("%w: %s", err, text)
	}
	return text, nil
}
