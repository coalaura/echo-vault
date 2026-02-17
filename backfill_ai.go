//go:build ai

package main

import (
	"context"
	"sync"
	"sync/atomic"
	"time"
)

const BackfillChunkSize = 512

func (d *EchoDatabase) Backfill(total uint64) {
	if vector == nil || total == 0 {
		return
	}

	var (
		wg        sync.WaitGroup
		completed atomic.Uint64
		offset    int

		queue = NewQueue(2)
		done  = make(chan bool)
	)

	log.Printf("Indexing images 0%% (0 of %d)\n", total)

	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	wg.Go(func() {
		var last uint64

		totalF := float64(total)

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

				log.Printf("Indexing images %.1f%% (%d of %d)\n", percentage, current, total)

				last = current
			}
		}
	})

	for {
		echos, err := d.FindAll(context.Background(), offset, BackfillChunkSize)
		if err != nil {
			log.Warnf("Backfill read failed: %v\n", err)

			break
		}

		for _, echo := range echos {
			if !echo.IsImage() || echo.Animated || !echo.Exists() {
				completed.Add(1)

				continue
			}

			if vector.Has(echo.Hash) {
				completed.Add(1)

				continue
			}

			queue.Work(func() error {
				defer completed.Add(1)

				err := vector.IndexImage(context.Background(), echo.Hash, echo.Storage())
				if err != nil {
					log.Warnf("[%s] Index failed: %v\n", echo.Hash, err)
				}

				return nil
			})
		}

		if len(echos) < BackfillChunkSize {
			break
		}

		offset += BackfillChunkSize
	}

	queue.Wait()
	close(done)
	wg.Wait()

	log.Printf("Indexing images 100%% (%d of %d)\n", total, total)
}
