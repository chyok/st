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

// Listen reacts to announcements from peers of the given role: it receives
// from senders, or sends cfg.FilePath to receivers.
func Listen(cfg *config.Config, client *transfer.Client, role Role) {
	addr, err := net.ResolveUDPAddr("udp", cfg.MulticastAddress)

	if err != nil {
		fmt.Printf("Failed to resolve %s: %v\n", cfg.MulticastAddress, err)
		return
	}
	conn, err := net.ListenMulticastUDP("udp", nil, addr)
	if err != nil {
		fmt.Printf("Failed to listen on %s: %v\n", cfg.MulticastAddress, err)
		return
	}
	defer conn.Close()

	buf := make([]byte, 1024)
	for {
		n, src, err := conn.ReadFromUDP(buf)
		if err != nil {
			fmt.Printf("Failed to read from %s: %v\n", cfg.MulticastAddress, err)
			continue
		}

		remoteAddr := src.IP.String() + ":" + cfg.Port

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
					err := client.ReceiveFiles(remoteAddr)
					if err != nil {
						fmt.Printf("Receive file from %s error: %s\n", remoteAddr, err)
					}
				}()
			}
		case Receiver:
			if role == Receiver {
				fmt.Printf("Discovered Receiver: %s (%s)\n", deviceName, remoteAddr)
				go func() {
					err := client.SendFiles(cfg.FilePath, fmt.Sprintf("http://%s", remoteAddr))
					if err != nil {
						fmt.Printf("Send file to %s error: %s\n", remoteAddr, err)
					}
				}()
			}
		}
	}
}

// Send announces this instance on the multicast group.
func Send(cfg *config.Config, role Role) {
	addr, err := net.ResolveUDPAddr("udp", cfg.MulticastAddress)
	if err != nil {
		fmt.Printf("Failed to resolve %s: %v\n", cfg.MulticastAddress, err)
		return
	}

	localAddr, err := net.ResolveUDPAddr("udp", cfg.LocalIP+":0")
	if err != nil {
		fmt.Printf("Failed to resolve %s: %v\n", cfg.MulticastAddress, err)
		return
	}

	conn, err := net.DialUDP("udp", localAddr, addr)
	if err != nil {
		fmt.Printf("Failed to dial %s: %v\n", cfg.MulticastAddress, err)
		return
	}
	defer conn.Close()

	message := fmt.Sprintf("%s%s%s", cfg.DeviceName, separator, role)
	conn.Write([]byte(message))
}
