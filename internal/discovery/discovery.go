// Package discovery finds other st instances on the LAN.
//
// Every instance multicasts an announcement, "<device name>|<role>", when it
// starts. Only receivers start transfers: a receiver that hears a sender
// pulls from it, and a sender that hears a receiver replies with its own
// announcement, sent straight to that receiver, so the receiver pulls. With
// just one side acting, two instances started at the same moment (which each
// hear the other) still transfer only once.
package discovery

import (
	"fmt"
	"net"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/chyok/st/internal/config"
)

const separator = "|"

type Role string

const (
	Sender   Role = "sender"
	Receiver Role = "receiver"
)

// Announcement is what a peer said about itself, and where it came from.
type Announcement struct {
	Name string
	Role Role
	IP   string
}

func encode(name string, role Role) []byte {
	return []byte(name + separator + string(role))
}

func parse(msg []byte, from net.IP) (Announcement, error) {
	parts := strings.Split(string(msg), separator)
	if len(parts) != 2 {
		return Announcement{}, fmt.Errorf("malformed announcement %q", msg)
	}
	role := Role(parts[1])
	if role != Sender && role != Receiver {
		return Announcement{}, fmt.Errorf("unknown role in announcement %q", msg)
	}
	return Announcement{Name: parts[0], Role: role, IP: from.String()}, nil
}

// dedupeWindow is how long a receiver ignores further announcements from a
// sender it has just started pulling from.
const dedupeWindow = 10 * time.Second

// node holds the discovery state of one st instance.
type node struct {
	name string
	role Role
	port int // HTTP and discovery port, the same on every peer

	// receive pulls from the sender at addr; called on its own goroutine.
	receive func(peer Announcement, addr string)
	// send delivers an announcement to a group or a peer.
	send func(to *net.UDPAddr, msg []byte) error
	now  func() time.Time

	mu     sync.Mutex
	recent map[string]time.Time // sender IP -> when we last started pulling
}

func (n *node) handle(a Announcement) {
	switch {
	case n.role == Receiver && a.Role == Sender:
		if !n.claim(a.IP) {
			return
		}
		addr := net.JoinHostPort(a.IP, strconv.Itoa(n.port))
		fmt.Printf("Discovered Sender: %s (%s)\n", a.Name, addr)
		go n.receive(a, addr)
	case n.role == Sender && a.Role == Receiver:
		fmt.Printf("Discovered Receiver: %s (%s)\n", a.Name, a.IP)
		to := &net.UDPAddr{IP: net.ParseIP(a.IP), Port: n.port}
		if err := n.send(to, encode(n.name, n.role)); err != nil {
			fmt.Printf("Failed to answer %s: %v\n", a.IP, err)
		}
	}
}

// claim reports whether we should pull from the sender at ip, and if so
// remembers it so a duplicate announcement right after is ignored.
func (n *node) claim(ip string) bool {
	n.mu.Lock()
	defer n.mu.Unlock()
	now := n.now()
	if last, ok := n.recent[ip]; ok && now.Sub(last) < dedupeWindow {
		return false
	}
	if n.recent == nil {
		n.recent = make(map[string]time.Time)
	}
	n.recent[ip] = now
	return true
}

// Run announces this instance and handles other instances' announcements
// until listening fails. A receiver calls receive for each sender it finds.
func Run(cfg *config.Config, role Role, receive func(peer Announcement, addr string)) error {
	group, err := net.ResolveUDPAddr("udp4", cfg.MulticastAddr())
	if err != nil {
		return err
	}
	// Listen before announcing, so no reply can arrive before we listen.
	// The socket is bound to the port on all addresses, so it also gets the
	// replies senders address to us directly.
	listener, err := net.ListenMulticastUDP("udp4", nil, group)
	if err != nil {
		return err
	}
	defer listener.Close()
	out, err := net.ListenUDP("udp4", &net.UDPAddr{IP: net.ParseIP(cfg.LocalIP)})
	if err != nil {
		return err
	}
	defer out.Close()

	n := &node{
		name:    cfg.DeviceName,
		role:    role,
		port:    group.Port,
		receive: receive,
		send: func(to *net.UDPAddr, msg []byte) error {
			_, err := out.WriteToUDP(msg, to)
			return err
		},
		now: time.Now,
	}
	if err := n.send(group, encode(n.name, n.role)); err != nil {
		fmt.Printf("Failed to announce on %s: %v\n", group, err)
	}

	buf := make([]byte, 1024)
	for {
		size, from, err := listener.ReadFromUDP(buf)
		if err != nil {
			return err
		}
		a, err := parse(buf[:size], from.IP)
		if err != nil {
			fmt.Printf("Ignoring message from %s: %v\n", from.IP, err)
			continue
		}
		n.handle(a)
	}
}
