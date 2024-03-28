package main

import (
	"fmt"
	"net/http"
	"os"

	"github.com/chyok/st/config"
	"github.com/chyok/st/internal/discovery"
	"github.com/chyok/st/internal/transfer"
	"github.com/chyok/st/web"
	"github.com/skip2/go-qrcode"
	"github.com/urfave/cli/v2"
)

var (
	port string
)

func initConfig(c *cli.Context) error {
	config.G.SetConf(port)
	return nil
}

func sendFile(_ *cli.Context) error {
	go discovery.Send(discovery.Sender)
	go discovery.Listen(discovery.Receiver, config.G.FilePath)

	url := fmt.Sprintf("http://%s:%s", config.G.LocalIP, config.G.Port)
	q, err := qrcode.New(url, qrcode.Low)
	if err != nil {
		return err
	}
	fmt.Println(q.ToSmallString(false))
	fmt.Printf("Server address: %s\n", url)

	http.HandleFunc("/", transfer.SendHandler)
	http.HandleFunc("/download/", transfer.DownloadFileHandler)

	return http.ListenAndServe(config.G.WildcardAddress, nil)
}

func receiveFile(_ *cli.Context) error {
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
	app := &cli.App{
		Name:      "st",
		Usage:     "simple file transfer tool",
		UsageText: "st [global options] [filename]",
		Description: "if file name provided, it will attempt to send files to all servers\n" +
			"else become the server and wait to receive the file",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:        "port",
				Value:       "9999",
				Usage:       "server port",
				Aliases:     []string{"p"},
				Destination: &port,
			},
		},
		Action: func(c *cli.Context) error {
			if c.NArg() > 0 {
				config.G.FilePath = c.Args().Get(0)
				return sendFile(c)
			}
			return receiveFile(c)
		},
		Before: initConfig,
	}
	err := app.Run(os.Args)
	if err != nil {
		panic(err)
	}
}
