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
	Order struct {
		OrderNumber int32   `json:"number"`
		Status      string  `json:"status"`
		Accrual     float64 `json:"accrual,omitempty"`
		UploadedAt  string  `json:"uploaded_at"`
	}
)
