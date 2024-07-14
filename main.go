package main

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"

	"github.com/chyok/st/config"
	"github.com/chyok/st/internal/discovery"
	"github.com/chyok/st/internal/handler"
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

	http.HandleFunc("/", handler.ReceiveHandler)
	http.Handle("/static/", http.StripPrefix("/static/",
		http.FileServer(http.FS(web.CssFs))))

	fmt.Println("Waiting for transfer...")
	return http.ListenAndServe(config.G.WildcardAddress, nil)
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

	http.HandleFunc("/", handler.SendHandler)
	http.HandleFunc("/download/", handler.FileServerHandler)

	fmt.Println("Waiting for transfer...")

	return http.ListenAndServe(config.G.WildcardAddress, nil)
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
				fmt.Printf("st version %s\n", config.G.Version)
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
	err := app.Run(os.Args)
	if err != nil {
		fmt.Println(err.Error())
	}
}
