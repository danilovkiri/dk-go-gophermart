// Package broker provides parallelization and queueing functionality for data processing.

package broker

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"sync"
	"time"

	"github.com/danilovkiri/dk-go-gophermart/internal/client"
	"github.com/danilovkiri/dk-go-gophermart/internal/models/modeldto"
	"github.com/danilovkiri/dk-go-gophermart/internal/models/modelqueue"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"golang.org/x/sync/errgroup"
)

// Broker defines attributes of a struct available to its methods.
type Broker struct {
	ctx           context.Context
	log           *zerolog.Logger
	queueIn       chan modelqueue.OrderQueueEntry
	queueOut      chan modelqueue.OrderQueueEntry
	wg            *sync.WaitGroup
	accrualClient *client.Client
	workerNumber  int
	retryNumber   int
}

// GetAccrualWorker defines attributes of a struct available to its methods.
type GetAccrualWorker struct {
	ID            int
	ctx           context.Context
	log           *zerolog.Logger
	queueIn       chan modelqueue.OrderQueueEntry
	queueOut      chan modelqueue.OrderQueueEntry
	accrualClient *client.Client
	retryNumber   int
}

// InitBroker initializes a queue management service.
func InitBroker(ctx context.Context, queueIn chan modelqueue.OrderQueueEntry, queueOut chan modelqueue.OrderQueueEntry, log *zerolog.Logger, wg *sync.WaitGroup, accrualClient *client.Client, nWorkers int, nRetries int) *Broker {
	broker := Broker{
		ctx:           ctx,
		log:           log,
		queueIn:       queueIn,
		queueOut:      queueOut,
		wg:            wg,
		accrualClient: accrualClient,
		workerNumber:  nWorkers,
		retryNumber:   nRetries,
	}
	return &broker
}

// ListenAndProcess starts queue management and defines its logic.
func (b *Broker) ListenAndProcess() {
	b.wg.Add(1)
	go func() {
		log.Info().Msg("started listening to queue for unprocessed orders")
		defer b.wg.Done()
		g, _ := errgroup.WithContext(b.ctx)
		for i := 0; i < b.workerNumber+1; i++ {
			w := &GetAccrualWorker{ID: i, ctx: b.ctx, queueIn: b.queueIn, queueOut: b.queueOut, log: b.log, accrualClient: b.accrualClient, retryNumber: b.retryNumber}
			g.Go(w.processAsync)
		}
		<-b.ctx.Done()
		close(b.queueIn)
		log.Info().Msg("closed queue for unprocessed orders")
		close(b.queueOut)
		log.Info().Msg("closed queue for processed orders")
		err := g.Wait()
		if err != nil {
			b.log.Fatal().Err(err).Msg("closing errgroup failed")
		}
		log.Info().Msg("stopped listening to queue for unprocessed orders")
	}()
}

// processAsync processes data from queue and manages its usage.
func (w *GetAccrualWorker) processAsync() error {
	for record := range w.queueIn {
		// check retry-after timeout, if nonzero and not finished - put back to queue
		if record.RetryAfter != 0 && time.Since(record.LastChecked) < record.RetryAfter {
			w.queueIn <- record
			continue
		}

		// wait for at least 10 seconds before querying the same order again
		// stop waiting upon ctx.Done()
		for time.Since(record.LastChecked) < 10*time.Second {
			select {
			case <-w.ctx.Done():
				return nil
			default:

			}
		}

		// retrieve status and accrual updates via client
		statusMap := map[string]string{
			"INVALID":    "INVALID",
			"PROCESSED":  "PROCESSED",
			"PROCESSING": "PROCESSING",
			"REGISTERED": "NEW",
		}
		resp, err := w.accrualClient.GetAccrual(w.ctx, record.OrderNumber)
		if err != nil || (resp != nil && (resp.StatusCode() != 429 && resp.StatusCode() != 200)) {
			if record.RetryCount >= w.retryNumber {
				// abandon processing if w.retryNumber retries were unsuccessfully performed
				w.log.Warn().Msg(fmt.Sprintf("WID %v, order %v — abandoning due to retry limit exceeding", w.ID, record.OrderNumber))
				finalRecord := modelqueue.OrderQueueEntry{
					UserID:      record.UserID,
					OrderNumber: record.OrderNumber,
					OrderStatus: record.OrderStatus,
					Accrual:     record.Accrual,
				}
				w.queueOut <- finalRecord
				continue
			} else {
				// put back to queue if querying resulted in error, increment RetryCount, set LastChecked to time.Now()
				w.log.Warn().Msg(fmt.Sprintf("WID %v, order %v — could not process, sending back to queue", w.ID, record.OrderNumber))
				record.RetryCount += 1
				record.LastChecked = time.Now()
				w.queueIn <- record
				continue
			}
		}

		if resp.StatusCode() == 429 {
			seconds, _ := strconv.Atoi(resp.Header().Get("Retry-After"))
			w.log.Warn().Msg(fmt.Sprintf("WID %v, order %v — request delay by %v, sending back to queue", w.ID, record.OrderNumber, seconds))
			retryAfter := time.Duration(int(time.Second) * seconds)
			record.LastChecked = time.Now()
			record.RetryAfter = retryAfter
			w.queueIn <- record
			continue
		}

		var accrualResponse modeldto.AccrualResponse
		err = json.Unmarshal(resp.Body(), &accrualResponse)
		if err != nil {
			w.log.Err(err).Msg(fmt.Sprintf("WID %v, order %v — could not parse response body", w.ID, record.OrderNumber))
			// put back to queue if querying resulted in error, increment RetryCount, set LastChecked to time.Now()
			w.log.Warn().Msg(fmt.Sprintf("WID %v, order %v — could not process, sending back to queue", w.ID, record.OrderNumber))
			record.RetryCount += 1
			record.LastChecked = time.Now()
			record.RetryAfter = 0
			w.queueIn <- record
			continue
		}
		newStatus := statusMap[accrualResponse.OrderStatus]
		newAccrual := accrualResponse.Accrual
		// put back to queue if no updates were found, set LastChecked to time.Now()
		if newStatus == record.OrderStatus {
			w.log.Info().Msg(fmt.Sprintf("WID %v, order %v — no updates, sending back to queue", w.ID, record.OrderNumber))
			record.LastChecked = time.Now()
			record.RetryAfter = 0
			w.queueIn <- record
		} else {
			// if status update was found, send for DB update
			w.log.Info().Msg(fmt.Sprintf("WID %v, order %v — updated, sending to DB", w.ID, record.OrderNumber))
			finalRecord := modelqueue.OrderQueueEntry{
				UserID:      record.UserID,
				OrderNumber: record.OrderNumber,
				OrderStatus: newStatus,
				Accrual:     newAccrual,
			}
			w.queueOut <- finalRecord
			// if status update is not final, put back to queue, set LastChecked to time.Now()
			if newStatus != "PROCESSED" && newStatus != "INVALID" {
				w.log.Info().Msg(fmt.Sprintf("WID %v, order %v — update is not final, sending back to queue", w.ID, record.OrderNumber))
				record.LastChecked = time.Now()
				record.RetryAfter = 0
				w.queueIn <- record
			}
		}
	}
	return nil
}
