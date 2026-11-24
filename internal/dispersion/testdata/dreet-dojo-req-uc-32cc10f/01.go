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
	Prereq    string
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
	PrereqCardID   string
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
	StatusPass    AttemptStatus = "pass"
	StatusGiveup  AttemptStatus = "giveup"
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
	CheckPass         CheckStatus = "pass"
	CheckCompileError CheckStatus = "compile_error"
	CheckTestFail     CheckStatus = "test_fail"
	CheckTimeout      CheckStatus = "timeout"
)

type Severity string

const (
	SevError   Severity = "error"
	SevWarning Severity = "warning"
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
	Status        CheckStatus
}

type NextOptions struct {
	NewPerDay int
}

type Stats struct {
	QueueToday       int
	ProcessedToday   int
	PerfectAttemptRate float64
	WeakTags         []TagStat
}

type TagStat struct {
	Tag            string
	AverageGrade   float64
}

type CardView struct {
	Meta       CardMeta
	Prompt     string
	CurrentCode string
	StartedAt  time.Time
	HintsShown []string
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

type Service interface {
	Next(ctx context.Context) (CardView, error)
	Current(ctx context.Context) (CardView, error)
	Run(ctx context.Context, cardID string, code string) (CheckResult, error)
	Hint(ctx context.Context) (string, error)
	Giveup(ctx context.Context, cardID string, code string) error
	Diff(ctx context.Context) (userCode, solutionCode string, err error)
}

func EvaluateGrade(in AttemptInput) Grade {}
func ScheduleSM2(st CardState, g Grade, now time.Time, rnd func() float64) CardState {}
