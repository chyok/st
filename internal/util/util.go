package util

import (
	"context"
	"fmt"
	"net"
	"strings"
	"time"
)

func In[T comparable](v T, arr []T) bool {
	for _, data := range arr {
		if v == data {
			return true
		}
	}
	return false
}

func PrintWaitDots(ctx context.Context, waitMessage string) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
			fmt.Print(waitMessage)
			for i := 0; i < 3; i++ {
				fmt.Print(".")
				time.Sleep(time.Microsecond * 500)
			}
			fmt.Printf("/r%s/r", strings.Repeat(" ", len(waitMessage)+3))
		}
	}
}

func GetLocalIPs() ([]string, error) {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return nil, err
	}
	var ips []string
	for _, addr := range addrs {
		var ip net.IP
		switch v := addr.(type) {
		case *net.IPNet:
			ip = v.IP
		case *net.IPAddr:
			ip = v.IP
		}
		if ip.To4() != nil {
			ips = append(ips, ip.To4().String())
		}

	}
	return ips, nil
}
