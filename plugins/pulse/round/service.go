package round

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"slices"
	"time"

	"github.com/jmoiron/sqlx"

	"github.com/abgeo/maroid/libs/pluginapi"
	"github.com/abgeo/maroid/libs/pluginapi/telegram/message"
	"github.com/abgeo/maroid/plugins/pulse/errs"
	"github.com/abgeo/maroid/plugins/pulse/model"
	"github.com/abgeo/maroid/plugins/pulse/repository"
)

// Plan is what one tick decided: the moment that fires, the check-ins that fire at
// it, and the mark of each check-in that skips an earlier occurrence.
type Plan struct {
	Moment time.Time
	Fire   []model.Checkin
	Skip   map[string]time.Time
}

// Closed is a round that expired, with the prompts that waited in it.
type Closed struct {
	Round   model.Round
	Waiting []Waiting
}

// Waiting is a prompt that waited when its round closed, with its question.
type Waiting struct {
	Prompt   model.Prompt
	Question model.Question
}

// Service runs the rounds of the acting user.
type Service struct {
	logger    *slog.Logger
	db        pluginapi.PluginDB
	messenger message.Messenger
	now       func() time.Time
}

// NewService creates a Service. The clock answers the moment of each answer and
// each claim.
func NewService(
	logger *slog.Logger,
	db pluginapi.PluginDB,
	messenger message.Messenger,
	now func() time.Time,
) *Service {
	return &Service{logger: logger, db: db, messenger: messenger, now: now}
}

// CloseOpen marks the open rounds expired, with their prompts, inside the
// transaction. A moment keeps the rounds at or after it open. Expire runs after the
// commit with the result.
func CloseOpen(ctx context.Context, tx *sqlx.Tx, before *time.Time) ([]Closed, error) {
	rounds, err := repository.NewRound(tx).ExpireOpen(ctx, before)
	if err != nil {
		return nil, fmt.Errorf("expiring the open rounds: %w", err)
	}

	if len(rounds) == 0 {
		return nil, nil
	}

	ids := make([]string, 0, len(rounds))
	for _, round := range rounds {
		ids = append(ids, round.ID)
	}

	waited, err := repository.NewPrompt(tx).ExpireOf(ctx, ids)
	if err != nil {
		return nil, fmt.Errorf("expiring the prompts of the rounds: %w", err)
	}

	questions, err := questionsByID(ctx, tx, promptQuestions(waited))
	if err != nil {
		return nil, err
	}

	closed := make([]Closed, 0, len(rounds))

	for _, round := range rounds {
		item := Closed{Round: round}

		for _, prompt := range waited {
			if prompt.RoundID == round.ID {
				item.Waiting = append(
					item.Waiting,
					Waiting{Prompt: prompt, Question: questions[prompt.QuestionID]},
				)
			}
		}

		closed = append(closed, item)
	}

	return closed, nil
}

// Fire runs the plan: it expires each earlier open round, stores a round for each
// check-in that fires, moves every mark, and then sends the messages.
func (s *Service) Fire(ctx context.Context, plan Plan) error {
	var (
		closed []Closed
		fired  []model.Round
	)

	err := s.db.WithTx(ctx, func(tx *sqlx.Tx) error {
		var txErr error

		// A round that fires expires every earlier open round.
		if closed, txErr = CloseOpen(ctx, tx, &plan.Moment); txErr != nil {
			return txErr
		}

		if fired, txErr = insertRounds(ctx, tx, plan); txErr != nil {
			return txErr
		}

		return moveMarks(ctx, tx, plan)
	})
	if err != nil {
		return fmt.Errorf("firing the check-ins: %w", err)
	}

	s.Expire(ctx, closed)

	var sendErrs []error

	for _, round := range fired {
		if sendErr := s.offer(ctx, round); sendErr != nil {
			sendErrs = append(sendErrs, sendErr)
		}
	}

	return errors.Join(sendErrs...)
}

