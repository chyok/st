package discovery

import (
	"fmt"
	"net"
	"strconv"
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

// Listen 监听发现广播
func Listen(role Role, filePath string) {
	conn, err := net.ListenPacket("udp", config.G.MulticastAddress)
	if err != nil {
		fmt.Printf("Failed to listen on %s: %v\n", config.G.MulticastAddress, err)
		return
	}
	defer conn.Close()

	buf := make([]byte, 1024)
	for {
		n, remoteAddr, err := conn.ReadFrom(buf)
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

		fmt.Printf("Discovered device: %s (%s)\n", deviceName, remoteAddr)

		switch remoteRole {
		case Sender:
			if role == Receiver {
				go transfer.ReceiveFile(remoteAddr.String(), filePath)
			}
		case Receiver:
			if role == Sender {
				go transfer.SendFile(filePath, remoteAddr.String())
			}
		}
	}
}

// Send 发送发现广播
func Send(role Role) {
	port, _ := strconv.Atoi(config.G.Port)

	conn, err := net.DialUDP("udp", nil, &net.UDPAddr{
		IP:   net.ParseIP(config.G.MulticastAddress),
		Port: port,
	})
	if err != nil {
		fmt.Printf("Failed to dial %s: %v\n", config.G.MulticastAddress, err)
		return
	}
	defer conn.Close()

	message := fmt.Sprintf("%s%s%s", config.G.DeviceName, separator, role)
	conn.Write([]byte(message))
	fmt.Printf("Sent discovery message: %s\n", message)
}
