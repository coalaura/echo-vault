package main

import (
	"context"
	"sync"
	"sync/atomic"
	"time"
)

const ReTagChunkSize = 1024

func (d *EchoDatabase) Backfill(total uint64) {
	if config.AI.OpenRouterToken == "" || !config.AI.ReTagEmpty || total == 0 {
		return
	}

	var (
		wg        sync.WaitGroup
		completed atomic.Uint64
		offset    int

		totalCostMx sync.RWMutex
		totalCost   float64

		queue = NewQueue(4)
		done  = make(chan bool)
	)

	log.Printf("Verifying tags 0%% (0 of %d)\n", total)

	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	wg.Go(func() {
		var (
			last   uint64
			totalF = float64(total)
		)

		for {
			select {
			case <-done:
				return
			case <-ticker.C:
				current := completed.Load()

				if current == last {
					continue
				}

				percentage := float64(current) / totalF * 100

				totalCostMx.RLock()
				cost := totalCost
				totalCostMx.RUnlock()

				log.Printf("Verifying tags %.1f%% (%d of %d, $%f)\n", percentage, current, total, cost)

				last = current
			}
		}
	})

	for {
		echos, err := d.FindAll(context.Background(), offset, ReTagChunkSize)
		if err != nil {
			break
		}

		for _, echo := range echos {
			if !echo.IsImage() || (echo.Description != "" && vector.Has(echo.Hash)) {
				completed.Add(1)

				continue
			}

			if !echo.Exists() {
				completed.Add(1)

				continue
			}

			queue.Work(func() error {
				defer completed.Add(1)

				cost := echo.GenerateTags(context.Background(), true, true)
				if cost > 0 {
					totalCostMx.Lock()
					totalCost += cost
					totalCostMx.Unlock()
				}

				return nil
			})
		}

		if len(echos) < ReTagChunkSize {
			break
		}

		offset += ReTagChunkSize
	}

	queue.Wait()

	close(done)

	wg.Wait()

	err := d.Optimize(10)
	if err != nil {
		log.Warnf("Backfill optimization failed: %v\n", err)
	}

	log.Printf("Verifying tags 100%% (%d of %d)\n", total, total)
}