// Expire edits the messages of the rounds that closed, and releases their claims. A
// failure is logged with the round, and the expiry goes on.
func (s *Service) Expire(ctx context.Context, closed []Closed) {
	for _, item := range closed {
		if item.Round.StartMessageID != nil {
			s.edit(ctx, item.Round.ID, *item.Round.StartMessageID,
				ClosedText(item.Round.CheckinName, model.RoundStateExpired))
		}

		for _, waiting := range item.Waiting {
			if waiting.Prompt.MessageID == nil {
				continue
			}

			messageID := *waiting.Prompt.MessageID
			s.edit(ctx, item.Round.ID, messageID, ClosedQuestionText(&waiting.Question, "expired"))
			s.closeClaim(ctx, item.Round.ID, &waiting.Question, messageID, LocationExpired)
		}
	}
}

// Press handles a press on a start prompt or on a question, and answers the notice
// that the client of the person shows.
func (s *Service) Press(ctx context.Context, press message.Press) (string, error) {
	err := s.press(ctx, press)
	if errors.Is(err, errs.ErrRoundClosed) {
		return ClosedNotice, nil
	}

	if err != nil {
		return "", err
	}

	return "", nil
}

// Reply handles a typed reply or a shared location on a claimed question.
func (s *Service) Reply(ctx context.Context, reply message.Reply) error {
	var (
		prompt *model.Prompt
		asked  *model.Question
	)

	err := s.inTx(ctx, "finding the replied question", func(tx *sqlx.Tx) error {
		var findErr error

		if prompt, findErr = repository.NewPrompt(tx).
			GetByMessage(ctx, string(reply.Message)); findErr != nil {
			return fmt.Errorf("finding the prompt of the message: %w", findErr)
		}

		asked, findErr = loadQuestion(ctx, tx, prompt.QuestionID)

		return findErr
	})

	switch {
	case errors.Is(err, sql.ErrNoRows):
		return s.release(ctx, reply.Message)
	case err != nil:
		return err
	case prompt.State != model.PromptStateWaiting:
		return s.release(ctx, reply.Message)
	}

	answer, retry := ParseReply(asked, reply)
	if retry != "" {
		if _, sendErr := s.messenger.Send(ctx, message.Message{Text: retry}); sendErr != nil {
			return fmt.Errorf("sending the retry of the prompt %s: %w", prompt.ID, sendErr)
		}

		return nil
	}

	if err = s.step(ctx, prompt, asked, answer); errors.Is(err, errs.ErrRoundClosed) {
		return s.release(ctx, reply.Message)
	}

	return err
}

// offer sends the start prompt of a stored round. A person whom Telegram does not
// reach gets the round expired, and the tick goes on.
func (s *Service) offer(ctx context.Context, round model.Round) error {
	id, err := s.messenger.Send(ctx, message.Message{
		Text: StartText(round.CheckinName, round.QuestionCount),
		Buttons: [][]message.Button{{
			{Label: "Start", Value: ValueStart},
			{Label: "Decline", Value: ValueDecline},
		}},
	})

	switch {
	case errors.Is(err, message.ErrUnreachable), errors.Is(err, message.ErrNoAccount):
		s.logger.WarnContext(ctx, "the person cannot receive the start prompt",
			slog.String("user", pluginapi.ActingUserFromContext(ctx)),
			slog.String("round", round.ID))

		return s.inTx(ctx, "expiring the unreachable round", func(tx *sqlx.Tx) error {
			_, moveErr := repository.NewRound(tx).
				Move(ctx, round.ID, model.RoundStateOffered, model.RoundStateExpired)
			if moveErr != nil {
				return fmt.Errorf("expiring the round %s: %w", round.ID, moveErr)
			}

			return nil
		})
	case err != nil:
		return fmt.Errorf("sending the start prompt of the round %s: %w", round.ID, err)
	}

	return s.inTx(ctx, "storing the start prompt", func(tx *sqlx.Tx) error {
		return repository.NewRound(tx).SetStartMessage(ctx, round.ID, string(id))
	})
}

