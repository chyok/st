package discovery

import (
	"fmt"
	"net"
	"strings"

	"github.com/chyok/st/config"
	"github.com/chyok/st/internal/transfer"
)

const separator = "|"

type Role string

const (
	Sender   Role = "sender"
	Receiver Role = "receiver"
)

func Listen(role Role, filePath string) {
	addr, err := net.ResolveUDPAddr("udp", config.G.MulticastAddress)

	if err != nil {
		fmt.Printf("Failed to resolve %s: %v\n", config.G.MulticastAddress, err)
		return
	}
	conn, err := net.ListenMulticastUDP("udp", nil, addr)
	if err != nil {
		fmt.Printf("Failed to listen on %s: %v\n", config.G.MulticastAddress, err)
		return
	}
	defer conn.Close()

	buf := make([]byte, 1024)
	for {
		n, src, err := conn.ReadFromUDP(buf)

		remoteAddr := src.IP.String() + ":" + config.G.Port

		if err != nil {
			fmt.Printf("Failed to read from %s: %v\n", remoteAddr, err)
			continue
		}

		message := string(buf[:n])
		parts := strings.Split(message, separator)
		if len(parts) != 2 {
			fmt.Printf("Received malformed message from %s: %s\n", remoteAddr, message)
			continue
		}

		deviceName := parts[0]
		remoteRole := Role(parts[1])
		switch remoteRole {
		case Sender:
			if role == Sender {
				fmt.Printf("Discovered Sender: %s (%s)\n", deviceName, remoteAddr)
				go func() {
					err := transfer.ReceiveFile(remoteAddr)
					if err != nil {
						fmt.Printf("Receive file from %s error: %s\n", remoteAddr, err)
					}
				}()
			}
		case Receiver:
			if role == Receiver {
				fmt.Printf("Discovered Receiver: %s (%s)\n", deviceName, remoteAddr)
				go func() {
					err := transfer.SendFile(filePath, fmt.Sprintf("http://%s", remoteAddr))
					if err != nil {
						fmt.Printf("Send file to %s error: %s\n", remoteAddr, err)
					}
				}()
			}
		}
	}
}

func Send(role Role) {
	addr, err := net.ResolveUDPAddr("udp", config.G.MulticastAddress)
	if err != nil {
		fmt.Printf("Failed to resolve %s: %v\n", config.G.MulticastAddress, err)
		return
	}

	conn, err := net.DialUDP("udp", nil, addr)
	if err != nil {
		fmt.Printf("Failed to dial %s: %v\n", config.G.MulticastAddress, err)
		return
	}
	defer conn.Close()

	message := fmt.Sprintf("%s%s%s", config.G.DeviceName, separator, role)
	conn.Write([]byte(message))
}
