// Package errs defines the errors that the packages of pulse share.
package errs

import (
	"errors"
	"fmt"
)

// ErrPromptAnswered is returned when a value names a prompt that already holds one.
var ErrPromptAnswered = errors.New("value: the prompt already holds a value")

var (
	// ErrRoundClosed is returned for a press or a reply that another press, or a
	// tick, overtook.
	ErrRoundClosed = errors.New("round: the check-in is closed")
	// ErrPauseArgument is returned for an argument of /pause that is no date and no
	// duration.
	ErrPauseArgument = errors.New("pause: the argument is no date and no duration")
	// ErrClockInvalid is returned for a local time outside 00:00 to 23:59.
	ErrClockInvalid = errors.New("clock: the value is no time from 00:00 to 23:59")
	// ErrListType is returned when a column holds no comma list.
	ErrListType = errors.New("list: the column holds no comma list")
)

var (
	// ErrQuestionHasValues is returned for a change of the type, or a delete, of a
	// question that holds a value.
	ErrQuestionHasValues = errors.New("question: the question holds a value")
	// ErrChoiceInUse is returned when a change leaves out a choice that a value
	// records.
	ErrChoiceInUse = errors.New("question: a value records the choice")
	// ErrChoiceUnknown is returned for a choice id that the question does not hold.
	ErrChoiceUnknown = errors.New("question: the question holds no such choice")
	// ErrQuestionUnusable is returned when a check-in names a question that is
	// archived or unknown.
	ErrQuestionUnusable = errors.New("checkin: the question is archived or unknown")
)

// ChoiceUnknownError names the item of a list of choices whose id the question does
// not hold.
type ChoiceUnknownError struct {
	Index int
}

func (e *ChoiceUnknownError) Error() string {
	return fmt.Sprintf("%v: item %d", ErrChoiceUnknown, e.Index)
}

// Is matches ErrChoiceUnknown.
func (e *ChoiceUnknownError) Is(target error) bool {
	return target == ErrChoiceUnknown
}

// ChoiceInUseError names the choice that a change left out while a value records it.
type ChoiceInUseError struct {
	ID    string
	Label string
}

func (e *ChoiceInUseError) Error() string {
	return fmt.Sprintf("%v: %s", ErrChoiceInUse, e.ID)
}

// Is matches ErrChoiceInUse.
func (e *ChoiceInUseError) Is(target error) bool {
	return target == ErrChoiceInUse
}

// QuestionUnusableError names the item of the questions of a check-in that is
// archived or unknown.
type QuestionUnusableError struct {
	Index int
}

func (e *QuestionUnusableError) Error() string {
	return fmt.Sprintf("%v: item %d", ErrQuestionUnusable, e.Index)
}

// Is matches ErrQuestionUnusable.
func (e *QuestionUnusableError) Is(target error) bool {
	return target == ErrQuestionUnusable
}

// ErrPulseOff is returned for a pause or a resume while pulse is off for the person.
var ErrPulseOff = errors.New("settings: pulse is off")
