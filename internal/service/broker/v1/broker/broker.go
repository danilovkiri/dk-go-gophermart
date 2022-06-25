package broker

import (
	"context"
	"fmt"
	"github.com/danilovkiri/dk-go-gophermart/internal/models/modelqueue"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"golang.org/x/sync/errgroup"
	"sync"
	"time"
)

type Broker struct {
	ctx      context.Context
	log      *zerolog.Logger
	queueIn  chan modelqueue.OrderQueueEntry
	queueOut chan modelqueue.OrderQueueEntry
	wg       *sync.WaitGroup
}

type GetAccrualWorker struct {
	ID       int
	ctx      context.Context
	log      *zerolog.Logger
	queueIn  chan modelqueue.OrderQueueEntry
	queueOut chan modelqueue.OrderQueueEntry
}

func InitBroker(ctx context.Context, queueIn chan modelqueue.OrderQueueEntry, queueOut chan modelqueue.OrderQueueEntry, log *zerolog.Logger, wg *sync.WaitGroup) *Broker {
	broker := Broker{
		ctx:      ctx,
		log:      log,
		queueIn:  queueIn,
		queueOut: queueOut,
		wg:       wg,
	}
	return &broker
}

func (b *Broker) ListenAndProcess() {
	b.wg.Add(1)
	go func() {
		log.Info().Msg("started listening to queue for unprocessed orders")
		defer b.wg.Done()
		g, _ := errgroup.WithContext(b.ctx)
		for i := 0; i < 8; i++ {
			w := &GetAccrualWorker{ID: i, ctx: b.ctx, queueIn: b.queueIn, queueOut: b.queueOut, log: b.log}
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

func (w *GetAccrualWorker) processAsync() error {
	for record := range w.queueIn {
		// wait for at least 10 seconds before querying the same order again
		// implement breaking out functionality upon ctx.Done()
		for time.Now().Sub(record.LastChecked) < 10*time.Second {
			select {
			case <-w.ctx.Done():
				return nil
			default:

			}
		}

		// retrieve status and accrual updates via client — TBD
		statusMap := map[string]string{
			"INVALID":    "INVALID",
			"PROCESSED":  "PROCESSED",
			"PROCESSING": "PROCESSING",
			"REGISTERED": "NEW",
		}
		var newStatus string
		var newAccrual float64
		var err error
		//
		newStatus = "REGISTERED"
		newAccrual = 0
		//
		newStatus = statusMap[newStatus]

		if err != nil {
			if record.RetryCount >= 3 {
				// abandon processing if 3 retries were unsuccessfully performed
				w.log.Warn().Msg(fmt.Sprintf("WID %v, order %v — abandonment due to retry limit exceeding", w.ID, record.OrderNumber))
				finalRecord := modelqueue.OrderQueueEntry{
					UserID:      record.UserID,
					OrderNumber: record.OrderNumber,
					OrderStatus: record.OrderStatus,
					Accrual:     record.Accrual,
				}
				w.queueOut <- finalRecord
			} else {
				// put back to queue if querying resulted in error, increment RetryCount, set LastChecked to time.Now()
				w.log.Warn().Msg(fmt.Sprintf("WID %v, order %v — could not process, sending back to queue", w.ID, record.OrderNumber))
				record.RetryCount += 1
				record.LastChecked = time.Now()
				w.queueIn <- record
			}
		}

		// put back to queue if no updates were found, set LastChecked to time.Now()
		if newStatus == record.OrderStatus {
			w.log.Info().Msg(fmt.Sprintf("WID %v, order %v — no updates, sending back to queue", w.ID, record.OrderNumber))
			record.LastChecked = time.Now()
			w.queueIn <- record
		} else {
			// if status update was found, send for DB update
			w.log.Info().Msg(fmt.Sprintf("WID %v, order %v — updated, sending to DB", w.ID, record.OrderNumber))
			finalRecord := modelqueue.OrderQueueEntry{
				UserID:      record.UserID,
				OrderNumber: record.OrderNumber,
				OrderStatus: record.OrderStatus,
				Accrual:     newAccrual,
			}
			w.queueOut <- finalRecord
			// if status update is not final, put back to queue, set LastChecked to time.Now()
			if newStatus != "PROCESSED" && newStatus != "INVALID" {
				w.log.Info().Msg(fmt.Sprintf("WID %v, order %v — update is not final, sending back to queue", w.ID, record.OrderNumber))
				record.LastChecked = time.Now()
				w.queueIn <- record
			}
		}
	}
	return nil
}
