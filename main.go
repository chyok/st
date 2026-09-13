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
	"github.com/chyok/st/internal/util"
	"github.com/chyok/st/web"
	"github.com/skip2/go-qrcode"
	"github.com/urfave/cli/v2"
)

var (
	port string
)

func initConfig(c *cli.Context) error {
	return config.G.SetConf(port)
}

func receiveClient() error {
	go discovery.Send(discovery.Receiver)
	go discovery.Listen(discovery.Sender, "")

	address := fmt.Sprintf("http://%s:%s", config.G.LocalIP, config.G.Port)
	q, err := qrcode.New(address, qrcode.Low)
	if err != nil {
		return err
	}
	fmt.Println(q.ToSmallString(false))
	fmt.Printf("Server address: %s\n", address)

	mux := http.NewServeMux()
	mux.HandleFunc("/", handler.ReceiveHandler)
	mux.Handle("/static/", http.StripPrefix("/static/",
		http.FileServer(http.FS(web.CssFs))))

	fmt.Println("Waiting for transfer...")
	return serve(mux)
}

func sendClient() error {
	go discovery.Send(discovery.Sender)
	go discovery.Listen(discovery.Receiver, config.G.FilePath)

	url := fmt.Sprintf("http://%s:%s", config.G.LocalIP, config.G.Port)
	q, err := qrcode.New(url, qrcode.Low)
	if err != nil {
		return err
	}
	fmt.Println(q.ToSmallString(false))
	fmt.Printf("Server address: %s\n", url)

	mux := http.NewServeMux()
	mux.HandleFunc("/", handler.SendHandler)
	mux.HandleFunc("/download/", handler.FileServerHandler)

	fmt.Println("Waiting for transfer...")
	return serve(mux)
}

// serve runs the HTTP server until it fails or the process is interrupted.
// On Ctrl+C it gives in-flight requests a moment to finish, then cuts the
// rest off and removes the temporary files of interrupted transfers.
func serve(h http.Handler) error {
	srv := &http.Server{
		Addr:              config.G.WildcardAddress,
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

			if c.NArg() > 0 {
				currentPath := filepath.ToSlash(c.Args().Get(0))
				absPath, err := filepath.Abs(currentPath)
				if err != nil {
					fmt.Printf("File error: %s\n", err)
					return nil
				}
				if _, err := os.Stat(absPath); os.IsNotExist(err) {
					fmt.Printf("File or Folder not exist: %s\n", err)
					return nil
				}
				config.G.FilePath = absPath
				return sendClient()
			}
			return receiveClient()
		},
		Before: initConfig,
	}
	if err := app.Run(os.Args); err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}
}
