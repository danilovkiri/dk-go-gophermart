package modeldto

type (
	User struct {
		Login    string `json:"login"`
		Password string `json:"password"`
	}
	Balance struct {
		CurrentAmount   float64 `json:"current"`
		WithdrawnAmount float64 `json:"withdrawn"`
	}
	Withdrawal struct {
		OrderNumber     int     `json:"order"`
		WithdrawnAmount float64 `json:"sum"`
		ProcessedAt     string  `json:"processed_at"`
	}
	Order struct {
		OrderNumber int     `json:"number"`
		Status      string  `json:"status"`
		Accrual     float64 `json:"accrual,omitempty"`
		UploadedAt  string  `json:"uploaded_at"`
	}
	NewOrderWithdrawal struct {
		OrderNumber int     `json:"order"`
		Amount      float64 `json:"sum"`
	}
)
