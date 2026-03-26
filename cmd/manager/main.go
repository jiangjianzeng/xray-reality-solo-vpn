package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"xray-reality-solo-vpn/internal/app"
	"xray-reality-solo-vpn/internal/config"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		slog.Error("load config", "error", err)
		os.Exit(1)
	}

	application, err := app.New(cfg)
	if err != nil {
		slog.Error("create app", "error", err)
		os.Exit(1)
	}
	defer application.Close()

	if hasArg("--issue-setup-ticket") {
		token, err := application.IssueSetupTicket(context.Background(), cfg.SetupTicketTTL)
		if err != nil {
			slog.Error("issue setup ticket", "error", err)
			os.Exit(1)
		}
		baseURL := strings.TrimRight(cfg.PanelBaseURL, "/")
		fmt.Printf("SETUP_URL=%s/_/setup/%s\n", baseURL, token)
		return
	}

	if hasArg("--setup-only") {
		if err := application.WriteRuntimeArtifacts(context.Background()); err != nil {
			slog.Error("setup", "error", err)
			os.Exit(1)
		}
		fmt.Println("Setup complete.")
		return
	}

	application.StartPolling()

	addr := cfg.ListenAddr()
	slog.Info("manager listening", "addr", addr)

	httpServer := &http.Server{
		Addr:              addr,
		Handler:           application.Router(),
		ReadHeaderTimeout: 5 * time.Second,
	}

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()
	go func() {
		<-ctx.Done()
		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer shutdownCancel()
		_ = httpServer.Shutdown(shutdownCtx)
	}()

	if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		slog.Error("listen", "error", err)
		os.Exit(1)
	}
}

func hasArg(target string) bool {
	for _, arg := range os.Args[1:] {
		if arg == target {
			return true
		}
	}
	return false
}
