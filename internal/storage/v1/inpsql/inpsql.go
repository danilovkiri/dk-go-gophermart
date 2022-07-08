// Package inpsql provides functionality for operating a relational DB.

package inpsql

import (
	"context"
	"crypto/sha256"
	"crypto/subtle"
	"database/sql"
	"errors"
	"fmt"
	"strconv"
	"sync"
	"time"

	"github.com/danilovkiri/dk-go-gophermart/internal/config"
	"github.com/danilovkiri/dk-go-gophermart/internal/models/modeldto"
	"github.com/danilovkiri/dk-go-gophermart/internal/models/modelqueue"
	storageErrors "github.com/danilovkiri/dk-go-gophermart/internal/storage/v1/errors"
	"github.com/danilovkiri/dk-go-gophermart/internal/storage/v1/modelstorage"
	"github.com/jackc/pgconn"
	"github.com/jackc/pgerrcode"
	_ "github.com/jackc/pgx/v4/stdlib"
	"github.com/rs/zerolog"
)

// Storage defines attributes of a struct available to its methods.
type Storage struct {
	mu       sync.Mutex
	cfg      *config.StorageConfig
	DB       *sql.DB
	log      *zerolog.Logger
	QueueIn  chan modelqueue.OrderQueueEntry
	QueueOut chan modelqueue.OrderQueueEntry
}

// InitStorage initializes a storage handling service.
func InitStorage(ctx context.Context, cfg *config.StorageConfig, log *zerolog.Logger, wg *sync.WaitGroup) (*Storage, error) {
	db, err := sql.Open("pgx", cfg.DatabaseDSN)
	if err != nil {
		log.Fatal().Err(err).Msg("could not prepare a DB connection")
	}
	// initialize a storage
	queueIn := make(chan modelqueue.OrderQueueEntry)
	queueOut := make(chan modelqueue.OrderQueueEntry)
	st := Storage{
		cfg:      cfg,
		DB:       db,
		log:      log,
		QueueIn:  queueIn,
		QueueOut: queueOut,
	}
	err = st.createTables(ctx)
	if err != nil {
		log.Fatal().Err(err).Msg("could not create DB tables")
	}
	log.Info().Msg("PSQL DB connection was established")

	// send unprocessed orders from DB to queueIn upon initialization
	wg.Add(1)
	go func() {
		defer wg.Done()
		stalledOrders, err := st.getStalledOrders(ctx)
		if err != nil {
			log.Fatal().Err(err).Msg("could not retrieve stalled orders")
		}
		for _, stalledOrder := range stalledOrders {
			st.SendToQueue(modelqueue.OrderQueueEntry{
				UserID:      stalledOrder.UserID,
				OrderNumber: stalledOrder.OrderNumber,
				OrderStatus: stalledOrder.Status,
			})
		}
		log.Info().Msg(fmt.Sprintf("%v stalled orders were sent for processing", len(stalledOrders)))
		<-ctx.Done()
		err = st.DB.Close()
		if err != nil {
			log.Fatal().Err(err).Msg("could not close DB connection")
		}
		log.Info().Msg("PSQL DB connection was closed")
	}()

	// listen for processed orders from queueOut and update them in DB
	wg.Add(1)
	go func() {
		log.Info().Msg("started listening to queue for processed orders")
		defer wg.Done()
		for record := range st.QueueOut {
			err := st.updateOrder(ctx, record.OrderNumber, record.OrderStatus, record.Accrual, record.UserID)
			if err != nil {
				log.Warn().Err(err).Msg(fmt.Sprintf("could not update order %v", record.OrderNumber))
			}
		}
		log.Info().Msg("stopped listening to queue for processed orders")
	}()
	return &st, nil
}

