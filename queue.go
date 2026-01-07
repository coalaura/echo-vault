package main

import (
	"sync"
)

type QueueJob func() error

type Queue struct {
	wg   sync.WaitGroup
	jobs chan QueueJob
	errs chan error
	once sync.Once
}

func NewQueue(worker int) *Queue {
	q := &Queue{
		jobs: make(chan QueueJob),
		errs: make(chan error, 1),
	}

	for i := 0; i < worker; i++ {
		q.wg.Add(1)

		go func() {
			defer q.wg.Done()

			for job := range q.jobs {
				if err := job(); err != nil {
					select {
					case q.errs <- err:
					default:
					}
				}
			}
		}()
	}

	return q
}

func (q *Queue) Work(job QueueJob) error {
	select {
	case err := <-q.errs:
		return err
	case q.jobs <- job:
		return nil
	}
}

func (q *Queue) Wait() error {
	q.once.Do(func() {
		close(q.jobs)
	})

	q.wg.Wait()

	close(q.errs)

	return <-q.errs
}