func (s *Service) press(ctx context.Context, press message.Press) error {
	messageID := string(press.Message)

	var (
		round  *model.Round
		prompt *model.Prompt
		asked  *model.Question
	)

	err := s.inTx(ctx, "finding the pressed message", func(tx *sqlx.Tx) error {
		var findErr error

		round, findErr = repository.NewRound(tx).GetByStartMessage(ctx, messageID)

		switch {
		case findErr == nil:
			return nil
		case !errors.Is(findErr, sql.ErrNoRows):
			return fmt.Errorf("finding the round of the message: %w", findErr)
		}

		round = nil

		prompt, findErr = repository.NewPrompt(tx).GetByMessage(ctx, messageID)
		if findErr != nil {
			return fmt.Errorf("finding the prompt of the message: %w", findErr)
		}

		asked, findErr = loadQuestion(ctx, tx, prompt.QuestionID)

		return findErr
	})

	switch {
	case errors.Is(err, sql.ErrNoRows):
		return errs.ErrRoundClosed
	case err != nil:
		return err
	case round != nil:
		return s.pressStart(ctx, round, press.Value)
	}

	answer, ok := ParsePress(asked, press.Value)
	if !ok {
		return errs.ErrRoundClosed
	}

	return s.step(ctx, prompt, asked, answer)
}

func (s *Service) pressStart(ctx context.Context, round *model.Round, value string) error {
	switch value {
	case ValueStart:
		return s.start(ctx, round)
	case ValueDecline:
		return s.decline(ctx, round)
	default:
		return errs.ErrRoundClosed
	}
}

// start moves an offered round to started and sends its first question.
func (s *Service) start(ctx context.Context, round *model.Round) error {
	var (
		first *model.Prompt
		asked *model.Question
	)

	err := s.inTx(ctx, "starting the round", func(tx *sqlx.Tx) error {
		started, moveErr := repository.NewRound(tx).
			Move(ctx, round.ID, model.RoundStateOffered, model.RoundStateStarted)
		if moveErr != nil {
			return fmt.Errorf("starting the round %s: %w", round.ID, moveErr)
		}

		*round = *started

		if first, moveErr = repository.NewPrompt(tx).WaitNext(ctx, round.ID); moveErr != nil {
			return fmt.Errorf("finding the first prompt of the round %s: %w", round.ID, moveErr)
		}

		asked, moveErr = loadQuestion(ctx, tx, first.QuestionID)

		return moveErr
	})
	if errors.Is(err, sql.ErrNoRows) {
		return errs.ErrRoundClosed
	}

	if err != nil {
		return err
	}

	s.edit(
		ctx,
		round.ID,
		*round.StartMessageID,
		StartedText(round.CheckinName, round.QuestionCount),
	)

	return s.ask(ctx, first, asked)
}

func (s *Service) decline(ctx context.Context, round *model.Round) error {
	err := s.inTx(ctx, "declining the round", func(tx *sqlx.Tx) error {
		_, moveErr := repository.NewRound(tx).
			Move(ctx, round.ID, model.RoundStateOffered, model.RoundStateDeclined)
		if moveErr != nil {
			return fmt.Errorf("declining the round %s: %w", round.ID, moveErr)
		}

		return nil
	})
	if errors.Is(err, sql.ErrNoRows) {
		return errs.ErrRoundClosed
	}

	if err != nil {
		return err
	}

	s.edit(
		ctx,
		round.ID,
		*round.StartMessageID,
		ClosedText(round.CheckinName, model.RoundStateDeclined),
	)

	return nil
}

// step closes the waiting prompt with the answer, records its value, and moves the
// round to its next question or to done, in one transaction. The messages follow
// the commit.
func (s *Service) step(
	ctx context.Context,
	prompt *model.Prompt,
	asked *model.Question,
	answer Answer,
) error {
	var (
		round    *model.Round
		next     *model.Prompt
		nextAsks *model.Question
	)

	err := s.inTx(ctx, "recording the answer", func(tx *sqlx.Tx) error {
		var txErr error

		round, next, nextAsks, txErr = s.record(ctx, tx, prompt, answer)

		return txErr
	})

	switch {
	case errors.Is(err, sql.ErrNoRows), errors.Is(err, errs.ErrPromptAnswered):
		return errs.ErrRoundClosed
	case err != nil:
		return err
	}

	messageID := *prompt.MessageID
	s.edit(ctx, round.ID, messageID, ClosedQuestionText(asked, answer.Shown))

	after := LocationSaved
	if answer.Skip {
		after = LocationSkipped
	}

	s.closeClaim(ctx, round.ID, asked, messageID, after)

	if next != nil {
		return s.ask(ctx, next, nextAsks)
	}

	s.edit(
		ctx,
		round.ID,
		*round.StartMessageID,
		ClosedText(round.CheckinName, model.RoundStateDone),
	)

	return nil
}