// AddNewUser adds a new user to DB.
func (s *Storage) AddNewUser(ctx context.Context, credentials modeldto.User, userID string) error {
	newUserStmt, err := s.DB.PrepareContext(ctx, "INSERT INTO users (user_id, login, password, registered_at) VALUES ($1, $2, $3, $4)")
	if err != nil {
		return &storageErrors.StatementPSQLError{Err: err}
	}
	defer newUserStmt.Close()
	newBalanceStmt, err := s.DB.PrepareContext(ctx, "INSERT INTO balance (user_id, amount) VALUES ($1, $2)")
	if err != nil {
		return &storageErrors.StatementPSQLError{Err: err}
	}
	defer newBalanceStmt.Close()
	chanOk := make(chan bool)
	chanEr := make(chan error)
	go func() {
		s.mu.Lock()
		defer s.mu.Unlock()
		_, err := newUserStmt.ExecContext(ctx, userID, credentials.Login, credentials.Password, time.Now().Format(time.RFC3339))
		if err != nil {
			if err, ok := err.(*pgconn.PgError); ok && err.Code == pgerrcode.UniqueViolation {
				chanEr <- &storageErrors.AlreadyExistsError{Err: err, ID: credentials.Login}
				return
			}
			chanEr <- &storageErrors.ExecutionPSQLError{Err: err}
			return
		}
		_, err = newBalanceStmt.ExecContext(ctx, userID, 0)
		if err != nil {
			if err, ok := err.(*pgconn.PgError); ok && err.Code == pgerrcode.UniqueViolation {
				chanEr <- &storageErrors.AlreadyExistsError{Err: err, ID: credentials.Login}
				return
			}
			chanEr <- &storageErrors.ExecutionPSQLError{Err: err}
			return
		}
		chanOk <- true
	}()

	select {
	case <-ctx.Done():
		s.log.Error().Err(ctx.Err()).Msg(fmt.Sprintf("adding new user failed for %s", credentials.Login))
		return &storageErrors.ContextTimeoutExceededError{Err: ctx.Err()}
	case methodErr := <-chanEr:
		s.log.Error().Err(methodErr).Msg(fmt.Sprintf("adding new user failed for %s", credentials.Login))
		return methodErr
	case <-chanOk:
		s.log.Info().Msg(fmt.Sprintf("adding new user done for %s", credentials.Login))
		return nil
	}
}

// CheckUser checks whether a user exists in DB.
func (s *Storage) CheckUser(ctx context.Context, credentials modeldto.User) (string, error) {
	selectStmt, err := s.DB.PrepareContext(ctx, "SELECT * FROM users WHERE login = $1")
	if err != nil {
		return "", &storageErrors.StatementPSQLError{Err: err}
	}
	defer selectStmt.Close()
	chanOk := make(chan string)
	chanEr := make(chan error)
	go func() {
		s.mu.Lock()
		defer s.mu.Unlock()
		var queryOutput modelstorage.UserStorageEntry
		err := selectStmt.QueryRowContext(ctx, credentials.Login).Scan(&queryOutput.ID, &queryOutput.UserID, &queryOutput.Login, &queryOutput.Password, &queryOutput.RegisteredAt)
		if err != nil {
			switch {
			case errors.Is(err, sql.ErrNoRows):
				chanEr <- &storageErrors.NotFoundError{Err: err}
				return
			default:
				chanEr <- err
				return
			}
		}
		passwordHash := sha256.Sum256([]byte(credentials.Password))
		expectedPasswordHash := sha256.Sum256([]byte(queryOutput.Password))
		passwordMatch := subtle.ConstantTimeCompare(passwordHash[:], expectedPasswordHash[:]) == 1
		if !passwordMatch {
			chanEr <- &storageErrors.NotFoundError{Err: nil}
		}
		chanOk <- queryOutput.UserID
	}()

	select {
	case <-ctx.Done():
		s.log.Error().Err(ctx.Err()).Msg("user authentication failed")
		return "", &storageErrors.ContextTimeoutExceededError{Err: ctx.Err()}
	case methodErr := <-chanEr:
		s.log.Error().Err(methodErr).Msg("user authentication failed")
		return "", methodErr
	case userID := <-chanOk:
		s.log.Info().Msg("user authentication done")
		return userID, nil
	}
}

