package main

import (
	"log"
	"sync"
	"sync/atomic"
	"time"

	psnet "github.com/shirou/gopsutil/v3/net"
)

type BandwidthMonitor struct {
	mode         string
	bytesPerSec  int64
	maxBandwidth int64
	ourBytes     int64
	numParts     int32

	mu     sync.Mutex
	stopCh chan struct{}
}

func NewBandwidthMonitor() *BandwidthMonitor {
	bm := &BandwidthMonitor{
		bytesPerSec:  0,
		maxBandwidth: 0,
		mode:         "auto",
		numParts:     4,
		stopCh:       make(chan struct{}),
	}
	return bm
}

func (bm *BandwidthMonitor) Start() {
	go func() {
		ticker := time.NewTicker(2 * time.Second)
		defer ticker.Stop()

		lastIO, err := psnet.IOCounters(false)
		if err != nil {
			log.Println("BandwidthMonitor: failed to read network stats:", err)
			return
		}

		lastRecv := lastIO[0].BytesRecv
		lastTime := time.Now()

		atomic.StoreInt64(&bm.ourBytes, 0)
		lastOurBytes := int64(0)

		for {
			select {
			case <-bm.stopCh:
				return
			case <-ticker.C:
				currentIO, err := psnet.IOCounters(false)
				if err != nil {
					continue
				}

				now := time.Now()
				elapsed := now.Sub(lastTime).Seconds()
				if elapsed <= 0 {
					continue
				}

				currentRecv := currentIO[0].BytesRecv
				totalSpeed := float64(currentRecv-lastRecv) / elapsed

				currentOurBytes := atomic.LoadInt64(&bm.ourBytes)
				ourSpeed := float64(currentOurBytes-lastOurBytes) / elapsed

				otherUsage := totalSpeed - ourSpeed
				if otherUsage < 0 {
					otherUsage = 0
				}

				maxBW := atomic.LoadInt64(&bm.maxBandwidth)
				if ourSpeed > float64(maxBW) {
					atomic.StoreInt64(&bm.maxBandwidth, int64(ourSpeed))
					maxBW = int64(ourSpeed)
				} else if maxBW > 0 {
					decayed := int64(float64(maxBW) * 0.98)
					atomic.StoreInt64(&bm.maxBandwidth, decayed)
					maxBW = decayed
				}

				bm.mu.Lock()
				mode := bm.mode
				bm.mu.Unlock()

				var newLimit int64

				switch mode {
				case "snail":
					newLimit = maxBW * 30 / 100
					if newLimit < 50*1024 {
						newLimit = 50 * 1024
					}
				case "auto":
					available := float64(maxBW) - otherUsage
					if available < float64(maxBW)*0.3 {
						available = float64(maxBW) * 0.3
					}
					newLimit = int64(available)
					if newLimit < 50*1024 {
						newLimit = 50 * 1024
					}

					if maxBW == 0 {
						newLimit = 0
					}
				case "turbo":
					newLimit = 0
				}

				atomic.StoreInt64(&bm.bytesPerSec, newLimit)

				lastRecv = currentRecv
				lastOurBytes = currentOurBytes
				lastTime = now
			}
		}
	}()
}

func (bm *BandwidthMonitor) Stop() {
	close(bm.stopCh)
}

func (bm *BandwidthMonitor) SetMode(mode string) {
	bm.mu.Lock()
	bm.mode = mode
	bm.mu.Unlock()

	if mode == "turbo" {
		atomic.StoreInt64(&bm.bytesPerSec, 0)
	}

	log.Println("Bandwidth mode set to:", mode)
}

func (bm *BandwidthMonitor) Wait(n int) {
	limit := atomic.LoadInt64(&bm.bytesPerSec)
	if limit <= 0 {
		return
	}

	parts := int64(atomic.LoadInt32(&bm.numParts))
	if parts < 1 {
		parts = 1
	}
	perPartLimit := limit / parts
	sleepDuration := time.Duration(float64(n) / float64(perPartLimit) * float64(time.Second))
	time.Sleep(sleepDuration)
}

func (bm *BandwidthMonitor) AddBytes(n int) {
	atomic.AddInt64(&bm.ourBytes, int64(n))
}

func (bm *BandwidthMonitor) SetParts(n int) {
	atomic.StoreInt32(&bm.numParts, int32(n))
}
