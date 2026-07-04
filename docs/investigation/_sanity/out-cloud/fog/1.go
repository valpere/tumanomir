package payment

type Provider string

type Transaction struct {
	ID            string
	Provider      Provider
	Amount        float64
	Currency      string
	PayerID       string
	PayeeID       string
	Description   string
}

type TransactionResult struct {
	TransactionID string
	Success       bool
	ErrorMessage  string
}

type Logger interface {
	LogSuccess(transactionID string, provider Provider, amount float64, currency string)
	LogFailure(transactionID string, provider Provider, error string)
}

type PaymentProcessor interface {
	Process(tx Transaction) TransactionResult
	Refund(transactionID string) TransactionResult
	SupportedProviders() []Provider
}

type ProviderFactory interface {
	Create(provider Provider) (PaymentProcessor, error)
}

type PaymentService struct {
	logger  Logger
	factory ProviderFactory
}

func NewPaymentService(logger Logger, factory ProviderFactory) *PaymentService {
	return &PaymentService{}
}

func (s *PaymentService) Execute(tx Transaction) TransactionResult {
	return TransactionResult{}
}

func (s *PaymentService) RegisterProcessor(provider Provider, processor PaymentProcessor) {}

type TransactionRepository interface {
	Save(tx Transaction) error
	FindByID(id string) (Transaction, error)
	UpdateResult(id string, result TransactionResult) error
}
