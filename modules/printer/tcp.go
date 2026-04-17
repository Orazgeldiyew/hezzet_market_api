package printer

import (
	"fmt"
	"net"
	"strconv"
	"time"
)

// SendToTCP opens a TCP connection to the printer and writes the given bytes.
// Returns an error if connection fails or write fails.
func SendToTCP(ip string, port int, data []byte) error {
	addr := net.JoinHostPort(ip, strconv.Itoa(port))
	conn, err := net.DialTimeout("tcp", addr, 5*time.Second)
	if err != nil {
		return fmt.Errorf("connect %s: %w", addr, err)
	}
	defer conn.Close()

	// Deadline for write to avoid hanging on unresponsive printers
	_ = conn.SetWriteDeadline(time.Now().Add(5 * time.Second))

	_, err = conn.Write(data)
	if err != nil {
		return fmt.Errorf("write to %s: %w", addr, err)
	}
	return nil
}