// GetCurrentAmount retrieves the current user's balance from DB.
func (s *Storage) GetCurrentAmount(ctx context.Context, userID string) (float64, error) {
	selectStmt, err := s.DB.PrepareContext(ctx, "SELECT * FROM balance WHERE user_id = $1")
	if err != nil {
		return 0, &storageErrors.StatementPSQLError{Err: err}
	}
	defer selectStmt.Close()
	chanOk := make(chan float64)
	chanEr := make(chan error)
	go func() {
		s.mu.Lock()
		defer s.mu.Unlock()
		var queryOutput modelstorage.BalanceStorageEntry
		err := selectStmt.QueryRowContext(ctx, userID).Scan(&queryOutput.ID, &queryOutput.UserID, &queryOutput.Amount)
		if err != nil {
			switch {
			case errors.Is(err, sql.ErrNoRows):
				chanEr <- &storageErrors.NotFoundError{Err: err}
				return
			default:
				chanEr <- err
				return
			}
		}
		chanOk <- queryOutput.Amount
	}()
	select {
	case <-ctx.Done():
		s.log.Error().Err(ctx.Err()).Msg("getting current balance failed")
		return 0, &storageErrors.ContextTimeoutExceededError{Err: ctx.Err()}
	case methodErr := <-chanEr:
		s.log.Error().Err(methodErr).Msg("getting current balance failed")
		return 0, methodErr
	case amount := <-chanOk:
		s.log.Info().Msg("getting current balance done")
		return amount, nil
	}
}

// GetWithdrawnAmount retrieves the current user's withdrawn balance from DB.
func (s *Storage) GetWithdrawnAmount(ctx context.Context, userID string) (float64, error) {
	selectStmt, err := s.DB.PrepareContext(ctx, "SELECT * FROM withdrawals WHERE user_id = $1")
	if err != nil {
		return 0, &storageErrors.StatementPSQLError{Err: err}
	}
	defer selectStmt.Close()
	chanOk := make(chan float64)
	chanEr := make(chan error)
	go func() {
		s.mu.Lock()
		defer s.mu.Unlock()
		rows, err := selectStmt.QueryContext(ctx, userID)
		if err != nil {
			chanEr <- &storageErrors.ExecutionPSQLError{Err: err}
			return
		}
		defer rows.Close()
		var queryOutput []modelstorage.WithdrawalStorageEntry
		for rows.Next() {
			var queryOutputRow modelstorage.WithdrawalStorageEntry
			err = rows.Scan(&queryOutputRow.ID, &queryOutputRow.UserID, &queryOutputRow.OrderNumber, &queryOutputRow.Amount, &queryOutputRow.ProcessedAt)
			if err != nil {
				chanEr <- &storageErrors.ScanningPSQLError{Err: err}
				return
			}
			queryOutput = append(queryOutput, queryOutputRow)
		}
		err = rows.Err()
		if err != nil {
			chanEr <- &storageErrors.ScanningPSQLError{Err: err}
		}
		var withdrawnAmount float64
		for _, entry := range queryOutput {
			withdrawnAmount += entry.Amount
		}
		chanOk <- withdrawnAmount
	}()
	select {
	case <-ctx.Done():
		s.log.Error().Err(ctx.Err()).Msg("getting withdrawn balance failed")
		return 0, &storageErrors.ContextTimeoutExceededError{Err: ctx.Err()}
	case methodErr := <-chanEr:
		s.log.Error().Err(methodErr).Msg("getting withdrawn balance failed")
		return 0, methodErr
	case amount := <-chanOk:
		s.log.Info().Msg("getting withdrawn balance done")
		return amount, nil
	}
}

