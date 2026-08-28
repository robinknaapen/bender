//go:build linux

package linux

import (
	"image"
	"log"
	"sync"

	"github.com/godbus/dbus/v5"

	"github.com/pietjan/bender/internal/platform/appicon"
)

// notifier speaks org.freedesktop.Notifications. One toast pends at a
// time (each Notify replaces the last), matching the Windows balloon
// semantics the app's last-notifier tracking assumes. All bus calls run
// off the UI thread: a hung daemon (misconfigured activation) must
// never stall the app.
type notifier struct {
	backend *Backend
	conn    *dbus.Conn
	mu      sync.Mutex
	lastID  uint32
	onClick func()
}

func newNotifier(b *Backend, conn *dbus.Conn) *notifier {
	n := &notifier{backend: b, conn: conn}
	if err := conn.AddMatchSignal(
		dbus.WithMatchInterface("org.freedesktop.Notifications"),
		dbus.WithMatchMember("ActionInvoked"),
	); err != nil {
		log.Printf("linux: notification actions unavailable: %v", err)
		return n
	}
	ch := make(chan *dbus.Signal, 8)
	conn.Signal(ch)
	go func() {
		for sig := range ch {
			log.Printf("linux: dbus signal %s %v", sig.Name, sig.Body)
			if sig.Name != "org.freedesktop.Notifications.ActionInvoked" || len(sig.Body) < 2 {
				continue
			}
			id, _ := sig.Body[0].(uint32)
			n.mu.Lock()
			match := id == n.lastID
			n.mu.Unlock()
			if match && n.onClick != nil {
				// DBus goroutine → UI thread.
				b.Dispatch(n.onClick)
			}
		}
	}()
	return n
}

func (n *notifier) notify(title, body string, icon image.Image) {
	if n.conn == nil {
		return
	}
	if icon == nil {
		if img, err := appicon.Decode(); err == nil {
			icon = img
		}
	}
	hints := map[string]dbus.Variant{
		"urgency": dbus.MakeVariant(byte(1)),
	}
	if icon != nil {
		w, h, stride, alpha, bits, channels, pix := notificationImage(icon, 64)
		hints["image-data"] = dbus.MakeVariant(struct {
			Width, Height, Stride int32
			HasAlpha              bool
			Bits, Channels        int32
			Pixels                []byte
		}{w, h, stride, alpha, bits, channels, pix})
	}
	// Off the UI thread: a synchronous Call would freeze the app for the
	// full DBus timeout whenever the daemon hangs.
	go func() {
		n.mu.Lock()
		replaces := n.lastID
		n.mu.Unlock()
		obj := n.conn.Object("org.freedesktop.Notifications", "/org/freedesktop/Notifications")
		call := obj.Call("org.freedesktop.Notifications.Notify", 0,
			"Bender", replaces, "", title, body,
			[]string{"default", "Open"}, hints, int32(-1))
		if call.Err != nil {
			log.Printf("linux: notify: %v", call.Err)
			return
		}
		var id uint32
		if call.Store(&id) == nil {
			n.mu.Lock()
			n.lastID = id
			n.mu.Unlock()
		}
	}()
}
