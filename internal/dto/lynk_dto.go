package dto

type LynkWebhookPayload struct {
	Email         string `json:"email"`
	ProductName   string `json:"product_name"`
	Amount        int    `json:"amount"`
	Status        string `json:"status"`
	TransactionID string `json:"transaction_id"`
}
