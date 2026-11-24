package dreet

import (
	"context"
	"time"
)

type CardFormat string

const (
	FormatCloze       CardFormat = "cloze"
	FormatSpec        CardFormat = "spec"
	FormatProgressive CardFormat = "progressive"
	FormatBugfix      CardFormat = "bugfix"
	FormatSkeleton    CardFormat = "skeleton"
)

type CardMeta struct {
	ID        string
	Lang      string
	Title     string
	Format    CardFormat
	Prereq    *string
	Order     int
	TargetSec int
	Tags      []string
	EntryFile string
	Hints     []string
}

type CardFiles struct {
	Meta     string
	Prompt   string
	Module   string
	Stub     string
	Tests    []string
	Solution string
}

type CardState struct {
	CardID         string
	DeckID         string
	PrereqCardID   *string
	SortOrder      int
	Tags           []string
	Reps           int
	Lapses         int
	IntervalDays   float64
	EaseFactor     float64
	DueAt          time.Time
	LastReviewedAt *time.Time
	Active         bool
}

type AttemptStatus string

const (
	AttemptPass    AttemptStatus = "pass"
	AttemptGiveup   AttemptStatus = "giveup"
)

type Attempt struct {
	ID            int
	CardID        string
	StartedAt     time.Time
	FinishedAt    time.Time
	ElapsedSec    int
	TargetSec     int
	CompileErrors int
	TestFails     int
	HintsUsed     int
	Status        AttemptStatus
	Grade         int
}

type CheckStatus string

const (
	StatusPass         CheckStatus = "pass"
	StatusCompileError CheckStatus = "compile_error"
	StatusTestFail     CheckStatus = "test_fail"
	StatusTimeout      CheckStatus = "timeout"
)

type Severity string

const (
	SeverityError   Severity = "error"
	SeverityWarning Severity = "warning"
)

type Diagnostic struct {
	File     string
	Line     int
	Col      int
	Severity Severity
	Message  string
}

type CheckResult struct {
	Status      CheckStatus
	DurationMs  int
	Diagnostics []Diagnostic
	RawOutput   string
}

type Grade int

type AttemptInput struct {
	ElapsedSec    int
	TargetSec     int
	CompileErrors int
	TestFails     int
	HintsUsed     int
	Status        AttemptStatus
}

type NextOptions struct {
	NewPerDay int
}

type Stats struct {
	QueueToday           int
	ProcessedToday       int
	PerfectAttemptRatio   float64
	WeakestTags          []TagStat
}

type TagStat struct {
	Tag           string
	AverageGrade  float64
}

type CardView struct {
	Meta          CardMeta
	CurrentCode   string
	ShownHints    []string
	StartedAt     time.Time
	IsFinished    bool
	FinalGrade    int
	NextReviewIn  int
}

type Runner interface {
	Check(ctx context.Context, workDir string) (CheckResult, error)
}

type Loader interface {
	LoadAll(rootDir string) ([]CardMeta, error)
}

type Store interface {
	NextDue(ctx context.Context, now time.Time, opts NextOptions) (CardState, error)
	RecordAttempt(ctx context.Context, attempt Attempt) error
	Stats(ctx context.Context, now time.Time, newPerDay int) (Stats, error)
}

type Manager interface {
	Prepare(cardID string) error
	WriteCode(cardID string, code string) error
	ReadCode(cardID string) (string, error)
	ReadSolution(cardID string) (string, error)
	Cleanup(cardID string) error
}

type Service interface {
	Next(ctx context.Context) (CardView, error)
	Run(ctx context.Context, cardID string, code string) (CheckResult, error)
	Hint(ctx context.Context, cardID string) (string, error)
	GiveUp(ctx context.Context, cardID string, code string) (int, error)
	Diff(ctx context.Context, cardID string) (userCode, solutionCode string, err error)
	Current(ctx context.Context) (CardView, error)
}

func EvaluateGrade(in AttemptInput) Grade {}
func ScheduleSM2(st CardState, g Grade, now time.Time, rnd func() float64) CardState {}
