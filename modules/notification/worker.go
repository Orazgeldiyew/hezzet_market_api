package notification

import (
	"context"
	"log"
	"math"
	"time"
)

// Worker consumes jobs from the main queue and sends SMS.
type Worker struct {
	queue    *Queue
	provider SMSProvider
	from     string
	logRepo  *LogRepository
}

func NewWorker(queue *Queue, provider SMSProvider, from string, logRepo *LogRepository) *Worker {
	return &Worker{queue: queue, provider: provider, from: from, logRepo: logRepo}
}

// Start runs the consume loop until ctx is cancelled.
func (w *Worker) Start(ctx context.Context, id int) {
	log.Printf("[worker-%d] started (provider=%s)", id, w.provider.Name())
	for {
		select {
		case <-ctx.Done():
			log.Printf("[worker-%d] shutting down", id)
			return
		default:
		}

		job, err := w.queue.Dequeue(ctx)
		if err != nil {
			if ctx.Err() != nil {
				return
			}
			log.Printf("[worker-%d] dequeue error: %v", id, err)
			time.Sleep(time.Second)
			continue
		}
		if job == nil {
			continue // BLPOP timeout — loop back
		}

		w.processJob(ctx, id, job)
	}
}

func (w *Worker) processJob(ctx context.Context, workerID int, job *SMSJob) {
	// Mark as sending in the audit log.
	w.logRepo.MarkSending(ctx, job.JobID)

	// Rate-limit check
	allowed, err := w.queue.CheckRateLimit(ctx, job.ToPhone)
	if err != nil {
		log.Printf("[worker-%d] rate-limit check error phone=%s: %v", workerID, job.ToPhone, err)
		// On Redis error fall through — best-effort send.
	} else if !allowed {
		log.Printf("[worker-%d] rate-limited phone=%s job=%s", workerID, job.ToPhone, job.JobID)
		job.LastError = "rate_limited"
		if e := w.queue.PushDLQ(ctx, *job); e != nil {
			log.Printf("[worker-%d] dlq push error: %v", workerID, e)
		}
		w.logRepo.MarkRateLimited(ctx, job.JobID)
		return
	}

	// Send SMS with a bounded timeout.
	sendCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
	err = w.provider.Send(sendCtx, job.ToPhone, w.from, job.Message)
	cancel()

	if err == nil {
		log.Printf("[worker-%d] sent job=%s type=%s to=%s", workerID, job.JobID, job.Type, job.ToPhone)
		w.logRepo.MarkSent(ctx, job.JobID)
		return
	}

	// ── Failure path ──
	log.Printf("[worker-%d] send failed job=%s attempt=%d/%d err=%v",
		workerID, job.JobID, job.Attempt+1, job.MaxAttempts, err)

	job.Attempt++
	job.LastError = err.Error()

	if job.Attempt >= job.MaxAttempts {
		log.Printf("[worker-%d] max attempts reached, moving to DLQ job=%s", workerID, job.JobID)
		if e := w.queue.PushDLQ(ctx, *job); e != nil {
			log.Printf("[worker-%d] dlq push error: %v", workerID, e)
		}
		w.logRepo.MarkDLQ(ctx, job.JobID, job.LastError)
		return
	}

	// Exponential backoff: 2^attempt * 5s → 10s, 20s, 40s, …
	backoff := time.Duration(math.Pow(2, float64(job.Attempt))) * 5 * time.Second
	if err := w.queue.EnqueueDelayed(ctx, *job, time.Now().Add(backoff)); err != nil {
		log.Printf("[worker-%d] delayed enqueue error: %v", workerID, err)
	}
	w.logRepo.MarkRetrying(ctx, job.JobID, job.Attempt, job.LastError)
}

// RunDelayedPromoter periodically moves ready delayed jobs back to the main queue.
func RunDelayedPromoter(ctx context.Context, queue *Queue) {
	ticker := time.NewTicker(3 * time.Second)
	defer ticker.Stop()
	log.Println("[delayed-promoter] started")

	for {
		select {
		case <-ctx.Done():
			log.Println("[delayed-promoter] shutting down")
			return
		case <-ticker.C:
			n, err := queue.PromoteDelayed(ctx)
			if err != nil {
				if ctx.Err() != nil {
					return
				}
				log.Printf("[delayed-promoter] error: %v", err)
			} else if n > 0 {
				log.Printf("[delayed-promoter] promoted %d jobs", n)
			}
		}
	}
}

// StartWorkers launches n worker goroutines + the delayed-job promoter.
// All goroutines respect ctx cancellation for graceful shutdown.
func StartWorkers(ctx context.Context, n int, queue *Queue, provider SMSProvider, from string, logRepo *LogRepository) {
	for i := 0; i < n; i++ {
		w := NewWorker(queue, provider, from, logRepo)
		go w.Start(ctx, i)
	}
	go RunDelayedPromoter(ctx, queue)
}
