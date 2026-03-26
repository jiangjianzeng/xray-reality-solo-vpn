package runtime

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"xray-reality-solo-vpn/internal/config"
	"xray-reality-solo-vpn/internal/store"
)

func TestBuildMihomoConfig(t *testing.T) {
	manager := NewManager(&config.Config{
		LineServerAddress: "1.2.3.4",
		LinePublicPort:    443,
		XrayRealityNames:  []string{"www.cloudflare.com"},
		XrayPublicKey:     "public-key",
	}, nil)

	client := store.Client{
		Name:    "Alice's Mac",
		UUID:    "uuid-1",
		ShortID: "0123456789abcdef",
	}

	output := manager.BuildMihomoConfig(client)
	if !strings.Contains(output, "Alice''s Mac") {
		t.Fatalf("expected escaped client name, got %s", output)
	}
	if !strings.Contains(output, "public-key") {
		t.Fatalf("expected public key in output, got %s", output)
	}
}

func TestReadServiceStatusUsesSystemctl(t *testing.T) {
	tempDir := t.TempDir()
	logPath := filepath.Join(tempDir, "systemctl.log")
	systemctlPath := filepath.Join(tempDir, "systemctl")
	script := "#!/bin/sh\nprintf '%s\n' \"$@\" >\"" + logPath + "\"\necho active\n"
	if err := os.WriteFile(systemctlPath, []byte(script), 0o755); err != nil {
		t.Fatalf("write fake systemctl: %v", err)
	}

	t.Setenv("PATH", tempDir+string(os.PathListSeparator)+os.Getenv("PATH"))

	manager := NewManager(&config.Config{
		XrayServiceName: "xray-custom",
	}, nil)

	status := manager.ReadServiceStatus(context.Background())
	if status.State != "active" {
		t.Fatalf("expected active state, got %#v", status)
	}
	if status.Message != "active" {
		t.Fatalf("expected active message, got %#v", status)
	}

	logged, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("read fake systemctl log: %v", err)
	}
	if string(logged) != "is-active\nxray-custom\n" {
		t.Fatalf("unexpected systemctl args: %q", string(logged))
	}
}
