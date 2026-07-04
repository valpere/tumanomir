package payments

type TransactionStatus string

const (
	StatusSuccess TransactionStatus = "success"
	StatusFailed  TransactionStatus = "failed"
)

type Provider interface {
	Name() string
	Process(payment Payment) (Result, error)
}

type Payment struct {
	ID       string
	Provider string
	Amount   float64
	Currency string
	UserID   string
}

type Result struct {
	TransactionID string
	Status        TransactionStatus
	Message       string
}

type Logger interface {
	LogSuccess(result Result)
	LogFailure(result Result, err error)
}

type PaymentSystem struct {
	providers map[string]Provider
	logger    Logger
}

func NewPaymentSystem(logger Logger) *PaymentSystem {}

func (s *PaymentSystem) RegisterProvider(provider Provider) {}

func (s *PaymentSystem) ProcessPayment(payment Payment) (Result, error) {}
