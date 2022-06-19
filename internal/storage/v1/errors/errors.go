package errors

import (
	"fmt"
)

type (
	StatementPSQLError struct {
		Err error
	}
	AlreadyExistsError struct {
		Err error
		ID  string
	}
	ExecutionPSQLError struct {
		Err error
	}
	ContextTimeoutExceededError struct {
		Err error
	}
)

func (e *StatementPSQLError) Error() string {
	return fmt.Sprintf("%s: could not compile", e.Err.Error())
}

func (e *AlreadyExistsError) Error() string {
	return fmt.Sprintf("%s: already exists", e.ID)
}

func (e *ExecutionPSQLError) Error() string {
	return fmt.Sprintf("%s: could not execute", e.Err.Error())
}

func (e *ContextTimeoutExceededError) Error() string {
	return fmt.Sprintf("%s: context timeout exceeded", e.Err.Error())
}
