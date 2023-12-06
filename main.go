package main

import (
	"fmt"
	"net/http"
	"os"

	"github.com/chyok/st/config"
	"github.com/chyok/st/internal/pkg/discovery"
	"github.com/chyok/st/internal/pkg/transfer"
	"github.com/chyok/st/web"
	"github.com/skip2/go-qrcode"

	"github.com/urfave/cli/v2"
)

func sendFile(c *cli.Context) error {
	filePath := c.Args().Get(0)

	go discovery.Listen(config.G.WildcardAddress)
	go discovery.Send(config.G.MulticastAddress, config.G.DeviceName)

	transfer.SendToAll(filePath)
	return nil
}

func receiveFile(c *cli.Context) error {
	address := fmt.Sprintf("http://%s:%s", config.G.LocalIP, config.G.Port)
	q, _ := qrcode.New(address, qrcode.Low)
	fmt.Println(q.ToSmallString(false))
	fmt.Printf("Server address: %s \n", address)
	fmt.Println("Waiting transfer...")
	go discovery.Listen(config.G.MulticastAddress)
	http.HandleFunc("/", transfer.ReceiveFileHandler)
	http.Handle("/static/", http.StripPrefix("/static/",
		http.FileServer(http.FS(web.CssFs))))
	http.ListenAndServe(config.G.WildcardAddress, nil)
	return nil
}

var port string

func initConfig(*cli.Context) error {
	config.G.SetConf(port)
	return nil
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
