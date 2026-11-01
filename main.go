package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/chyok/st/internal/config"
	"github.com/chyok/st/internal/discovery"
	"github.com/chyok/st/internal/handler"
	"github.com/chyok/st/internal/transfer"
	"github.com/chyok/st/internal/util"
	"github.com/skip2/go-qrcode"
	"github.com/urfave/cli/v2"
)

func printAddress(cfg *config.Config) error {
	if len(cfg.Addresses) > 1 {
		fmt.Println("No internet connection, so the right network is unknown.")
		fmt.Println("Use the address on the same network as the other device:")
	}
	for _, a := range cfg.Addresses {
		address := fmt.Sprintf("http://%s:%s", a.IP, cfg.Port)
		q, err := qrcode.New(address, qrcode.Low)
		if err != nil {
			return err
		}
		fmt.Println(q.ToSmallString(false))
		if len(cfg.Addresses) > 1 {
			fmt.Printf("Server address (%s): %s\n\n", a.Interface, address)
		} else {
			fmt.Printf("Server address: %s\n", address)
		}
	}
	return nil
}

// runDiscovery finds other st instances; the transfer still works through
// the web page or QR code if it fails, so it only reports the error.
func runDiscovery(cfg *config.Config, role discovery.Role, receive func(discovery.Announcement, string)) {
	if err := discovery.Run(cfg, role, receive); err != nil {
		fmt.Printf("Device discovery stopped: %v\n", err)
	}
}

func receiveClient(cfg *config.Config, client *transfer.Client) error {
	go runDiscovery(cfg, discovery.Receiver, func(peer discovery.Announcement, addr string) {
		if err := client.ReceiveFiles(addr); err != nil {
			fmt.Printf("Receive file from %s error: %s\n", addr, err)
		}
	})

	if err := printAddress(cfg); err != nil {
		return err
	}
	h := &handler.Receiver{DeviceName: cfg.DeviceName, Dir: client.Dir}
	fmt.Println("Waiting for transfer...")
	return serve(cfg, h.Routes())
}

func sendClient(cfg *config.Config) error {
	go runDiscovery(cfg, discovery.Sender, nil)

	if err := printAddress(cfg); err != nil {
		return err
	}
	h := &handler.Sender{DeviceName: cfg.DeviceName, Path: cfg.FilePath}
	fmt.Println("Waiting for transfer...")
	return serve(cfg, h.Routes())
}

// serve runs the HTTP server until it fails or the process is interrupted.
// On Ctrl+C it gives in-flight requests a moment to finish, then cuts the
// rest off and removes the temporary files of interrupted transfers.
func serve(cfg *config.Config, h http.Handler) error {
	srv := &http.Server{
		Addr:              cfg.ListenAddr(),
		Handler:           h,
		ReadHeaderTimeout: 10 * time.Second,
		IdleTimeout:       2 * time.Minute,
		// No read/write timeouts: transfers of large files can take long.
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	errc := make(chan error, 1)
	go func() { errc <- srv.ListenAndServe() }()
	select {
	case err := <-errc:
		return err
	case <-ctx.Done():
	}
	stop() // a second Ctrl+C exits immediately

	fmt.Println("\nShutting down...")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		srv.Close()
	}
	util.RemovePartFiles()
	return nil
}

func main() {
	app := &cli.App{
		Name:      "st",
		Usage:     "simple file transfer tool",
		UsageText: "st [global options] [filename|foldername]",
		Description: "st is a simple command-line tool for fast local file/folder sharing, \n" +
			"offering web-based transfer with QR code scanning and automatic device discovery.",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "port",
				Value:   "53333",
				Usage:   "server port",
				Aliases: []string{"p"},
			},
			&cli.BoolFlag{
				Name:    "version",
				Aliases: []string{"v"},
				Usage:   "print the version",
			},
		},
		Action: func(c *cli.Context) error {
			if c.Bool("version") {
				fmt.Printf("st version %s\n", config.CurrentVersion())
				return nil
			}

			cfg, err := config.New(c.String("port"))
			if err != nil {
				return err
			}
			if c.NArg() > 0 {
				absPath, err := filepath.Abs(c.Args().Get(0))
				if err != nil {
					return fmt.Errorf("file error: %w", err)
				}
				if _, err := os.Stat(absPath); err != nil {
					return fmt.Errorf("file or folder not found: %w", err)
				}
				cfg.FilePath = absPath
				return sendClient(cfg)
			}
			return receiveClient(cfg, &transfer.Client{Dir: "."})
		},
	}
	if err := app.Run(os.Args); err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}
}
