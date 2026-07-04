package payment

import (
	"time"
)

type TransactionStatus string
type PaymentProvider string

const (
	StatusSuccess TransactionStatus = "success"
	StatusFailed  TransactionStatus = "failed"
	StatusPending TransactionStatus = "pending"
)

const (
	ProviderStripe    PaymentProvider = "stripe"
	ProviderPayPal    PaymentProvider = "paypal"
	ProviderBank      PaymentProvider = "bank"
	ProviderApplePay  PaymentProvider = "apple_pay"
	ProviderGooglePay PaymentProvider = "google_pay"
)

type Transaction struct {
	ID            string
	Amount        float64
	Currency      string
	Status        TransactionStatus
	Provider      PaymentProvider
	Description   string
	CreatedAt     time.Time
	UpdatedAt     time.Time
	Error         *string
	UserID        string
	ReferenceID   string
	Metadata      map[string]interface{}
}

type TransactionResult struct {
	Transaction *Transaction
	Error       error
}

type PaymentProcessor interface {
	ProcessTransaction(transaction *Transaction) *TransactionResult
	GetSupportedProviders() []PaymentProvider
}

type Logger interface {
	LogTransaction(transaction *Transaction)
	LogError(operation string, err error)
}

type PaymentService struct {
	processor PaymentProcessor
	logger  Logger
}

func (s *PaymentService) HandleTransaction(transaction *Transaction) *TransactionResult {
	return nil
}

func (s *PaymentService) GetTransaction(id string) (*Transaction, error) {
	return nil, nil
}

func (s *PaymentService) ListTransactions(userID string, limit int) ([]*Transaction, error) {
	return nil, nil
}
