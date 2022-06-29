package modelqueue

import "time"

type OrderQueueEntry struct {
	UserID      string
	OrderNumber int
	OrderStatus string
	RetryCount  int
	Accrual     float64
	LastChecked time.Time
	RetryAfter  time.Duration
}
