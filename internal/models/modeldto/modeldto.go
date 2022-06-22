package modeldto

type (
	Balance struct {
		CurrentAmount   float64 `json:"current"`
		WithdrawnAmount float64 `json:"withdrawn"`
	}
)
