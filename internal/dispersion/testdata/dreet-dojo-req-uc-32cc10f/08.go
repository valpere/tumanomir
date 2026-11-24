package dreet

import (
	"context"
	"time"
)

// --- Domain Types ---

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
	CardID          string
	DeckID          string
	PrereqCardID    string
	SortOrder       int
	Tags            []string
	Reps            int
	Lapses          int
	IntervalDays    float64
	EaseFactor      float64
	DueAt           time.Time
	LastReviewedAt  *time.Time
	Active          bool
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
	Status        CheckStatus
}

type Stats struct {
	TodayQueue       int
	ReviewedToday    int
	CleanAttemptRate float64
	WeakTags         []TagStat
}

type TagStat struct {
	Tag     string
	AvgGrade float64
}

type NextOptions struct {
	NewPerDay int
	PracticeAhead bool
}

type CardView struct {
	Meta     CardMeta
	Code     string
	Prompt   string
	Hints    []string
	StartedAt time.Time
}

// --- Service Interfaces ---

type Loader interface {
	LoadAll(rootDir string) ([]CardMeta, error)
}

type Runner interface {
	Check(ctx context.Context, workDir string) (CheckResult, error)
}

type Store interface {
	NextDue(ctx context.Context, now time.Time, opts NextOptions) (CardState, error)
	RecordAttempt(ctx context.Context, att Attempt, state CardState) error
	Stats(ctx context.Context, now time.Time, newPerDay int) (Stats, error)
}

type Manager interface {
	Prepare(ctx context.Context, meta CardMeta) (string, error)
	WriteCode(ctx context.Context, workDir, fileName, code string) error
}

type SessionService interface {
	Next(ctx context.Context) (CardView, error)
	Current(ctx context.Context) (CardView, error)
	Run(ctx context.Context, cardID, code string) (CheckResult, error)
	Hint(ctx context.Context) (string, error)
	Giveup(ctx context.Context, cardID, code string) error
	Diff(ctx context.Context) (userCode, solutionCode string, err error)
}

// --- Pure Logic Functions ---

func EvaluateGrade(in AttemptInput) Grade {}

func ScheduleSM2(st CardState, g Grade, now time.Time, rnd func() float64) CardState {}

// --- CLI / Entry points ---

func ValidateDecks(decksDir string) (int, error) {}