// GetWithdrawals retrieves a user's history of withdrawals from DB.
func (s *Storage) GetWithdrawals(ctx context.Context, userID string) ([]modelstorage.WithdrawalStorageEntry, error) {
	selectStmt, err := s.DB.PrepareContext(ctx, "SELECT * FROM withdrawals WHERE user_id = $1")
	if err != nil {
		return nil, &storageErrors.StatementPSQLError{Err: err}
	}
	defer selectStmt.Close()
	chanOk := make(chan []modelstorage.WithdrawalStorageEntry)
	chanEr := make(chan error)
	go func() {
		s.mu.Lock()
		defer s.mu.Unlock()
		rows, err := selectStmt.QueryContext(ctx, userID)
		if err != nil {
			chanEr <- &storageErrors.ExecutionPSQLError{Err: err}
			return
		}
		defer rows.Close()
		var queryOutput []modelstorage.WithdrawalStorageEntry
		for rows.Next() {
			var queryOutputRow modelstorage.WithdrawalStorageEntry
			err = rows.Scan(&queryOutputRow.ID, &queryOutputRow.UserID, &queryOutputRow.OrderNumber, &queryOutputRow.Amount, &queryOutputRow.ProcessedAt)
			if err != nil {
				chanEr <- &storageErrors.ScanningPSQLError{Err: err}
				return
			}
			queryOutput = append(queryOutput, queryOutputRow)
		}
		err = rows.Err()
		if err != nil {
			chanEr <- &storageErrors.ScanningPSQLError{Err: err}
		}
		chanOk <- queryOutput
	}()
	select {
	case <-ctx.Done():
		s.log.Error().Err(ctx.Err()).Msg("getting withdrawals failed")
		return nil, &storageErrors.ContextTimeoutExceededError{Err: ctx.Err()}
	case methodErr := <-chanEr:
		s.log.Error().Err(methodErr).Msg("getting withdrawals failed")
		return nil, methodErr
	case query := <-chanOk:
		s.log.Info().Msg("getting withdrawals done")
		return query, nil
	}
}

// GetOrders retrieves a user's history of orders from DB.
func (s *Storage) GetOrders(ctx context.Context, userID string) ([]modelstorage.OrderStorageEntry, error) {
	selectStmt, err := s.DB.PrepareContext(ctx, "SELECT * FROM orders WHERE user_id = $1")
	if err != nil {
		return nil, &storageErrors.StatementPSQLError{Err: err}
	}
	defer selectStmt.Close()
	chanOk := make(chan []modelstorage.OrderStorageEntry)
	chanEr := make(chan error)
	go func() {
		s.mu.Lock()
		defer s.mu.Unlock()
		rows, err := selectStmt.QueryContext(ctx, userID)
		if err != nil {
			chanEr <- &storageErrors.ExecutionPSQLError{Err: err}
			return
		}
		defer rows.Close()
		var queryOutput []modelstorage.OrderStorageEntry
		for rows.Next() {
			var queryOutputRow modelstorage.OrderStorageEntry
			err = rows.Scan(&queryOutputRow.ID, &queryOutputRow.UserID, &queryOutputRow.OrderNumber, &queryOutputRow.Status, &queryOutputRow.Accrual, &queryOutputRow.CreatedAt)
			if err != nil {
				chanEr <- &storageErrors.ScanningPSQLError{Err: err}
				return
			}
			queryOutput = append(queryOutput, queryOutputRow)
		}
		err = rows.Err()
		if err != nil {
			chanEr <- &storageErrors.ScanningPSQLError{Err: err}
		}
		chanOk <- queryOutput
	}()
	select {
	case <-ctx.Done():
		s.log.Error().Err(ctx.Err()).Msg("getting orders failed")
		return nil, &storageErrors.ContextTimeoutExceededError{Err: ctx.Err()}
	case methodErr := <-chanEr:
		s.log.Error().Err(methodErr).Msg("getting orders failed")
		return nil, methodErr
	case query := <-chanOk:
		s.log.Info().Msg("getting orders done")
		return query, nil
	}
}

