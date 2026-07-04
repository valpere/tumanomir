package payment

import (
	"context"
	"time"
)

type Provider interface {
	Process(ctx context.Context, tx Transaction) (Result, error)
	Name() string
}

type Transaction struct {
	ID        string
	Amount    float64
	Currency  string
	Provider  string
	UserID    string
	CreatedAt time.Time
}

type Result struct {
	Success   bool
	TxID      string
	ErrorCode string
	Message   string
	Timestamp time.Time
}

type Status string

const (
	StatusSuccess Status = "success"
	StatusFailed  Status = "failed"
	StatusPending Status = "pending"
)

type Operation struct {
	ID        string
	Provider  string
	Status    Status
	Result    Result
	RecordedAt time.Time
}

type Logger interface {
	Log(ctx context.Context, op Operation) error
}

type Processor struct {
	providers map[string]Provider
	logger    Logger
}

func NewProcessor(logger Logger) *Processor {
	return &Processor{}
}

func (p *Processor) Register(provider Provider) {}

func (p *Processor) Process(ctx context.Context, tx Transaction) (Result, error) {
	return Result{}, nil
}
