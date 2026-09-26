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

	"github.com/chyok/st/config"
	"github.com/chyok/st/internal/discovery"
	"github.com/chyok/st/internal/handler"
	"github.com/chyok/st/internal/transfer"
	"github.com/chyok/st/internal/util"
	"github.com/skip2/go-qrcode"
	"github.com/urfave/cli/v2"
)

var (
	port string
)

func printAddress(cfg *config.Config) error {
	address := fmt.Sprintf("http://%s:%s", cfg.LocalIP, cfg.Port)
	q, err := qrcode.New(address, qrcode.Low)
	if err != nil {
		return err
	}
	fmt.Println(q.ToSmallString(false))
	fmt.Printf("Server address: %s\n", address)
	return nil
}

func receiveClient(cfg *config.Config, client *transfer.Client) error {
	go discovery.Send(cfg, discovery.Receiver)
	go discovery.Listen(cfg, client, discovery.Sender)

	if err := printAddress(cfg); err != nil {
		return err
	}
	h := &handler.Receiver{DeviceName: cfg.DeviceName, Dir: client.Dir}
	fmt.Println("Waiting for transfer...")
	return serve(cfg, h.Routes())
}

func sendClient(cfg *config.Config, client *transfer.Client) error {
	go discovery.Send(cfg, discovery.Sender)
	go discovery.Listen(cfg, client, discovery.Receiver)

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
		Addr:              cfg.WildcardAddress,
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
				Name:        "port",
				Value:       "53333",
				Usage:       "server port",
				Aliases:     []string{"p"},
				Destination: &port,
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

			cfg, err := config.New(port)
			if err != nil {
				return err
			}
			client := &transfer.Client{Dir: "."}

			if c.NArg() > 0 {
				currentPath := filepath.ToSlash(c.Args().Get(0))
				absPath, err := filepath.Abs(currentPath)
				if err != nil {
					return fmt.Errorf("file error: %w", err)
				}
				if _, err := os.Stat(absPath); err != nil {
					return fmt.Errorf("file or folder not found: %w", err)
				}
				cfg.FilePath = absPath
				return sendClient(cfg, client)
			}
			return receiveClient(cfg, client)
		},
	}
	if err := app.Run(os.Args); err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}
}
