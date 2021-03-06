package sdnotify

import "errors"
import "net"
import "sync"
import "os"

// Error returned if no systemd notify protocol socket can be found.
//
// This is an indication that the service is not running under systemd or
// Type=notify is not set in the systemd unit file.
var ErrNoSocket = errors.New("No socket")

// sdNotifySocket
var sdNotifyMutex sync.Mutex
var sdNotifySocket *net.UnixConn
var sdNotifyInited bool

// Send sends a message to the init daemon. It is common to ignore the error.
//
// Taken from coreos/go-systemd/daemon. Since that code closes the socket
// after each call it won't work in a chroot. It is customized here to keep
// the socket open.
func Send(state string) error {
	sdNotifyMutex.Lock()
	defer sdNotifyMutex.Unlock()

	if !sdNotifyInited {
		sdNotifyInited = true

		socketAddr := &net.UnixAddr{
			Name: os.Getenv("NOTIFY_SOCKET"),
			Net:  "unixgram",
		}

		if socketAddr.Name == "" {
			return ErrNoSocket
		}

		conn, err := net.DialUnix(socketAddr.Net, nil, socketAddr)
		if err != nil {
			return err
		}

		sdNotifySocket = conn
	}

	if sdNotifySocket == nil {
		return ErrNoSocket
	}

	_, err := sdNotifySocket.Write([]byte(state))
	return err
}
