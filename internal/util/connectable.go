package util

import (
	"fmt"
	"math"
	"math/rand"
	"net"
	"time"
)

var (
	initialDelay = time.Second
	maxDelay     = time.Second * 10
)

// TimedOutError is returned from WaitConnectable when the connection attempts
// time out.
type TimedOutError struct {
	Network, Address string
	Tries            int
	Timeout          time.Duration
}

func (e TimedOutError) Error() string {
	return fmt.Sprintf("timed out connecting to %v %v after %v tries in %v",
		e.Network, e.Address, e.Tries, e.Timeout)
}

// WaitConnectable waits until the given address is connectable (that is,
// net.Dial returns a connection.) It will retry if failures occur within the
// timeout duration given.
//
// On success, WaitConnectable returns nil. On failure, it returns a
// TimedOutError.
func WaitConnectable(network, address string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)

	tries := 0
	for deadline.Sub(time.Now()) > 0 {
		// randomized exponential backoff
		delay := time.Duration(
			(0.5 + 0.5*rand.Float64()) *
				math.Pow(1.5, float64(tries)) * float64(initialDelay))

		if delay > maxDelay {
			delay = maxDelay
		}

		// don't go over the deadline
		delayEnd := time.Now().Add(delay)
		if delayEnd.After(deadline) {
			delay = deadline.Sub(time.Now())
			delayEnd = time.Now().Add(delay)
		}

		tries++
		c, err := net.DialTimeout(network, address, delay)
		if err == nil {
			_ = c.Close()
			return nil
		}

		// If the DialTimeout returned early (for example, because the
		// connection was refused), we still want to wait the full delay time
		// before trying again.
		extra := delayEnd.Sub(time.Now())
		if extra > 0 {
			time.Sleep(extra)
		}
	}

	return TimedOutError{
		Network: network,
		Address: address,
		Tries:   tries,
		Timeout: timeout,
	}
}
