package main

import (
	"runtime"
	"sync"
	"sync/atomic"
	"time"
)

const ReTagChunkSize = 1024

func RunBackfill(total uint64) {
	if config.AI.OpenRouterToken == "" || !config.AI.ReTagEmpty || total == 0 {
		return
	}

	var (
		wg        sync.WaitGroup
		completed atomic.Uint64
		offset    int

		queue = NewQueue(min(4, runtime.NumCPU()))
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

				log.Printf("Verifying tags %.1f%% (%d of %d)\n", percentage, current, total)

				last = current
			}
		}
	})

	for {
		echos, err := database.FindAll(offset, ReTagChunkSize)
		if err != nil {
			break
		}

		for _, echo := range echos {
			if !echo.IsImage() || echo.Tag.Safety != "" {
				completed.Add(1)

				continue
			}

			queue.Work(func() error {
				defer completed.Add(1)

				echo.GenerateTags(true)

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

	log.Printf("Verifying tags 100%% (%d of %d)\n", total, total)
}
