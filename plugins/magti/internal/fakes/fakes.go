package fakes

import (
	"context"
	"sync"
	"time"

	"github.com/abgeo/maroid/libs/pluginapi"
	"github.com/abgeo/maroid/libs/rest/page"
	"github.com/abgeo/maroid/libs/rest/precondition"
	"github.com/abgeo/maroid/plugins/magti/errs"
	"github.com/abgeo/maroid/plugins/magti/internal/fakeportal"
	"github.com/abgeo/maroid/plugins/magti/model"
	"github.com/abgeo/maroid/plugins/magti/repository"
)

// The workspace and the credentials of the tests.
const (
	WorkspaceW = "0199c5e2-8f3a-7c21-9b4e-2f6a1d0c5e78"
	Username   = "599000001"
	Password   = "p4ss-word"

	secretSession = "session"
)

// Acting answers a context that acts in workspace W.
func Acting(ctx context.Context) context.Context {
	return pluginapi.ContextWithActingWorkspace(ctx, WorkspaceW)
}

// Secrets is a PluginSecrets in memory, keyed by the acting workspace and the name.
type Secrets struct {
	mu      sync.Mutex
	entries map[string]string
}

// NewSecrets answers secrets that hold the session with the cookie of the fake portal
// for workspace W, or none for an empty cookie.
func NewSecrets(cookie string) *Secrets {
	secrets := &Secrets{entries: map[string]string{}}

	if cookie != "" {
		stored, _ := model.Session{
			Cookies: map[string]string{fakeportal.CookieName: cookie},
		}.MarshalSecret()
		secrets.entries[WorkspaceW+"/"+secretSession] = stored
	}

	return secrets
}

func secretKey(ctx context.Context, name string) string {
	return pluginapi.ActingWorkspaceFromContext(ctx) + "/" + name
}

// Get answers the secret, or pluginapi.ErrSecretAbsent.
func (f *Secrets) Get(ctx context.Context, _ pluginapi.SecretScope, name string) (string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	value, found := f.entries[secretKey(ctx, name)]
	if !found {
		return "", pluginapi.ErrSecretAbsent
	}

	return value, nil
}

// Put stores the secret.
func (f *Secrets) Put(
	ctx context.Context,
	scope pluginapi.SecretScope,
	name string,
	value string,
) error {
	return f.PutUntil(ctx, scope, name, value, time.Time{})
}

// PutUntil stores the secret, and ignores the expiry.
func (f *Secrets) PutUntil(
	ctx context.Context,
	_ pluginapi.SecretScope,
	name string,
	value string,
	_ time.Time,
) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.entries[secretKey(ctx, name)] = value

	return nil
}

// Delete removes the secret.
func (f *Secrets) Delete(ctx context.Context, _ pluginapi.SecretScope, name string) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	delete(f.entries, secretKey(ctx, name))

	return nil
}

// Session answers the stored session of workspace W, empty when none is stored.
func (f *Secrets) Session() string {
	f.mu.Lock()
	defer f.mu.Unlock()

	return f.entries[WorkspaceW+"/"+secretSession]
}

// Cookie answers the session cookie of workspace W, empty when none is stored.
func (f *Secrets) Cookie() string {
	session, err := model.ParseSession(f.Session())
	if err != nil {
		return ""
	}

	return session.Cookies[fakeportal.CookieName]
}

// Settings answers the stored credentials, or pluginapi.ErrSettingsAbsent with none.
type Settings struct {
	Values map[string]any
}

// Get answers the values.
func (f *Settings) Get(context.Context) (map[string]any, error) {
	if f.Values == nil {
		return nil, pluginapi.ErrSettingsAbsent
	}

	return f.Values, nil
}

// Credentials answers the settings of workspace W.
func Credentials() *Settings {
	return &Settings{Values: map[string]any{"username": Username, "password": Password}}
}

// Store holds the assignment, the traffic, and the money entries of workspace W in
// memory. A page read answers every row in one page.
type Store struct {
	mu         sync.Mutex
	assignment *model.Assignment
	// Days and Entries hold what the collections stored.
	Days    map[string]model.TrafficDay
	Entries []model.MoneyEntry
	// Writes counts the calls of SaveCollection.
	Writes int
}

// NewStore answers a store with the assignment of the number, or none for an empty one.
func NewStore(number string, kind model.SubscriberKind) *Store {
	store := &Store{Days: map[string]model.TrafficDay{}}

	if number != "" {
		store.assignment = &model.Assignment{
			SubscriberNumber: number,
			Kind:             kind,
			AssignedAt:       time.Date(2026, time.November, 20, 8, 0, 0, 0, time.UTC),
		}
	}

	return store
}

// Assignment answers the assignment, or errs.ErrNotAssigned.
func (f *Store) Assignment(context.Context) (model.Assignment, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	if f.assignment == nil {
		return model.Assignment{}, errs.ErrNotAssigned
	}

	return *f.assignment, nil
}

// Assign replaces the assignment, under the validator when it names one.
func (f *Store) Assign(
	_ context.Context,
	number string,
	kind model.SubscriberKind,
	ifMatch *time.Time,
) (model.Assignment, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	if ifMatch != nil && (f.assignment == nil || !f.assignment.AssignedAt.Equal(*ifMatch)) {
		return model.Assignment{}, precondition.ErrModified
	}

	assignedAt := time.Now().UTC()
	if f.assignment != nil && !assignedAt.After(f.assignment.AssignedAt) {
		assignedAt = f.assignment.AssignedAt.Add(time.Microsecond)
	}

	f.assignment = &model.Assignment{SubscriberNumber: number, Kind: kind, AssignedAt: assignedAt}

	return *f.assignment, nil
}

// Unassign removes the assignment.
func (f *Store) Unassign(context.Context) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.assignment = nil

	return nil
}

// SaveCollection keeps the rows, and counts the entries that it did not hold.
func (f *Store) SaveCollection(
	_ context.Context,
	days []model.TrafficDay,
	entries []model.MoneyEntry,
) (model.Collection, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.Writes++

	for _, day := range days {
		f.Days[day.Day.Format(time.DateOnly)] = day
	}

	collection := model.Collection{Days: len(days)}

	for _, entry := range entries {
		if !holds(f.Entries, entry) {
			f.Entries = append(f.Entries, entry)
			collection.NewEntries++
		}
	}

	return collection, nil
}

func holds(entries []model.MoneyEntry, entry model.MoneyEntry) bool {
	for _, held := range entries {
		if held.OccurredAt.Equal(entry.OccurredAt) && held.Kind == entry.Kind &&
			held.Direction == entry.Direction && held.Amount == entry.Amount {
			return true
		}
	}

	return false
}

// TrafficPage answers no row, after the checks of the assignment.
func (f *Store) TrafficPage(
	context.Context,
	page.Request,
	model.Period,
	repository.Order,
) (page.Window[model.TrafficDay], error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	return page.Window[model.TrafficDay]{}, f.mobile()
}

// MoneyPage answers no row, after the checks of the assignment.
func (f *Store) MoneyPage(
	context.Context,
	page.Request,
	repository.Order,
) (page.Window[model.MoneyEntry], error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	return page.Window[model.MoneyEntry]{}, f.mobile()
}

// MonthTotals answers no month, after the checks of the assignment.
func (f *Store) MonthTotals(context.Context, model.Month) ([]model.MonthTotal, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	return nil, f.mobile()
}

func (f *Store) mobile() error {
	if f.assignment == nil {
		return errs.ErrNotAssigned
	}

	if f.assignment.IsHomeLine() {
		return errs.ErrHomeLine
	}

	return nil
}
