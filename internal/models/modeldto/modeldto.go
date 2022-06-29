package modeldto

type (
	User struct {
		Login    string `json:"login,omitempty"`
		Password string `json:"password,omitempty"`
	}
	Balance struct {
		CurrentAmount   float64 `json:"current"`
		WithdrawnAmount float64 `json:"withdrawn"`
	}
	Withdrawal struct {
		OrderNumber     string  `json:"order"`
		WithdrawnAmount float64 `json:"sum"`
		ProcessedAt     string  `json:"processed_at"`
	}
	Order struct {
		OrderNumber string  `json:"number"`
		Status      string  `json:"status"`
		Accrual     float64 `json:"accrual,omitempty"`
		UploadedAt  string  `json:"uploaded_at"`
	}
	NewOrderWithdrawal struct {
		OrderNumber string  `json:"order"`
		Amount      float64 `json:"sum"`
	}
	AccrualResponse struct {
		OrderNumber string  `json:"order"`
		OrderStatus string  `json:"status"`
		Accrual     float64 `json:"accrual,omitempty"`
	}
)
