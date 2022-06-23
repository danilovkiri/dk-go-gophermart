package errors

type (
	ServiceFoundNilArgument struct {
		Msg string
	}
	ServiceIllegalOrderNumber struct {
		Msg string
	}
	ServiceNotEnoughFunds struct {
		Msg string
	}
)

func (e *ServiceFoundNilArgument) Error() string {
	return e.Msg
}

func (e *ServiceIllegalOrderNumber) Error() string {
	return e.Msg
}

func (e *ServiceNotEnoughFunds) Error() string {
	return e.Msg
}
