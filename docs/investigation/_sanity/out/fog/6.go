package payment

import (
	"time"
)

type PaymentProvider string

const (
	ProviderStripe    PaymentProvider = "stripe"
	ProviderPayPal    PaymentProvider = "paypal"
	ProviderBank      PaymentProvider = "bank"
	ProviderApple     PaymentProvider = "apple"
	ProviderGoogle    PaymentProvider = "google"
)

type TransactionStatus string

const (
	StatusPending   TransactionStatus = "pending"
	StatusSuccess   TransactionStatus = "success"
	StatusFailed    TransactionStatus = "failed"
	StatusCancelled TransactionStatus = "cancelled"
)

type Transaction struct {
	ID            string            `json:"id"`
	Provider      PaymentProvider   `json:"provider"`
	Amount        float64           `json:"amount"`
	Currency      string            `json:"currency"`
	Status        TransactionStatus `json:"status"`
	Description   string            `json:"description"`
	CreatedAt     time.Time         `json:"created_at"`
	UpdatedAt     time.Time         `json:"updated_at"`
	Error         *string           `json:"error,omitempty"`
	Metadata      map[string]string `json:"metadata,omitempty"`
}

type PaymentRequest struct {
	Amount      float64            `json:"amount"`
	Currency    string             `json:"currency"`
	Description string             `json:"description"`
	Provider    PaymentProvider    `json:"provider"`
	Metadata    map[string]string  `json:"metadata,omitempty"`
	SuccessURL  *string            `json:"success_url,omitempty"`
	FailureURL  *string            `json:"failure_url,omitempty"`
}

type PaymentResponse struct {
	TransactionID string            `json:"transaction_id"`
	Status        TransactionStatus `json:"status"`
	URL           *string           `json:"url,omitempty"`
	Error         *string           `json:"error,omitempty"`
}

type Logger interface {
	Log(transaction *Transaction)
}

type PaymentProcessor interface {
	ProcessPayment(request PaymentRequest) (*PaymentResponse, error)
	GetTransaction(id string) (*Transaction, error)
	ListTransactions(provider PaymentProvider, status TransactionStatus, limit int) ([]*Transaction, error)
}