// AddNewWithdrawal adds a new withdrawal event to DB.
func (s *Storage) AddNewWithdrawal(ctx context.Context, userID string, withdrawal modeldto.NewOrderWithdrawal) error {
	newOrderStmt, err := s.DB.PrepareContext(ctx, "INSERT INTO orders (user_id, order_number, status, accrual, created_at) VALUES ($1, $2, $3, $4, $5)")
	if err != nil {
		return &storageErrors.StatementPSQLError{Err: err}
	}
	defer newOrderStmt.Close()
	newWithdrawalStmt, err := s.DB.PrepareContext(ctx, "INSERT INTO withdrawals (user_id, order_number, amount, processed_at) VALUES ($1, $2, $3, $4)")
	if err != nil {
		return &storageErrors.StatementPSQLError{Err: err}
	}
	defer newWithdrawalStmt.Close()
	updBalanceStmt, err := s.DB.PrepareContext(ctx, "UPDATE balance SET amount = (amount - $1) WHERE user_id = $2")
	if err != nil {
		return &storageErrors.StatementPSQLError{Err: err}
	}
	defer updBalanceStmt.Close()
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return &storageErrors.ExecutionPSQLError{Err: err}
	}
	defer tx.Rollback()
	txNewOrderStmt := tx.StmtContext(ctx, newOrderStmt)
	txNewWithdrawalStmt := tx.StmtContext(ctx, newWithdrawalStmt)
	txUpdBalanceStmt := tx.StmtContext(ctx, updBalanceStmt)
	chanOk := make(chan bool)
	chanEr := make(chan error)
	go func() {
		_, err = txNewOrderStmt.ExecContext(ctx, userID, withdrawal.OrderNumber, "PROCESSED", 0.0, time.Now().Format(time.RFC3339))
		if err != nil {
			if err, ok := err.(*pgconn.PgError); ok && err.Code == pgerrcode.UniqueViolation {
				chanEr <- &storageErrors.AlreadyExistsError{Err: err, ID: withdrawal.OrderNumber}
			}
			chanEr <- &storageErrors.ExecutionPSQLError{Err: err}
		}
		_, err = txNewWithdrawalStmt.ExecContext(ctx, userID, withdrawal.OrderNumber, withdrawal.Amount, time.Now().Format(time.RFC3339))
		if err != nil {
			if err, ok := err.(*pgconn.PgError); ok && err.Code == pgerrcode.UniqueViolation {
				chanEr <- &storageErrors.AlreadyExistsError{Err: err, ID: withdrawal.OrderNumber}
			}
			chanEr <- &storageErrors.ExecutionPSQLError{Err: err}
		}
		_, err = txUpdBalanceStmt.ExecContext(ctx, withdrawal.Amount, userID)
		if err != nil {
			chanEr <- &storageErrors.ExecutionPSQLError{Err: err}
		}
		chanOk <- true
	}()
	select {
	case <-ctx.Done():
		s.log.Error().Err(ctx.Err()).Msg("processing new withdrawal order failed")
		return &storageErrors.ContextTimeoutExceededError{Err: ctx.Err()}
	case methodErr := <-chanEr:
		s.log.Error().Err(methodErr).Msg("processing new withdrawal order failed")
		return methodErr
	case <-chanOk:
		s.log.Info().Msg("processing new withdrawal order done")
		return tx.Commit()
	}
}

// SendToQueue sends an order to processing queue.
func (s *Storage) SendToQueue(item modelqueue.OrderQueueEntry) {
	s.QueueIn <- item
}

