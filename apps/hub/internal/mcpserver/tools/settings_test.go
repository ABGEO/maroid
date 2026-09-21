package tools_test

import (
	"context"
	"encoding/json"
	"io/fs"
	"log/slog"
	"slices"
	"strings"
	"testing"

	"github.com/jmoiron/sqlx"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/require"

	"github.com/abgeo/maroid/apps/hub/db"
	"github.com/abgeo/maroid/apps/hub/internal/domain/errs"
	"github.com/abgeo/maroid/apps/hub/internal/mcpserver/tools"
	"github.com/abgeo/maroid/apps/hub/internal/registry"
	"github.com/abgeo/maroid/apps/hub/internal/secret"
	"github.com/abgeo/maroid/apps/hub/internal/settings"
	"github.com/abgeo/maroid/libs/pluginapi"
	"github.com/abgeo/maroid/libs/testdb"
)

const (
	probePluginID   = "dev.maroid.probe"
	unknownPluginID = "dev.maroid.unknown"

	keyEmail    = "email"
	keyPassword = "password"

	valueEmail    = "person@example.com"
	valuePassword = "the-stored-credential"
	otherEmail    = "other@example.com"
	//nolint:gosec // G101: the scenario needs a literal that no log line may hold.
	newCredential = "the-new-credential"
)

// probeModel declares one required text field, one required secret field, and
// one optional switch.
type probeModel struct {
	Email    string `json:"email"    jsonschema:"title=Email,required"`
	Password string `json:"password" jsonschema:"title=Password,format=password,writeOnly=true,required"`
	Notify   bool   `json:"notify"   jsonschema:"title=Notify me"`
}

// fakeCipher marks a value instead of protecting it, so a scenario that reads the
// stored secret needs no protection service. PSET-SC-013 measures the protection.
type fakeCipher struct{}

func (fakeCipher) Encrypt(_ context.Context, key secret.Key, plaintext string) (string, error) {
	return string(key) + ":" + plaintext, nil
}

func (fakeCipher) Decrypt(_ context.Context, key secret.Key, ciphertext string) (string, error) {
	return strings.TrimPrefix(ciphertext, string(key)+":"), nil
}

// stubSettings answers each call with what the scenario gives it.
type stubSettings struct {
	document json.RawMessage
	secrets  []string
	values   map[string]any
	changed  []string
	failure  error
	saved    map[string]any
}

var _ settings.Service = (*stubSettings)(nil)

func (s *stubSettings) Declares(string) bool { return s.failure == nil }

func (s *stubSettings) Schema(string) (json.RawMessage, error) {
	return s.document, s.failure
}

func (s *stubSettings) SecretFields(string) ([]string, error) {
	return s.secrets, s.failure
}

func (s *stubSettings) ChangedSecrets(string, map[string]any) ([]string, error) {
	return s.changed, s.failure
}

func (s *stubSettings) Read(context.Context, string) (map[string]any, error) {
	return s.values, s.failure
}

func (s *stubSettings) Settings(
	context.Context,
	*pluginapi.PluginID,
) (map[string]any, error) {
	return s.values, s.failure
}

func (s *stubSettings) Save(_ context.Context, _ string, input map[string]any) error {
	if s.failure != nil {
		return s.failure
	}

	s.saved = input

	return nil
}

// session installs one tool on a server and connects a client to it. The
// middleware stands in for mcpserver.actingUserMiddleware, which carries the
// acting user of the verified token into the context of every handler.
func session(t *testing.T, entry registry.MCPTool, user string) *mcp.ClientSession {
	t.Helper()

	server := mcp.NewServer(&mcp.Implementation{Name: "test", Version: "0.0.1"}, nil)
	entry.Install(server)
	server.AddReceivingMiddleware(actingUser(user))

	client := mcp.NewClient(&mcp.Implementation{Name: "test-client", Version: "0.0.1"}, nil)
	serverTransport, clientTransport := mcp.NewInMemoryTransports()

	serverSession, err := server.Connect(t.Context(), serverTransport, nil)
	require.NoError(t, err)

	t.Cleanup(func() { _ = serverSession.Close() })

	clientSession, err := client.Connect(t.Context(), clientTransport, nil)
	require.NoError(t, err)

	t.Cleanup(func() { _ = clientSession.Close() })

	return clientSession
}

