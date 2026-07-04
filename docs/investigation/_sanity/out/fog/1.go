package payment

import (
    "context"
    "time"
)

type PaymentProvider string

const (
    ProviderStripe PaymentProvider = "stripe"
    ProviderPayPal PaymentProvider = "paypal"
    ProviderBank   PaymentProvider = "bank"
)

type TransactionStatus string

const (
    StatusPending   TransactionStatus = "pending"
    StatusSuccess   TransactionStatus = "success"
    StatusFailed    TransactionStatus = "failed"
    StatusCancelled TransactionStatus = "cancelled"
)

type Transaction struct {
    ID          string            `json:"id"`
    Provider    PaymentProvider   `json:"provider"`
    Amount      float64           `json:"amount"`
    Currency    string            `json:"currency"`
    Status      TransactionStatus `json:"status"`
    Metadata    map[string]string `json:"metadata,omitempty"`
    CreatedAt   time.Time         `json:"created_at"`
    UpdatedAt   time.Time         `json:"updated_at"`
    Description string            `json:"description,omitempty"`
}

type PaymentRequest struct {
    Provider    PaymentProvider   `json:"provider"`
    Amount      float64           `json:"amount"`
    Currency    string            `json:"currency"`
    Metadata    map[string]string `json:"metadata,omitempty"`
    Description string            `json:"description,omitempty"`
}

type PaymentResponse struct {
    TransactionID string            `json:"transaction_id"`
    Status        TransactionStatus `json:"status"`
    Error         error             `json:"error,omitempty"`
    Provider      PaymentProvider   `json:"provider"`
    Amount        float64           `json:"amount"`
    Currency      string            `json:"currency"`
}

type Logger interface {
    LogTransaction(ctx context.Context, transaction *Transaction) error
    LogError(ctx context.Context, err error, transactionID string) error
}

type PaymentProcessor interface {
    ProcessPayment(ctx context.Context, request *PaymentRequest) (*PaymentResponse, error)
    GetTransaction(ctx context.Context, id string) (*Transaction, error)
    CancelTransaction(ctx context.Context, id string) error
}

type paymentProcessor struct{}

func (p *paymentProcessor) ProcessPayment(ctx context.Context, request *PaymentRequest) (*PaymentResponse, error) {
    return nil, nil
}

func (p *paymentProcessor) GetTransaction(ctx context.Context, id string) (*Transaction, error) {
    return nil, nil
}

func (p *paymentProcessor) CancelTransaction(ctx context.Context, id string) error {
    return nil
}
