package errors

type (
	ServiceFoundNilArgument struct {
		Msg string
	}
)

func (e *ServiceFoundNilArgument) Error() string {
	return e.Msg
}
