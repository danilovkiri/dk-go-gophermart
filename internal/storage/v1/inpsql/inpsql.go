package inpsql

import (
	"context"
	"database/sql"
	"github.com/danilovkiri/dk-go-gophermart/internal/config"
	"github.com/danilovkiri/dk-go-gophermart/internal/models/modeluser"
	storageErrors "github.com/danilovkiri/dk-go-gophermart/internal/storage/v1/errors"
	"github.com/jackc/pgconn"
	"github.com/jackc/pgerrcode"
	_ "github.com/jackc/pgx/v4/stdlib"
	"log"
	"sync"
	"time"
)

type Storage struct {
	mu       sync.Mutex
	Cfg      *config.StorageConfig
	DB       *sql.DB
	localLog *log.Logger
}

func InitStorage(ctx context.Context, cfg *config.StorageConfig, minorlog *log.Logger) (*Storage, error) {
	db, err := sql.Open("pgx", cfg.DatabaseDSN)
	if err != nil {
		log.Fatal(err)
	}
	// initialize a Storage
	st := Storage{
		Cfg:      cfg,
		DB:       db,
		localLog: minorlog,
	}
	err = st.createTables(ctx)
	if err != nil {
		log.Println(err)
		st.localLog.Fatal(err)
	}
	log.Println("PSQL DB connection was established")
	st.localLog.Println("PSQL DB connection was established")

	return &st, nil
}

func (s *Storage) AddNewUser(ctx context.Context, credentials modeluser.ModelCredentials, userID string) (err error) {
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
		log.Println("Adding new user:", ctx.Err())
		s.localLog.Println("Adding new user:", ctx.Err())
		return &storageErrors.ContextTimeoutExceededError{Err: ctx.Err()}
	case methodErr := <-chanEr:
		log.Println("Adding new user:", methodErr.Error())
		s.localLog.Println("Adding new user:", methodErr.Error())
		return methodErr
	case <-chanOk:
		log.Println("Adding new user: done for", credentials.Login)
		s.localLog.Println("Adding new user: done for", credentials.Login)
		return nil
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
		id           BIGSERIAL   NOT NULL,
		user_id      TEXT        NOT NULL UNIQUE,
		order_number BIGINT      NOT NULL UNIQUE,
		status		 TEXT 		 NOT NULL,
		accrual	     BIGINT      NOT NULL,
		created_at   TIMESTAMPTZ NOT NULL  
	);`
	queries = append(queries, query)
	query = `CREATE TABLE IF NOT EXISTS balance (
		id      BIGSERIAL NOT NULL,
		user_id TEXT      NOT NULL UNIQUE,
		amount  BIGINT    NOT NULL
	);`
	queries = append(queries, query)
	query = `CREATE TABLE IF NOT EXISTS withdrawals (
		id           BIGSERIAL   NOT NULL,
		user_id      TEXT        NOT NULL UNIQUE,
		order_number BIGINT      NOT NULL UNIQUE,
		amount       BIGINT      NOT NULL UNIQUE,
		processed_at TIMESTAMPTZ NOT NULL 
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