func actingUser(user string) mcp.Middleware {
	return func(next mcp.MethodHandler) mcp.MethodHandler {
		return func(ctx context.Context, method string, req mcp.Request) (mcp.Result, error) {
			return next(pluginapi.ContextWithActingUser(ctx, user), method, req)
		}
	}
}

func call(
	t *testing.T,
	entry registry.MCPTool,
	user string,
	arguments string,
) *mcp.CallToolResult {
	t.Helper()

	result, err := session(t, entry, user).CallTool(t.Context(), &mcp.CallToolParams{
		Name:      entry.Name,
		Arguments: json.RawMessage(arguments),
	})
	require.NoError(t, err)

	return result
}

// failureText returns the text that an agent reads from a failed call.
func failureText(t *testing.T, result *mcp.CallToolResult) string {
	t.Helper()

	require.True(t, result.IsError)
	require.Len(t, result.Content, 1)

	text, ok := result.Content[0].(*mcp.TextContent)
	require.True(t, ok)

	return text.Text
}

// content decodes the structured content of a result that carries no failure.
func content(t *testing.T, result *mcp.CallToolResult) map[string]any {
	t.Helper()

	require.False(t, result.IsError)

	encoded, err := json.Marshal(result.StructuredContent)
	require.NoError(t, err)

	var decoded map[string]any

	require.NoError(t, json.Unmarshal(encoded, &decoded))

	return decoded
}

func probeSchema(t *testing.T) *settings.Schema {
	t.Helper()

	schema, err := settings.Infer(&probeModel{})
	require.NoError(t, err)

	return schema
}

// probeManager builds a manager over the real schema. A scenario that reaches no
// row gives it no database and no protection service.
func probeManager(t *testing.T, handle *sqlx.DB) *settings.Manager {
	t.Helper()

	schemas := registry.NewSettingsRegistry()
	require.NoError(t, schemas.Register(pluginapi.ParsePluginID(probePluginID), probeSchema(t)))

	return settings.NewManager(handle, schemas, fakeCipher{})
}

// world holds a database with the migrations of the hub and two user records.
type world struct {
	instance *testdb.Instance
	manager  *settings.Manager
	userA    string
	userB    string
}

func newWorld(t *testing.T) *world {
	t.Helper()

	instance := testdb.Start(t)

	migrations, err := fs.Sub(db.GetMigrationsFS(), "migrations")
	require.NoError(t, err)

	instance.Migrate(t, "public", migrations)

	return &world{
		instance: instance,
		manager:  probeManager(t, instance.DB),
		userA:    addUser(t, instance, "Temuri"),
		userB:    addUser(t, instance, "Nino"),
	}
}

func addUser(t *testing.T, instance *testdb.Instance, firstName string) string {
	t.Helper()

	var id string

	require.NoError(t, instance.DB.Get(
		&id, `INSERT INTO public.users (first_name) VALUES ($1) RETURNING id;`, firstName,
	))

	return id
}

// store writes the settings of one user, the way the deck writes them.
func (w *world) store(t *testing.T, user string, input map[string]any) {
	t.Helper()

	ctx := pluginapi.ContextWithActingUser(t.Context(), user)
	require.NoError(t, w.manager.Save(ctx, probePluginID, input))
}

// storedSecret reads the secret of one user in its plaintext form.
func (w *world) storedSecret(t *testing.T, user string) string {
	t.Helper()

	ctx := pluginapi.ContextWithActingUser(t.Context(), user)

	values, err := w.manager.Settings(ctx, pluginapi.ParsePluginID(probePluginID))
	require.NoError(t, err)

	plaintext, ok := values[keyPassword].(string)
	require.True(t, ok)

	return plaintext
}

func logger() *slog.Logger {
	return slog.New(slog.DiscardHandler)
}