// record is the transaction of step. It answers the round, and the next prompt
// with its question, or no prompt when the round is done.
func (s *Service) record(
	ctx context.Context,
	tx *sqlx.Tx,
	prompt *model.Prompt,
	answer Answer,
) (*model.Round, *model.Prompt, *model.Question, error) {
	state := model.PromptStateAnswered
	if answer.Skip {
		state = model.PromptStateSkipped
	}

	// The update on the expected state lets one press win.
	closed, err := repository.NewPrompt(tx).Close(ctx, prompt.ID, state)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("closing the prompt %s: %w", prompt.ID, err)
	}

	round, err := repository.NewRound(tx).GetByID(ctx, closed.RoundID)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("reading the round %s: %w", closed.RoundID, err)
	}

	if !answer.Skip {
		if err = s.insertValue(ctx, tx, closed, round, answer); err != nil {
			return nil, nil, nil, err
		}
	}

	next, err := repository.NewPrompt(tx).WaitNext(ctx, round.ID)
	if errors.Is(err, sql.ErrNoRows) {
		// A round with no question left is done, also when every
		// question was skipped.
		done, moveErr := repository.NewRound(tx).
			Move(ctx, round.ID, model.RoundStateStarted, model.RoundStateDone)
		if moveErr != nil {
			return nil, nil, nil, fmt.Errorf("ending the round %s: %w", round.ID, moveErr)
		}

		return done, nil, nil, nil
	}

	if err != nil {
		return nil, nil, nil, fmt.Errorf(
			"finding the next prompt of the round %s: %w",
			round.ID,
			err,
		)
	}

	asks, err := loadQuestion(ctx, tx, next.QuestionID)

	return round, next, asks, err
}

func (s *Service) insertValue(
	ctx context.Context,
	tx *sqlx.Tx,
	prompt *model.Prompt,
	round *model.Round,
	answer Answer,
) error {
	asked, err := loadQuestion(ctx, tx, prompt.QuestionID)
	if err != nil {
		return err
	}

	// A value carries the moment of its occurrence, of its answer, and
	// its source.
	err = repository.NewValue(tx).Insert(ctx, &model.Value{
		QuestionID: prompt.QuestionID,
		Type:       asked.Type,
		OccurredAt: round.OccurrenceAt,
		AnsweredAt: s.now(),
		Source:     model.SourceTelegram,
		PromptID:   &prompt.ID,
		Number:     answer.Number,
		Boolean:    answer.Boolean,
		ChoiceID:   answer.ChoiceID,
		Text:       answer.Text,
		Latitude:   answer.Latitude,
		Longitude:  answer.Longitude,
	})
	if err != nil {
		return fmt.Errorf("storing the value of the prompt %s: %w", prompt.ID, err)
	}

	return nil
}

// ask sends the question of the prompt, and stores its message.
func (s *Service) ask(ctx context.Context, prompt *model.Prompt, asked *model.Question) error {
	id, err := s.messenger.Send(ctx, QuestionMessage(asked, s.now()))
	if err != nil {
		return fmt.Errorf("sending the prompt %s: %w", prompt.ID, err)
	}

	return s.inTx(ctx, "storing the message of the prompt", func(tx *sqlx.Tx) error {
		return repository.NewPrompt(tx).SetMessage(ctx, prompt.ID, string(id))
	})
}

// closeClaim releases the claim of a question that claims one. A location question
// also sends the text that removes its keyboard.
func (s *Service) closeClaim(
	ctx context.Context,
	roundID string,
	asked *model.Question,
	messageID string,
	removal string,
) {
	if !asked.Type.ClaimsReply() {
		return
	}

	if err := s.messenger.Release(ctx, message.ID(messageID)); err != nil {
		s.logFailure(ctx, "failed to release the claim of a question", roundID, err)
	}

	if asked.Type != model.QuestionTypeLocation {
		return
	}

	if _, err := s.messenger.Send(
		ctx,
		message.Message{Text: removal, RemoveKeyboard: true},
	); err != nil {
		s.logFailure(ctx, "failed to remove the location keyboard", roundID, err)
	}
}

