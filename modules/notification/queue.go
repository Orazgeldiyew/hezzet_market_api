package notification

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// Redis key prefixes / names.
const (
	keyQueue   = "sms:queue"
	keyDelayed = "sms:queue:delayed"
	keyDLQ     = "sms:dlq"
	keyDedup   = "sms:dedup:"
	keyRate    = "sms:rate:"
)

// SMSJob is the unit of work that travels through the queue.
type SMSJob struct {
	JobID       string `json:"job_id"`
	Type        string `json:"type"` // "customer_order_created" | "admin_low_stock" | …
	ToPhone     string `json:"to_phone"`
	Message     string `json:"message"`
	CreatedAt   int64  `json:"created_at"`
	Attempt     int    `json:"attempt"`
	MaxAttempts int    `json:"max_attempts"`
	DedupKey    string `json:"dedup_key"`
	LastError   string `json:"last_error,omitempty"`
}

// Queue wraps a Redis client and provides enqueue / dequeue / dedup / rate-limit ops.
type Queue struct {
	rdb              *redis.Client
	rateLimitPerHour int
}

func NewQueue(rdb *redis.Client, rateLimitPerHour int) *Queue {
	return &Queue{rdb: rdb, rateLimitPerHour: rateLimitPerHour}
}

// ---------- helpers ----------

func generateUUID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	b[6] = (b[6] & 0x0f) | 0x40 // version 4
	b[8] = (b[8] & 0x3f) | 0x80 // variant 10
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}

// ---------- Enqueue ----------

// Enqueue adds a job to the main queue.
// Returns (true, nil) if enqueued, (false, nil) if deduplicated / skipped.
func (q *Queue) Enqueue(ctx context.Context, job SMSJob, dedupTTL time.Duration) (bool, error) {
	// Idempotency: SETNX dedup key with TTL — primary dedup gate.
	if job.DedupKey != "" {
		ok, err := q.rdb.SetNX(ctx, keyDedup+job.DedupKey, "1", dedupTTL).Result()
		if err != nil {
			return false, fmt.Errorf("dedup setnx: %w", err)
		}
		if !ok {
			return false, nil // already enqueued within TTL window
		}
	}

	if job.JobID == "" {
		job.JobID = generateUUID()
	}
	if job.CreatedAt == 0 {
		job.CreatedAt = time.Now().Unix()
	}
	if job.MaxAttempts == 0 {
		job.MaxAttempts = 3
	}

	data, err := json.Marshal(job)
	if err != nil {
		return false, fmt.Errorf("marshal job: %w", err)
	}
	if err := q.rdb.RPush(ctx, keyQueue, data).Err(); err != nil {
		return false, fmt.Errorf("rpush: %w", err)
	}
	return true, nil
}

// ---------- Dequeue ----------

// Dequeue blocks up to 2 s for a job. Returns (nil, nil) on timeout.
func (q *Queue) Dequeue(ctx context.Context) (*SMSJob, error) {
	res, err := q.rdb.BLPop(ctx, 2*time.Second, keyQueue).Result()
	if err != nil {
		if err == redis.Nil {
			return nil, nil
		}
		return nil, fmt.Errorf("blpop: %w", err)
	}
	if len(res) < 2 {
		return nil, nil
	}

	var job SMSJob
	if err := json.Unmarshal([]byte(res[1]), &job); err != nil {
		return nil, fmt.Errorf("unmarshal job: %w", err)
	}
	return &job, nil
}

// ---------- Delayed / retry ----------

// EnqueueDelayed adds a job to the delayed sorted set (score = executeAt unix).
func (q *Queue) EnqueueDelayed(ctx context.Context, job SMSJob, executeAt time.Time) error {
	data, err := json.Marshal(job)
	if err != nil {
		return fmt.Errorf("marshal delayed: %w", err)
	}
	return q.rdb.ZAdd(ctx, keyDelayed, redis.Z{
		Score:  float64(executeAt.Unix()),
		Member: string(data),
	}).Err()
}

// PromoteDelayed moves jobs whose score <= now from the delayed set to the main queue.
// Returns the number of promoted jobs.
func (q *Queue) PromoteDelayed(ctx context.Context) (int, error) {
	now := fmt.Sprintf("%d", time.Now().Unix())

	members, err := q.rdb.ZRangeByScore(ctx, keyDelayed, &redis.ZRangeBy{
		Min: "-inf",
		Max: now,
	}).Result()
	if err != nil {
		return 0, fmt.Errorf("zrangebyscore: %w", err)
	}
	if len(members) == 0 {
		return 0, nil
	}

	promoted := 0
	for _, m := range members {
		// Remove first — if another worker already removed it, skip.
		removed, err := q.rdb.ZRem(ctx, keyDelayed, m).Result()
		if err != nil {
			return promoted, fmt.Errorf("zrem: %w", err)
		}
		if removed == 0 {
			continue
		}
		if err := q.rdb.RPush(ctx, keyQueue, m).Err(); err != nil {
			return promoted, fmt.Errorf("rpush promoted: %w", err)
		}
		promoted++
	}
	return promoted, nil
}

// ---------- DLQ ----------

// PushDLQ moves a permanently failed job to the dead-letter queue.
func (q *Queue) PushDLQ(ctx context.Context, job SMSJob) error {
	data, err := json.Marshal(job)
	if err != nil {
		return fmt.Errorf("marshal dlq: %w", err)
	}
	return q.rdb.RPush(ctx, keyDLQ, data).Err()
}

// ---------- Rate limiting ----------

// CheckRateLimit returns true if the phone is still within the hourly limit.
func (q *Queue) CheckRateLimit(ctx context.Context, phone string) (bool, error) {
	if q.rateLimitPerHour <= 0 {
		return true, nil
	}

	key := keyRate + phone + ":" + time.Now().UTC().Format("2006010215")

	count, err := q.rdb.Incr(ctx, key).Result()
	if err != nil {
		return false, fmt.Errorf("rate incr: %w", err)
	}
	if count == 1 {
		// First increment — set expiry so the counter auto-cleans.
		q.rdb.Expire(ctx, key, 2*time.Hour)
	}
	return count <= int64(q.rateLimitPerHour), nil
}