// AddNewOrder adds a new order event to DB.
func (s *Storage) AddNewOrder(ctx context.Context, userID string, orderNumber int) error {
	selectStmt, err := s.DB.PrepareContext(ctx, "SELECT * FROM orders WHERE order_number = $1")
	if err != nil {
		return &storageErrors.StatementPSQLError{Err: err}
	}
	newOrderStmt, err := s.DB.PrepareContext(ctx, "INSERT INTO orders (user_id, order_number, status, accrual, created_at) VALUES ($1, $2, $3, $4, $5)")
	if err != nil {
		return &storageErrors.StatementPSQLError{Err: err}
	}
	defer selectStmt.Close()
	defer newOrderStmt.Close()
	chanOk := make(chan bool)
	chanEr := make(chan error)
	go func() {
		_, err = newOrderStmt.ExecContext(ctx, userID, orderNumber, "NEW", 0.0, time.Now().Format(time.RFC3339))
		if err != nil {
			if err, ok := err.(*pgconn.PgError); ok && err.Code == pgerrcode.UniqueViolation {
				// distinguish http.StatusOK from http.Conflict
				var queryOutput modelstorage.OrderStorageEntry
				err := selectStmt.QueryRowContext(ctx, orderNumber).Scan(&queryOutput.ID, &queryOutput.UserID, &queryOutput.OrderNumber, &queryOutput.Status, &queryOutput.Accrual, &queryOutput.CreatedAt)
				if err != nil {
					chanEr <- &storageErrors.ExecutionPSQLError{Err: err}
				} else {
					if queryOutput.UserID == userID {
						chanEr <- &storageErrors.AlreadyExistsError{Err: err, ID: strconv.Itoa(orderNumber)}
					}
					chanEr <- &storageErrors.AlreadyExistsAndViolatesError{Err: err, ID: strconv.Itoa(orderNumber)}
				}
			}
			chanEr <- &storageErrors.ExecutionPSQLError{Err: err}
		}
		chanOk <- true
	}()

	select {
	case <-ctx.Done():
		s.log.Error().Err(ctx.Err()).Msg(fmt.Sprintf("adding new order failed for order %v", orderNumber))
		return &storageErrors.ContextTimeoutExceededError{Err: ctx.Err()}
	case methodErr := <-chanEr:
		s.log.Error().Err(methodErr).Msg(fmt.Sprintf("adding new order failed for order %v", orderNumber))
		return methodErr
	case <-chanOk:
		s.log.Info().Msg(fmt.Sprintf("adding new order done for order %v", orderNumber))
		return nil
	}
}

// getStalledOrders retrieves all unprocessed orders from DB upon server startup and sends them to queue for processing.
func (s *Storage) getStalledOrders(ctx context.Context) ([]modelstorage.OrderStorageEntry, error) {
	selectStmt, err := s.DB.PrepareContext(ctx, "SELECT * FROM orders WHERE status NOT IN ('PROCESSED', 'INVALID')")
	if err != nil {
		return nil, &storageErrors.StatementPSQLError{Err: err}
	}
	defer selectStmt.Close()
	chanOk := make(chan []modelstorage.OrderStorageEntry)
	chanEr := make(chan error)
	go func() {
		s.mu.Lock()
		defer s.mu.Unlock()
		rows, err := selectStmt.QueryContext(ctx)
		if err != nil {
			chanEr <- &storageErrors.ExecutionPSQLError{Err: err}
			return
		}
		defer rows.Close()
		var queryOutput []modelstorage.OrderStorageEntry
		for rows.Next() {
			var queryOutputRow modelstorage.OrderStorageEntry
			err = rows.Scan(&queryOutputRow.ID, &queryOutputRow.UserID, &queryOutputRow.OrderNumber, &queryOutputRow.Status, &queryOutputRow.Accrual, &queryOutputRow.CreatedAt)
			if err != nil {
				chanEr <- &storageErrors.ScanningPSQLError{Err: err}
				return
			}
			queryOutput = append(queryOutput, queryOutputRow)
		}
		err = rows.Err()
		if err != nil {
			chanEr <- &storageErrors.ScanningPSQLError{Err: err}
		}
		chanOk <- queryOutput
	}()
	select {
	case <-ctx.Done():
		s.log.Error().Err(ctx.Err()).Msg("getting stalled orders failed")
		return nil, &storageErrors.ContextTimeoutExceededError{Err: ctx.Err()}
	case methodErr := <-chanEr:
		s.log.Error().Err(methodErr).Msg("getting stalled orders failed")
		return nil, methodErr
	case query := <-chanOk:
		s.log.Info().Msg("getting stalled orders done")
		return query, nil
	}
}

