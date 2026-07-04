package payment

import (
	"time"
)

type PaymentProvider string

const (
	ProviderStripe    PaymentProvider = "stripe"
	ProviderPayPal    PaymentProvider = "paypal"
	ProviderBank      PaymentProvider = "bank"
	ProviderApplePay  PaymentProvider = "applepay"
	ProviderGooglePay PaymentProvider = "googlepay"
)

type TransactionStatus string

const (
	StatusPending   TransactionStatus = "pending"
	StatusSuccess   TransactionStatus = "success"
	StatusFailed    TransactionStatus = "failed"
	StatusCancelled TransactionStatus = "cancelled"
)

type PaymentMethod string

const (
	MethodCreditCard  PaymentMethod = "credit_card"
	MethodDebitCard   PaymentMethod = "debit_card"
	MethodBankTransfer PaymentMethod = "bank_transfer"
	MethodWallet      PaymentMethod = "wallet"
)

type Transaction struct {
	ID             string            `json:"id"`
	Provider       PaymentProvider   `json:"provider"`
	Method         PaymentMethod     `json:"method"`
	Amount         float64           `json:"amount"`
	Currency       string            `json:"currency"`
	Status         TransactionStatus `json:"status"`
	Description    string            `json:"description"`
	CreatedAt      time.Time         `json:"created_at"`
	UpdatedAt      time.Time         `json:"updated_at"`
	UserID         string            `json:"user_id"`
	ReferenceID    string            `json:"reference_id"`
	Error          string            `json:"error,omitempty"`
	Metadata       map[string]interface{} `json:"metadata,omitempty"`
}

type PaymentRequest struct {
	Amount      float64         `json:"amount"`
	Currency    string          `json:"currency"`
	Method      PaymentMethod   `json:"method"`
	Description string          `json:"description"`
	UserID      string          `json:"user_id"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

type PaymentResponse struct {
	Transaction *Transaction `json:"transaction"`
	Error       error        `json:"error,omitempty"`
}

type Logger interface {
	Log(transaction *Transaction)
}

type PaymentProcessor interface {
	ProcessPayment(request PaymentRequest) PaymentResponse
	GetTransaction(id string) (*Transaction, error)
	ListTransactions(userID string, limit int, offset int) ([]*Transaction, error)
	UpdateTransactionStatus(id string, status TransactionStatus, error string) error
}

func NewPaymentProcessor(logger Logger) PaymentProcessor {
	return &paymentProcessor{}
}

type paymentProcessor struct {
	logger Logger
}

func (p *paymentProcessor) ProcessPayment(request PaymentRequest) PaymentResponse {
	return PaymentResponse{}
}

func (p *paymentProcessor) GetTransaction(id string) (*Transaction, error) {
	return &Transaction{}, nil
}

func (p *paymentProcessor) ListTransactions(userID string, limit int, offset int) ([]*Transaction, error) {
	return []*Transaction{}, nil
}

func (p *paymentProcessor) UpdateTransactionStatus(id string, status TransactionStatus, error string) error {
	return nil
}
