package discovery

import (
	"net"
	"slices"
	"sync"
	"testing"
	"time"
)

func TestParse(t *testing.T) {
	t.Parallel()
	from := net.ParseIP("192.168.1.5")
	a, err := parse([]byte("laptop|sender"), from)
	if err != nil || a != (Announcement{Name: "laptop", Role: Sender, IP: "192.168.1.5"}) {
		t.Errorf("parse = %+v, %v", a, err)
	}
	for _, msg := range []string{"", "laptop", "a|b|c", "laptop|admin"} {
		if a, err := parse([]byte(msg), from); err == nil {
			t.Errorf("parse(%q) = %+v, want error", msg, a)
		}
	}
}

type sent struct {
	to  string
	msg string
}

// testNode returns a node whose pulls and sends are recorded instead of
// performed, with a clock the test controls.
func testNode(name string, role Role) (n *node, pulls chan string, sends *[]sent, clock *time.Time) {
	pulls = make(chan string, 10)
	sends = new([]sent)
	clock = new(time.Time)
	*clock = time.Unix(1_000_000, 0)
	var mu sync.Mutex
	n = &node{
		name:    name,
		role:    role,
		port:    53333,
		receive: func(_ Announcement, addr string) { pulls <- addr },
		send: func(to *net.UDPAddr, msg []byte) error {
			mu.Lock()
			defer mu.Unlock()
			*sends = append(*sends, sent{to.String(), string(msg)})
			return nil
		},
		now: func() time.Time { return *clock },
	}
	return n, pulls, sends, clock
}

// expectPulls checks that exactly the wanted pulls happen, in any order:
// each runs on its own goroutine.
func expectPulls(t *testing.T, pulls chan string, want ...string) {
	t.Helper()
	var got []string
	for range want {
		select {
		case addr := <-pulls:
			got = append(got, addr)
		case <-time.After(time.Second):
			t.Fatalf("pulled from %v, want %v", got, want)
		}
	}
	select {
	case addr := <-pulls:
		got = append(got, addr)
	case <-time.After(50 * time.Millisecond):
	}
	slices.Sort(got)
	want = slices.Sorted(slices.Values(want))
	if !slices.Equal(got, want) {
		t.Errorf("pulled from %v, want %v", got, want)
	}
}

func TestReceiverPullsOncePerSender(t *testing.T) {
	t.Parallel()
	n, pulls, sends, clock := testNode("desk", Receiver)

	a := Announcement{Name: "laptop", Role: Sender, IP: "10.0.0.2"}
	n.handle(a)
	n.handle(a) // duplicate, e.g. multicast plus direct reply
	n.handle(Announcement{Name: "phone", Role: Sender, IP: "10.0.0.3"})
	n.handle(Announcement{Name: "other", Role: Receiver, IP: "10.0.0.4"})
	expectPulls(t, pulls, "10.0.0.2:53333", "10.0.0.3:53333")

	*clock = clock.Add(dedupeWindow) // the sender restarted later
	n.handle(a)
	expectPulls(t, pulls, "10.0.0.2:53333")

	if len(*sends) != 0 {
		t.Errorf("receiver sent %v, want nothing", *sends)
	}
}

func TestSenderAnswersReceivers(t *testing.T) {
	t.Parallel()
	n, pulls, sends, _ := testNode("laptop", Sender)

	n.handle(Announcement{Name: "desk", Role: Receiver, IP: "10.0.0.1"})
	n.handle(Announcement{Name: "other", Role: Sender, IP: "10.0.0.9"})

	want := []sent{{"10.0.0.1:53333", "laptop|sender"}}
	if len(*sends) != 1 || (*sends)[0] != want[0] {
		t.Errorf("sent %v, want %v", *sends, want)
	}
	expectPulls(t, pulls) // a sender never pulls
}

// Two instances started at the same moment each hear the other's
// announcement; the file must still be transferred only once.
func TestSimultaneousStartTransfersOnce(t *testing.T) {
	t.Parallel()
	receiver, pulls, _, _ := testNode("desk", Receiver)
	sender, _, replies, _ := testNode("laptop", Sender)

	sender.handle(Announcement{Name: "desk", Role: Receiver, IP: "10.0.0.1"})
	receiver.handle(Announcement{Name: "laptop", Role: Sender, IP: "10.0.0.2"})
	// Deliver the sender's direct reply to the receiver.
	for _, r := range *replies {
		a, err := parse([]byte(r.msg), net.ParseIP("10.0.0.2"))
		if err != nil {
			t.Fatal(err)
		}
		receiver.handle(a)
	}
	expectPulls(t, pulls, "10.0.0.2:53333")
}
