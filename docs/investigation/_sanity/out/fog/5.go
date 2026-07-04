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

type PaymentStatus string

const (
	StatusPending   PaymentStatus = "pending"
	StatusSuccess   PaymentStatus = "success"
	StatusFailed    PaymentStatus = "failed"
	StatusCancelled PaymentStatus = "cancelled"
)

type Payment struct {
	ID            string        `json:"id"`
	Amount        float64       `json:"amount"`
	Currency      string        `json:"currency"`
	Provider      PaymentProvider `json:"provider"`
	Status        PaymentStatus `json:"status"`
	Description   string        `json:"description"`
	CreatedAt     time.Time     `json:"created_at"`
	UpdatedAt     time.Time     `json:"updated_at"`
	TransactionID string        `json:"transaction_id,omitempty"`
	Error         string        `json:"error,omitempty"`
}

type PaymentRequest struct {
	Amount      float64         `json:"amount"`
	Currency    string          `json:"currency"`
	Provider    PaymentProvider `json:"provider"`
	Description string          `json:"description"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

type PaymentResponse struct {
	ID            string        `json:"id"`
	Status        PaymentStatus `json:"status"`
	TransactionID string        `json:"transaction_id,omitempty"`
	Error         string        `json:"error,omitempty"`
}

type Logger interface {
	LogPayment(payment *Payment) error
	LogError(operation string, err error) error
}

type PaymentProcessor interface {
	ProcessPayment(request *PaymentRequest) (*PaymentResponse, error)
	GetPaymentStatus(id string) (*Payment, error)
	CancelPayment(id string) error
}

func NewPaymentProcessor(logger Logger) PaymentProcessor {
	return &paymentProcessor{}
}

type paymentProcessor struct{}

func (p *paymentProcessor) ProcessPayment(request *PaymentRequest) (*PaymentResponse, error) {
	return &PaymentResponse{}, nil
}

func (p *paymentProcessor) GetPaymentStatus(id string) (*Payment, error) {
	return &Payment{}, nil
}

func (p *paymentProcessor) CancelPayment(id string) error {
	return nil
}
