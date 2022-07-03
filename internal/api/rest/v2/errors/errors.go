// Package errors provides custom error types.

package errors

type (
	HandlersFoundNilArgument struct {
		Msg string
	}
)

func (e *HandlersFoundNilArgument) Error() string {
	return e.Msg
}
