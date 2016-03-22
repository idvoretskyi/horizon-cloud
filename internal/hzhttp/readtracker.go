package hzhttp

import (
	"io"
	"time"
)

type transferStats struct {
	Duration time.Duration
	Bytes    int64
}

func (t transferStats) MarshalJSON() ([]byte, error) {
	panic("transferStats should never be marshalled to JSON. call .Wire().")
}

func (t transferStats) Wire() transferStatsWire {
	return transferStatsWire{
		Duration: t.Duration.Seconds(),
		Bytes:    t.Bytes,
	}
}

type transferStatsWire struct {
	Duration float64 `json:"duration"`
	Bytes    int64   `json:"bytes"`
}

type readTracker struct {
	io.ReadCloser
	Stats transferStats
}

func (rt *readTracker) Read(p []byte) (int, error) {
	start := time.Now()
	n, err := rt.ReadCloser.Read(p)
	rt.Stats.Duration += time.Now().Sub(start)
	rt.Stats.Bytes += int64(n)
	return n, err
}
