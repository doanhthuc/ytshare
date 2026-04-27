// Package jobs implements an in-process background worker pool.
package jobs

import (
	"context"
	"sync"
	"time"

	"go.uber.org/zap"
)

type Job func(ctx context.Context) error

type Worker struct {
	queue   chan Job
	wg      sync.WaitGroup
	log     *zap.Logger
	timeout time.Duration
}

func NewWorker(poolSize, queueSize int, log *zap.Logger) *Worker {
	if poolSize <= 0 {
		poolSize = 4
	}
	if queueSize <= 0 {
		queueSize = 256
	}
	w := &Worker{
		queue:   make(chan Job, queueSize),
		log:     log,
		timeout: 30 * time.Second,
	}
	for i := 0; i < poolSize; i++ {
		w.wg.Add(1)
		go w.run(i)
	}
	return w
}

// Submit enqueues a job; blocks if the queue is full.
func (w *Worker) Submit(j Job) {
	w.queue <- j
}

// Stop drains the queue and waits for in-flight jobs to finish.
func (w *Worker) Stop() {
	close(w.queue)
	w.wg.Wait()
}

func (w *Worker) run(id int) {
	defer w.wg.Done()
	logger := w.log.With(zap.Int("worker", id))
	for job := range w.queue {
		ctx, cancel := context.WithTimeout(context.Background(), w.timeout)
		if err := job(ctx); err != nil {
			logger.Error("job_failed", zap.Error(err))
		}
		cancel()
	}
}