// updateOrder updates order entry in DB.
func (s *Storage) updateOrder(ctx context.Context, orderNumber int, status string, accrual float64, userID string) error {
	updOrderStmt, err := s.DB.PrepareContext(ctx, "UPDATE orders SET status = $1, accrual = $2 WHERE order_number = $3")
	if err != nil {
		return &storageErrors.StatementPSQLError{Err: err}
	}
	defer updOrderStmt.Close()
	updBalanceStmt, err := s.DB.PrepareContext(ctx, "UPDATE balance SET amount = (amount + $1) WHERE user_id = $2")
	if err != nil {
		return &storageErrors.StatementPSQLError{Err: err}
	}
	defer updBalanceStmt.Close()
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return &storageErrors.ExecutionPSQLError{Err: err}
	}
	defer tx.Rollback()
	txUpdOrderStmt := tx.StmtContext(ctx, updOrderStmt)
	txUpdBalanceStmt := tx.StmtContext(ctx, updBalanceStmt)
	chanOk := make(chan bool)
	chanEr := make(chan error)
	go func() {
		s.mu.Lock()
		defer s.mu.Unlock()
		_, err = txUpdOrderStmt.ExecContext(ctx, status, accrual, orderNumber)
		if err != nil {
			chanEr <- &storageErrors.ExecutionPSQLError{Err: err}
		}
		_, err = txUpdBalanceStmt.ExecContext(ctx, accrual, userID)
		if err != nil {
			chanEr <- &storageErrors.ExecutionPSQLError{Err: err}
		}
		chanOk <- true
	}()

	select {
	case <-ctx.Done():
		s.log.Error().Err(ctx.Err()).Msg(fmt.Sprintf("updating order failed for order %v", orderNumber))
		return &storageErrors.ContextTimeoutExceededError{Err: ctx.Err()}
	case methodErr := <-chanEr:
		s.log.Error().Err(methodErr).Msg(fmt.Sprintf("updating order failed for order %v", orderNumber))
		return methodErr
	case <-chanOk:
		s.log.Info().Msg(fmt.Sprintf("updating order done for order %v", orderNumber))
		return tx.Commit()
	}
}

// createTables creates DB tables if not exist.
func (s *Storage) createTables(ctx context.Context) error {
	var queries []string
	query := `CREATE TABLE IF NOT EXISTS users (
		id            BIGSERIAL   NOT NULL UNIQUE,
		user_id       TEXT        NOT NULL UNIQUE,
		login         TEXT        NOT NULL UNIQUE,
		password      TEXT        NOT NULL,
		registered_at TIMESTAMPTZ NOT NULL  
	);`
	queries = append(queries, query)
	query = `CREATE TABLE IF NOT EXISTS orders (
		id           BIGSERIAL      NOT NULL UNIQUE,
		user_id      TEXT           NOT NULL,
		order_number BIGINT         NOT NULL UNIQUE,
		status		 TEXT 		    NOT NULL,
		accrual	     NUMERIC(10, 2) NOT NULL,
		created_at   TIMESTAMPTZ    NOT NULL  
	);`
	queries = append(queries, query)
	query = `CREATE TABLE IF NOT EXISTS balance (
		id      BIGSERIAL      NOT NULL UNIQUE,
		user_id TEXT           NOT NULL UNIQUE,
		amount  NUMERIC(10, 2) NOT NULL
	);`
	queries = append(queries, query)
	query = `CREATE TABLE IF NOT EXISTS withdrawals (
		id           BIGSERIAL      NOT NULL UNIQUE,
		user_id      TEXT           NOT NULL,
		order_number BIGINT         NOT NULL UNIQUE,
		amount       NUMERIC(10, 2) NOT NULL,
		processed_at TIMESTAMPTZ    NOT NULL 
	);`
	queries = append(queries, query)
	for _, subquery := range queries {
		_, err := s.DB.ExecContext(ctx, subquery)
		if err != nil {
			return err
		}
	}
	return nil
}