// recordingHandler keeps the text of every line that the logger writes, with the
// attributes that each one carries.
type recordingHandler struct {
	lines *[]string
	attrs []slog.Attr
}

func newRecorder() *recordingHandler {
	return &recordingHandler{lines: &[]string{}}
}

func (h *recordingHandler) Enabled(context.Context, slog.Level) bool { return true }

func (h *recordingHandler) Handle(_ context.Context, record slog.Record) error {
	line := strings.Builder{}
	line.WriteString(record.Message)

	for _, attr := range h.attrs {
		line.WriteString(" " + attr.String())
	}

	record.Attrs(func(attr slog.Attr) bool {
		line.WriteString(" " + attr.String())

		return true
	})

	*h.lines = append(*h.lines, line.String())

	return nil
}

func (h *recordingHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &recordingHandler{lines: h.lines, attrs: append(slices.Clone(h.attrs), attrs...)}
}

func (h *recordingHandler) WithGroup(string) slog.Handler { return h }

// MCPHUB-SC-017: One call reports the settings schema, the key of each secret
// field, and the values that the acting user stored.
func TestGetReportsTheSchemaTheSecretKeysAndTheValues(t *testing.T) {
	t.Parallel()

	service := &stubSettings{
		document: json.RawMessage(`{"type":"object","properties":{"email":{"type":"string"}}}`),
		secrets:  []string{keyPassword},
		values:   map[string]any{keyEmail: valueEmail, keyPassword: settings.SecretMask},
	}

	result := content(t, call(
		t,
		tools.NewGetPluginSettings(logger(), service),
		"user",
		`{"plugin":"`+probePluginID+`"}`,
	))

	require.Equal(t, probePluginID, result["plugin"])
	require.Equal(t, []any{keyPassword}, result["secretFields"])
	require.Equal(
		t,
		map[string]any{keyEmail: valueEmail, keyPassword: settings.SecretMask},
		result["values"],
	)

	schema, ok := result["schema"].(map[string]any)
	require.True(t, ok)
	require.Equal(t, "object", schema["type"])
}

// MCPHUB-SC-018: A plugin that declares no settings schema answers a failure that
// names the identifier that the call carried.
func TestGetNamesThePluginThatDeclaresNoSettings(t *testing.T) {
	t.Parallel()

	service := &stubSettings{failure: errs.ErrSettingsSchemaNotFound}

	result := call(
		t,
		tools.NewGetPluginSettings(logger(), service),
		"user",
		`{"plugin":"`+unknownPluginID+`"}`,
	)

	text := failureText(t, result)
	require.Contains(t, text, unknownPluginID)
	require.Contains(t, text, "declares no settings")
}

// MCPHUB-SC-020: A rejected save names each field that caused the rejection, with
// the reason of each one.
func TestSaveNamesEachFieldThatCausedTheRejection(t *testing.T) {
	t.Parallel()

	service := &stubSettings{
		failure: &settings.InvalidError{Fields: map[string]string{
			keyEmail: "the field is required",
			"colour": "the field is unknown",
		}},
	}

	result := call(
		t,
		tools.NewSavePluginSettings(logger(), service),
		"user",
		`{"plugin":"`+probePluginID+`","values":{"colour":"red"}}`,
	)

	text := failureText(t, result)
	require.Contains(t, text, keyEmail+": the field is required")
	require.Contains(t, text, "colour: the field is unknown")
}

// MCPHUB-SC-024: A call that carries a credential reaches no log line and no
// result that holds it. LOG-008 holds at this entry point.
func TestACredentialReachesNoLogLineAndNoResult(t *testing.T) {
	t.Parallel()

	recorder := newRecorder()
	entry := tools.NewSavePluginSettings(
		slog.New(recorder),
		probeManager(t, nil),
	)

	result := call(
		t,
		entry,
		"user",
		`{"plugin":"`+probePluginID+`","values":{"password":"`+newCredential+`"}}`,
	)

	text := failureText(t, result)
	require.Contains(t, text, keyPassword)
	require.Contains(t, text, "/plugins/"+probePluginID+"/settings")
	require.NotContains(t, text, newCredential)

	for _, line := range *recorder.lines {
		require.NotContains(t, line, newCredential)
	}
}

