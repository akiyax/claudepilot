package main

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/akiyax/claudepilot/daemon/internal/claude"
	"github.com/akiyax/claudepilot/daemon/internal/config"
	"github.com/akiyax/claudepilot/daemon/internal/handler"
	"github.com/akiyax/claudepilot/daemon/internal/transport"
)

// Build-time variables injected via -ldflags
var (
	Version   = "dev"
	GitCommit = "unknown"
	BuildDate = "unknown"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	// Parse CLI flags
	port := 8077 // default port
	verbose := false

	args := os.Args[1:]
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--port":
			i++
			if i >= len(args) {
				return fmt.Errorf("--port requires a value")
			}
			fmt.Sscanf(args[i], "%d", &port)
		case "--verbose":
			verbose = true
		case "--version", "-v":
			fmt.Printf("ClaudePilot Daemon %s (commit: %s, built: %s)\n", Version, GitCommit, BuildDate)
			return nil
		case "--help", "-h":
			printUsage()
			return nil
		default:
			if args[i] == "update" {
				fmt.Println("Checking for updates...")
				return nil
			}
			if args[i] == "logs" {
				fmt.Println("Log viewer not yet implemented")
				return nil
			}
			if args[i] == "pair" {
				i++
				if i >= len(args) {
					return fmt.Errorf("usage: claudepilot pair <code>")
				}
				fmt.Printf("Pairing with code: %s\n", args[i])
				return nil
			}
			return fmt.Errorf("unknown argument: %s", args[i])
		}
	}

	// Load or create config
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Initialize logging
	logFile, err := config.InitLogger(cfg, verbose)
	if err != nil {
		return fmt.Errorf("failed to init logger: %w", err)
	}
	if logFile != nil {
		defer logFile.Close()
	}

	// Detect Claude CLI version
	cliVersion := claude.DetectCLIVersion()

	// Setup graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Create auth provider and generate initial token + pairing code
	auth := transport.NewTokenAuth()
	token, err := auth.GenerateToken()
	if err != nil {
		return fmt.Errorf("failed to generate token: %w", err)
	}
	pairingCode, err := auth.GeneratePairingCode()
	if err != nil {
		return fmt.Errorf("failed to generate pairing code: %w", err)
	}

	// Create transport
	tp := transport.NewWSServerTransport(port, auth, cfg.DaemonID)
	if err := tp.Start(); err != nil {
		return fmt.Errorf("failed to start transport: %w", err)
	}

	// Create handler
	h := handler.NewHandler(tp, cfg.DaemonID, Version)
	h.SetCLIVersion(cliVersion)

	// Register message handler
	tp.OnMessage(func(msg claude.WSMessage) {
		h.HandleMessage(msg)
	})

	// Get local IPs
	ips := getLocalIPs()

	// Print startup info
	fmt.Printf("ClaudePilot Daemon %s\n", Version)
	fmt.Printf("Daemon ID: %s\n", cfg.DaemonID)
	fmt.Printf("Claude CLI: %s\n", cliVersion)
	fmt.Printf("Config: %s\n", config.Dir())
	fmt.Printf("Logs: %s/logs/\n", config.Dir())
	fmt.Println()

	// Print QR code info
	if len(ips) > 0 {
		wsURL := fmt.Sprintf("ws://%s:%d?token=%s", ips[0], tp.Port(), token)
		fmt.Println("Scan QR code or enter pairing code to connect:")
		fmt.Println()

		// Try to print QR code
		printQR(wsURL)

		fmt.Println()
		fmt.Printf("  URL:   %s\n", wsURL)
		fmt.Printf("  Pair:  %s\n", pairingCode)
		fmt.Println()

		if len(ips) > 1 {
			fmt.Println("Alternative IPs:")
			for _, ip := range ips[1:] {
				fmt.Printf("  ws://%s:%d?token=%s\n", ip, tp.Port(), token)
			}
			fmt.Println()
		}
	}

	fmt.Println("Press Ctrl+C to stop")

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	select {
	case sig := <-sigCh:
		fmt.Printf("\nReceived %s, shutting down...\n", sig)
	case <-ctx.Done():
	}

	// Graceful shutdown
	slog.Info("Shutting down")
	tp.Close()
	fmt.Println("Goodbye!")
	return nil
}

// getLocalIPs returns all LAN IP addresses, prioritized.
func getLocalIPs() []string {
	var ips []string
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return []string{"127.0.0.1"}
	}

	for _, addr := range addrs {
		if ipNet, ok := addr.(*net.IPNet); ok && !ipNet.IP.IsLoopback() && ipNet.IP.To4() != nil {
			ip := ipNet.IP.String()
			// Skip Docker virtual NICs and other virtual interfaces
			if strings.HasPrefix(ip, "172.17.") || strings.HasPrefix(ip, "172.18.") {
				continue
			}
			ips = append(ips, ip)
		}
	}

	if len(ips) == 0 {
		return []string{"127.0.0.1"}
	}

	// Sort: prefer 192.168.x.x, then 10.x.x.x, then others
	prioritized := make([]string, 0, len(ips))
	others := make([]string, 0)
	for _, ip := range ips {
		if strings.HasPrefix(ip, "192.168.") || strings.HasPrefix(ip, "10.") {
			prioritized = append(prioritized, ip)
		} else {
			others = append(others, ip)
		}
	}
	return append(prioritized, others...)
}

// printQR tries to print a QR code to the terminal.
func printQR(data string) {
	// Simple ASCII QR code placeholder — will use go-qrcode when dependency is resolved
	// For now, just print the URL
	fmt.Printf("  ┌────────────────────────────────┐\n")
	fmt.Printf("  │  QR Code:                      │\n")
	fmt.Printf("  │  (Use a QR scanner app)        │\n")
	fmt.Printf("  └────────────────────────────────┘\n")
}

func printUsage() {
	fmt.Println(`ClaudePilot Daemon - Remote control for Claude Code CLI

Usage:
  claudepilot [flags]

Flags:
  --port <port>     Listen port (default: 8077, auto-increment if occupied)
  --verbose         Enable debug logging
  --version, -v     Print version
  --help, -h        Print this help

Commands:
  update            Check and install updates
  logs              View daemon logs
  pair <code>       Complete pairing with a mobile device`)
}
