// Copyright The Shipwright Contributors
//
// SPDX-License-Identifier: Apache-2.0

package util

import (
	"fmt"
	"net"
	"time"
)

// TestConnection tries to establish a connection to a provided host using a 5 seconds timeout.
func TestConnection(hostname string, port int, retries int) bool {
	host := fmt.Sprintf("%s:%d", hostname, port)

	dialer := &net.Dialer{
		Timeout: 5 * time.Second,
	}

	for i := 0; i <= retries; i++ {
		conn, _ := dialer.Dial("tcp", host)
		if conn != nil {
			_ = conn.Close()
			return true
		}
	}

	return false
}
