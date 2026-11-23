package catalog

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"

	"github.com/abgeo/maroid/plugins/pulse/model"
	"github.com/abgeo/maroid/plugins/pulse/repository"
)

// Seed copies the defaults to the acting user. Each check-in fires first after the
// moment now.
func Seed(ctx context.Context, tx *sqlx.Tx, now time.Time) error {
	defaults := Defaults()
	questions := repository.NewQuestion(tx)
	ids := make(map[string]string, len(defaults.Questions))

	for _, item := range defaults.Questions {
		question, err := item.question()
		if err != nil {
			return err
		}

		if err = questions.Insert(ctx, question); err != nil {
			return fmt.Errorf("copying the question %s: %w", item.Name, err)
		}

		ids[item.Name] = question.ID
	}

	checkins := repository.NewCheckin(tx)

	for _, item := range defaults.Checkins {
		checkin := &model.Checkin{
			Name:       item.Name,
			LocalTimes: item.Times,
			Weekdays:   item.Weekdays,
			Enabled:    true,
			FiresAfter: now,
		}

		for _, name := range item.Questions {
			checkin.QuestionIDs = append(checkin.QuestionIDs, ids[name])
		}

		if err := checkins.Insert(ctx, checkin); err != nil {
			return fmt.Errorf("copying the check-in %s: %w", item.Name, err)
		}
	}

	return nil
}

func (q DefaultQuestion) question() (*model.Question, error) {
	var config any = struct{}{}

	switch {
	case q.Scale != nil:
		config = q.Scale
	case q.Number != nil:
		config = q.Number
	}

	encoded, err := json.Marshal(config)
	if err != nil {
		return nil, fmt.Errorf("encoding the config of %s: %w", q.Name, err)
	}

	return &model.Question{Type: q.Type, Prompt: q.Prompt, Config: encoded}, nil
}