// MCPHUB-SC-019: A save that names one field keeps every other stored field, and
// the answer reports the row that the save left.
func TestSaveKeepsEveryFieldThatTheCallDoesNotName(t *testing.T) {
	t.Parallel()

	instance := newWorld(t)
	instance.store(t, instance.userA, map[string]any{
		keyEmail:    valueEmail,
		keyPassword: valuePassword,
	})

	result := content(t, call(
		t,
		tools.NewSavePluginSettings(logger(), instance.manager),
		instance.userA,
		`{"plugin":"`+probePluginID+`","values":{"email":"`+otherEmail+`"}}`,
	))

	values, ok := result["values"].(map[string]any)
	require.True(t, ok)
	require.Equal(t, otherEmail, values[keyEmail])
	require.Equal(t, settings.SecretMask, values[keyPassword])
	require.Equal(t, valuePassword, instance.storedSecret(t, instance.userA))
}

// MCPHUB-SC-021: A call that replaces a stored secret stores nothing, and it
// leaves every field of the row as it was.
func TestSaveStoresNothingWhenTheCallReplacesASecret(t *testing.T) {
	t.Parallel()

	instance := newWorld(t)
	instance.store(t, instance.userA, map[string]any{
		keyEmail:    valueEmail,
		keyPassword: valuePassword,
	})

	result := call(
		t,
		tools.NewSavePluginSettings(logger(), instance.manager),
		instance.userA,
		`{"plugin":"`+probePluginID+`","values":{"email":"`+otherEmail+`","password":"`+
			newCredential+`"}}`,
	)

	require.Contains(t, failureText(t, result), keyPassword)
	require.Equal(t, valuePassword, instance.storedSecret(t, instance.userA))

	stored := content(t, call(
		t,
		tools.NewGetPluginSettings(logger(), instance.manager),
		instance.userA,
		`{"plugin":"`+probePluginID+`"}`,
	))

	values, ok := stored["values"].(map[string]any)
	require.True(t, ok)
	require.Equal(t, valueEmail, values[keyEmail])
}

// MCPHUB-SC-022: The mask changes no secret, so the save stores every other field
// and the row keeps the secret that it held.
func TestSaveStoresTheOtherFieldsWhenTheCallMasksTheSecret(t *testing.T) {
	t.Parallel()

	instance := newWorld(t)
	instance.store(t, instance.userA, map[string]any{
		keyEmail:    valueEmail,
		keyPassword: valuePassword,
	})

	result := content(t, call(
		t,
		tools.NewSavePluginSettings(logger(), instance.manager),
		instance.userA,
		`{"plugin":"`+probePluginID+`","values":{"email":"`+otherEmail+`","password":"`+
			settings.SecretMask+`"}}`,
	))

	values, ok := result["values"].(map[string]any)
	require.True(t, ok)
	require.Equal(t, otherEmail, values[keyEmail])
	require.Equal(t, valuePassword, instance.storedSecret(t, instance.userA))
}

// MCPHUB-SC-025: Each user reads the values that they stored, and the values of
// the other user reach neither of them. OWN-006 holds at this entry point.
func TestEachUserReadsOnlyTheirOwnSettings(t *testing.T) {
	t.Parallel()

	instance := newWorld(t)
	instance.store(t, instance.userA, map[string]any{
		keyEmail:    valueEmail,
		keyPassword: valuePassword,
	})
	instance.store(t, instance.userB, map[string]any{
		keyEmail:    otherEmail,
		keyPassword: valuePassword,
	})

	entry := tools.NewGetPluginSettings(logger(), instance.manager)

	for user, want := range map[string]string{
		instance.userA: valueEmail,
		instance.userB: otherEmail,
	} {
		result := content(t, call(t, entry, user, `{"plugin":"`+probePluginID+`"}`))

		values, ok := result["values"].(map[string]any)
		require.True(t, ok)
		require.Equal(t, want, values[keyEmail])
	}
}