func (s *Service) edit(ctx context.Context, roundID string, messageID string, text string) {
	if err := s.messenger.Edit(ctx, message.ID(messageID), text, nil); err != nil {
		s.logFailure(ctx, "failed to edit a message of a round", roundID, err)
	}
}

func (s *Service) release(ctx context.Context, id message.ID) error {
	if err := s.messenger.Release(ctx, id); err != nil {
		return fmt.Errorf("releasing the claim: %w", err)
	}

	return nil
}

func (s *Service) logFailure(ctx context.Context, text string, roundID string, err error) {
	s.logger.ErrorContext(ctx, text,
		slog.String("user", pluginapi.ActingUserFromContext(ctx)),
		slog.String("round", roundID),
		slog.Any("error", err))
}

func (s *Service) inTx(ctx context.Context, action string, work func(tx *sqlx.Tx) error) error {
	if err := s.db.WithTx(ctx, work); err != nil {
		return fmt.Errorf("%s: %w", action, err)
	}

	return nil
}

// insertRounds stores one offered round for each check-in that fires, in the order
// of the plan, and answers the rounds that this run stored.
func insertRounds(ctx context.Context, tx *sqlx.Tx, plan Plan) ([]model.Round, error) {
	rounds := repository.NewRound(tx)
	prompts := repository.NewPrompt(tx)

	var fired []model.Round

	for _, checkin := range plan.Fire {
		round := model.Round{
			CheckinID:     &checkin.ID,
			CheckinName:   checkin.Name,
			OccurrenceAt:  plan.Moment,
			QuestionCount: len(checkin.QuestionIDs),
		}

		// A second run of the same occurrence stores no round.
		inserted, err := rounds.Insert(ctx, &round)
		if err != nil {
			return nil, fmt.Errorf("storing the round of the check-in %s: %w", checkin.ID, err)
		}

		if !inserted {
			continue
		}

		if err = prompts.InsertAll(ctx, round.ID, checkin.QuestionIDs); err != nil {
			return nil, fmt.Errorf("storing the prompts of the round %s: %w", round.ID, err)
		}

		fired = append(fired, round)
	}

	return fired, nil
}

func moveMarks(ctx context.Context, tx *sqlx.Tx, plan Plan) error {
	checkins := repository.NewCheckin(tx)

	ids := make([]string, 0, len(plan.Fire))
	for _, checkin := range plan.Fire {
		ids = append(ids, checkin.ID)
	}

	if err := checkins.SetFiresAfter(ctx, ids, plan.Moment); err != nil {
		return fmt.Errorf("moving the marks of the fired check-ins: %w", err)
	}

	for id, moment := range plan.Skip {
		if err := checkins.SetFiresAfter(ctx, []string{id}, moment); err != nil {
			return fmt.Errorf("moving the mark of the check-in %s: %w", id, err)
		}
	}

	return nil
}

// loadQuestion reads a question with its choices.
func loadQuestion(ctx context.Context, tx *sqlx.Tx, id string) (*model.Question, error) {
	questions, err := questionsByID(ctx, tx, []string{id})
	if err != nil {
		return nil, err
	}

	asked, ok := questions[id]
	if !ok {
		return nil, sql.ErrNoRows
	}

	return &asked, nil
}

// questionsByID reads the questions with their choices, by id.
func questionsByID(
	ctx context.Context,
	tx *sqlx.Tx,
	ids []string,
) (map[string]model.Question, error) {
	found, err := repository.NewQuestion(tx).GetMany(ctx, ids)
	if err != nil {
		return nil, fmt.Errorf("reading the questions: %w", err)
	}

	choices, err := repository.NewChoice(tx).ListFor(ctx, ids)
	if err != nil {
		return nil, fmt.Errorf("reading the choices: %w", err)
	}

	questions := make(map[string]model.Question, len(found))

	for _, question := range found {
		for _, choice := range choices {
			if choice.QuestionID == question.ID {
				question.Choices = append(question.Choices, choice)
			}
		}

		questions[question.ID] = question
	}

	return questions, nil
}

func promptQuestions(prompts []model.Prompt) []string {
	ids := make([]string, 0, len(prompts))
	for _, prompt := range prompts {
		if !slices.Contains(ids, prompt.QuestionID) {
			ids = append(ids, prompt.QuestionID)
		}
	}

	return ids
}
