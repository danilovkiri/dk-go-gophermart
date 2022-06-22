package modeldto

type (
	Balance struct {
		CurrentAmount   float64 `json:"current"`
		WithdrawnAmount float64 `json:"withdrawn"`
	}
	Withdrawal struct {
		OrderNumber     int32   `json:"order"`
		WithdrawnAmount float64 `json:"sum"`
		ProcessedAt     string  `json:"processed_at"`
	}
)
