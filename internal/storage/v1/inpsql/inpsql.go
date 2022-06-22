package inpsql

import (
	"context"
	"crypto/sha256"
	"crypto/subtle"
	"database/sql"
	"errors"
	"fmt"
	"github.com/danilovkiri/dk-go-gophermart/internal/config"
	"github.com/danilovkiri/dk-go-gophermart/internal/models/modelstorage"
	"github.com/danilovkiri/dk-go-gophermart/internal/models/modeluser"
	storageErrors "github.com/danilovkiri/dk-go-gophermart/internal/storage/v1/errors"
	"github.com/jackc/pgconn"
	"github.com/jackc/pgerrcode"
	_ "github.com/jackc/pgx/v4/stdlib"
	"github.com/rs/zerolog"
	"sync"
	"time"
)

type Storage struct {
	mu  sync.Mutex
	Cfg *config.StorageConfig
	DB  *sql.DB
	log *zerolog.Logger
}

func InitStorage(ctx context.Context, cfg *config.StorageConfig, log *zerolog.Logger) (*Storage, error) {
	db, err := sql.Open("pgx", cfg.DatabaseDSN)
	if err != nil {
		log.Fatal().Err(err).Msg("")
	}
	// initialize a Storage
	st := Storage{
		Cfg: cfg,
		DB:  db,
		log: log,
	}
	err = st.createTables(ctx)
	if err != nil {
		log.Fatal().Err(err).Msg("")
	}
	log.Info().Msg("PSQL DB connection was established")
	return &st, nil
}

func (s *Storage) AddNewUser(ctx context.Context, credentials modeluser.ModelCredentials, userID string) error {
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

func (s *Storage) CheckUser(ctx context.Context, credentials modeluser.ModelCredentials) (string, error) {
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
		s.log.Error().Err(ctx.Err()).Msg(fmt.Sprint("user authentication failed"))
		return "", &storageErrors.ContextTimeoutExceededError{Err: ctx.Err()}
	case methodErr := <-chanEr:
		s.log.Error().Err(methodErr).Msg(fmt.Sprint("user authentication failed"))
		return "", methodErr
	case userID := <-chanOk:
		s.log.Info().Msg(fmt.Sprint("user authentication done"))
		return userID, nil
	}
}

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
		s.log.Error().Err(ctx.Err()).Msg(fmt.Sprint("getting current balance failed"))
		return 0, &storageErrors.ContextTimeoutExceededError{Err: ctx.Err()}
	case methodErr := <-chanEr:
		s.log.Error().Err(methodErr).Msg(fmt.Sprint("getting current balance failed"))
		return 0, methodErr
	case amount := <-chanOk:
		s.log.Info().Msg(fmt.Sprint("getting current balance done"))
		return amount, nil
	}
}

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
		s.log.Error().Err(ctx.Err()).Msg(fmt.Sprint("getting withdrawn balance failed"))
		return 0, &storageErrors.ContextTimeoutExceededError{Err: ctx.Err()}
	case methodErr := <-chanEr:
		s.log.Error().Err(methodErr).Msg(fmt.Sprint("getting withdrawn balance failed"))
		return 0, methodErr
	case amount := <-chanOk:
		s.log.Info().Msg(fmt.Sprint("getting withdrawn balance done"))
		return amount, nil
	}
}

func (s *Storage) createTables(ctx context.Context) error {
	var queries []string
	query := `CREATE TABLE IF NOT EXISTS users (
		id            BIGSERIAL   NOT NULL,
		user_id       TEXT        NOT NULL UNIQUE,
		login         TEXT        NOT NULL UNIQUE,
		password      TEXT        NOT NULL,
		registered_at TIMESTAMPTZ NOT NULL  
	);`
	queries = append(queries, query)
	query = `CREATE TABLE IF NOT EXISTS orders (
		id           BIGSERIAL      NOT NULL,
		user_id      TEXT           NOT NULL UNIQUE,
		order_number BIGINT         NOT NULL UNIQUE,
		status		 TEXT 		    NOT NULL,
		accrual	     NUMERIC(10, 2) NOT NULL,
		created_at   TIMESTAMPTZ    NOT NULL  
	);`
	queries = append(queries, query)
	query = `CREATE TABLE IF NOT EXISTS balance (
		id      BIGSERIAL      NOT NULL,
		user_id TEXT           NOT NULL UNIQUE,
		amount  NUMERIC(10, 2) NOT NULL
	);`
	queries = append(queries, query)
	query = `CREATE TABLE IF NOT EXISTS withdrawals (
		id           BIGSERIAL      NOT NULL,
		user_id      TEXT           NOT NULL UNIQUE,
		order_number BIGINT         NOT NULL UNIQUE,
		amount       NUMERIC(10, 2) NOT NULL UNIQUE,
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
