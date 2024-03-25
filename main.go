package main

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"

	"github.com/chyok/st/config"
	"github.com/chyok/st/internal/discovery"
	"github.com/chyok/st/internal/transfer"
	"github.com/chyok/st/web"
	"github.com/skip2/go-qrcode"
	"github.com/urfave/cli/v2"
)

var (
	port     string
	filePath string
)

func initConfig(c *cli.Context) error {
	config.G.SetConf(port)
	return nil
}

func sendFile(c *cli.Context) error {
	go discovery.Send(discovery.Sender)
	go discovery.Listen(discovery.Receiver, filePath)

	url := fmt.Sprintf("http://%s:%s", config.G.LocalIP, config.G.Port)
	q, err := qrcode.New(url, qrcode.Low)
	if err != nil {
		return err
	}
	fmt.Println(q.ToSmallString(false))
	fmt.Printf("Server address: %s\n", url)

	http.HandleFunc("/"+filepath.Base(filePath), func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, filePath)
	})

	return http.ListenAndServe(config.G.WildcardAddress, nil)
}

func receiveFile(c *cli.Context) error {
	go discovery.Send(discovery.Receiver)
	go discovery.Listen(discovery.Sender, "")

	address := fmt.Sprintf("http://%s:%s", config.G.LocalIP, config.G.Port)
	q, err := qrcode.New(address, qrcode.Low)
	if err != nil {
		return err
	}
	fmt.Println(q.ToSmallString(false))
	fmt.Printf("Server address: %s\n", address)
	fmt.Println("Waiting for transfer...")

	http.HandleFunc("/", transfer.ReceiveHandler)
	http.Handle("/static/", http.StripPrefix("/static/",
		http.FileServer(http.FS(web.CssFs))))
	return http.ListenAndServe(config.G.WildcardAddress, nil)
}

func main() {
	app := cli.NewApp()
	app.Flags = []cli.Flag{
		&cli.StringFlag{
			Name:        "port",
			Value:       "9999",
			Destination: &port,
			Usage:       "Server port",
		},
		&cli.StringFlag{
			Name:        "file",
			Destination: &filePath,
			Usage:       "File or directory to send",
		},
	}
	app.Before = initConfig
	app.Action = func(c *cli.Context) error {
		if filePath != "" {
			return sendFile(c)
		} else {
			return receiveFile(c)
		}
	}

	err := app.Run(os.Args)
	if err != nil {
		fmt.Println(err)
	}
}
