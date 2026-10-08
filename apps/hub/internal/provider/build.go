package provider

import (
	"encoding/json"
	"fmt"
	"maps"
	"net/url"
	"regexp"
	"slices"
	"strings"
	"unicode/utf8"

	"github.com/abgeo/maroid/apps/hub/internal/dex"
)

// The fixed values of the presets. `spec.md` section 4.4 gives them.
const (
	localID          = "local"
	localName        = "Email"
	telegramID       = "telegram"
	telegramName     = "Telegram"
	telegramIssuer   = "https://oauth.telegram.org"
	telegramClaim    = "id"
	defaultClaim     = "sub"
	connectorLocal   = "local"
	connectorOIDC    = "oidc"
	maxNameRunes     = 64
	noInputDetail    = "the local preset takes no input"
	fixedFieldDetail = "the preset fixes this field"
)

var identifierPattern = regexp.MustCompile(`^[a-z][a-z0-9-]{1,31}$`)

func telegramScopes() []string { return []string{"openid", "profile"} }

func defaultScopes() []string { return []string{"openid", "profile", "email"} }

// build checks a new provider and answers the connector that Dex stores for it.
func (m *Manager) build(input Input) (dex.Connector, error) {
	switch input.Preset {
	case PresetLocal:
		return buildLocal(input)
	case PresetTelegram:
		return m.buildTelegram(input)
	case PresetOIDC:
		return m.buildOIDC(input)
	default:
		return dex.Connector{}, fieldError("/preset", "the preset is not local, telegram, or oidc")
	}
}

func buildLocal(input Input) (dex.Connector, error) {
	if pointer := firstPresent(input, "/id", "/name", "/issuer", "/client_id",
		"/client_secret", "/user_id_key", "/scopes", "/options"); pointer != "" {
		return dex.Connector{}, fieldError(pointer, noInputDetail)
	}

	return connectorOf(localID, connectorLocal, localName, map[string]any{
		MarkerKey: PresetLocal,
	})
}

func (m *Manager) buildTelegram(input Input) (dex.Connector, error) {
	if pointer := firstPresent(input, "/id", "/name", "/issuer", "/user_id_key",
		"/scopes"); pointer != "" {
		return dex.Connector{}, fieldError(pointer, fixedFieldDetail)
	}

	clientID := m.telegramBotID
	if input.ClientID != nil {
		clientID = strings.TrimSpace(*input.ClientID)
	}

	if clientID == "" {
		return dex.Connector{}, fieldError("/client_id", "the client identifier is empty")
	}

	config, err := clientConfig(input, clientID)
	if err != nil {
		return dex.Connector{}, err
	}

	config[keyIssuer] = telegramIssuer
	config[keyUserIDKey] = telegramClaim
	config[keyScopes] = telegramScopes()
	config[keyRedirectURI] = m.redirectURI
	config[MarkerKey] = PresetTelegram

	return connectorOf(telegramID, connectorOIDC, telegramName, config)
}

func (m *Manager) buildOIDC(input Input) (dex.Connector, error) {
	id, err := checkIdentifier(input.ID)
	if err != nil {
		return dex.Connector{}, err
	}

	name, err := checkName(input.Name)
	if err != nil {
		return dex.Connector{}, err
	}

	issuer, err := checkIssuer(input.Issuer)
	if err != nil {
		return dex.Connector{}, err
	}

	claim, err := checkClaim(input.UserIDKey)
	if err != nil {
		return dex.Connector{}, err
	}

	scopes := defaultScopes()
	if input.Scopes != nil {
		if scopes, err = checkScopes(*input.Scopes); err != nil {
			return dex.Connector{}, err
		}
	}

	config, err := clientConfig(input, strings.TrimSpace(value(input.ClientID)))
	if err != nil {
		return dex.Connector{}, err
	}

	config[keyIssuer] = issuer
	config[keyUserIDKey] = claim
	config[keyScopes] = scopes
	config[keyRedirectURI] = m.redirectURI
	config[MarkerKey] = PresetOIDC

	return connectorOf(id, connectorOIDC, name, config)
}

func checkIdentifier(id *string) (string, error) {
	checked := value(id)
	if !identifierPattern.MatchString(checked) || checked == localID || checked == telegramID {
		return "", fieldError("/id",
			"the identifier is 2 to 32 lower case letters, digits, or hyphens, and is not "+
				"local or telegram")
	}

	return checked, nil
}

func checkIssuer(issuer *string) (string, error) {
	checked := value(issuer)

	parsed, err := url.Parse(checked)
	if err != nil || parsed.Scheme != "https" || parsed.Host == "" {
		return "", fieldError("/issuer", "the issuer is not an https address")
	}

	return checked, nil
}

func checkClaim(claim *string) (string, error) {
	if claim == nil {
		return defaultClaim, nil
	}

	checked := strings.TrimSpace(*claim)
	if checked == "" {
		return "", fieldError("/user_id_key", "the claim is empty")
	}

	return checked, nil
}

// clientConfig checks the client and the options that both OIDC presets take, and
// answers a config that holds them.
func clientConfig(input Input, clientID string) (map[string]any, error) {
	if clientID == "" {
		return nil, fieldError("/client_id", "the client identifier is empty")
	}

	if strings.TrimSpace(value(input.ClientSecret)) == "" {
		return nil, fieldError("/client_secret", "the client secret is empty")
	}

	if err := ValidateOptions(input.Options); err != nil {
		return nil, err
	}

	config := make(map[string]any, len(input.Options)+len(FixedKeys()))
	for key, option := range input.Options {
		config[key] = option
	}

	config[keyClientID] = clientID
	config[keyClientSecret] = *input.ClientSecret

	return config, nil
}

// applyChange writes a merge patch into the config of a stored provider, and answers
// the name that the provider then carries.
func applyChange(
	preset Preset,
	config map[string]json.RawMessage,
	change Change,
	name string,
) (string, error) {
	if err := refuseForPreset(preset, change); err != nil {
		return "", err
	}

	if change.Name != nil {
		checked, err := checkName(change.Name)
		if err != nil {
			return "", err
		}

		name = checked
	}

	if err := setText(config, keyClientID, "/client_id", change.ClientID); err != nil {
		return "", err
	}

	if err := setText(config, keyClientSecret, "/client_secret", change.ClientSecret); err != nil {
		return "", err
	}

	if change.Scopes != nil {
		scopes, err := checkScopes(*change.Scopes)
		if err != nil {
			return "", err
		}

		if err = setJSON(config, keyScopes, scopes); err != nil {
			return "", err
		}
	}

	if change.Options != nil {
		return name, mergeOptions(config, change.Options)
	}

	return name, nil
}

func refuseForPreset(preset Preset, change Change) error {
	switch {
	case preset == PresetLocal && change.ClientID != nil:
		return fieldError("/client_id", noInputDetail)
	case preset == PresetLocal && change.ClientSecret != nil:
		return fieldError("/client_secret", noInputDetail)
	case preset == PresetLocal && change.Options != nil:
		return fieldError("/options", noInputDetail)
	case preset != PresetOIDC && change.Scopes != nil:
		return fieldError("/scopes", fixedFieldDetail)
	default:
		return nil
	}
}

// mergeOptions merges the patch into the options of the config. A null member removes
// its option, as merge patch gives.
func mergeOptions(config map[string]json.RawMessage, patch map[string]json.RawMessage) error {
	merged := optionsOf(config)

	for key, option := range patch {
		if string(option) == "null" {
			delete(merged, key)
		} else {
			merged[key] = option
		}
	}

	if err := ValidateOptions(merged); err != nil {
		return err
	}

	maps.DeleteFunc(config, func(key string, _ json.RawMessage) bool {
		return !slices.Contains(FixedKeys(), key)
	})
	maps.Copy(config, merged)

	return nil
}

func setText(
	config map[string]json.RawMessage,
	key string,
	pointer string,
	changed *string,
) error {
	if changed == nil {
		return nil
	}

	trimmed := strings.TrimSpace(*changed)
	if trimmed == "" {
		return fieldError(pointer, "the value is empty")
	}

	return setJSON(config, key, trimmed)
}

func checkName(name *string) (string, error) {
	trimmed := strings.TrimSpace(value(name))
	if trimmed == "" || utf8.RuneCountInString(trimmed) > maxNameRunes {
		return "", fieldError("/name", "the name holds 1 to 64 characters")
	}

	return trimmed, nil
}

func checkScopes(scopes []string) ([]string, error) {
	if len(scopes) == 0 {
		return nil, fieldError("/scopes", "the list holds no scope")
	}

	for i, scope := range scopes {
		if strings.TrimSpace(scope) == "" {
			return nil, fieldError(fmt.Sprintf("/scopes/%d", i), "the scope is empty")
		}
	}

	return scopes, nil
}

// firstPresent answers the first pointer whose member the input carries.
func firstPresent(input Input, pointers ...string) string {
	present := map[string]bool{
		"/id":            input.ID != nil,
		"/name":          input.Name != nil,
		"/issuer":        input.Issuer != nil,
		"/client_id":     input.ClientID != nil,
		"/client_secret": input.ClientSecret != nil,
		"/user_id_key":   input.UserIDKey != nil,
		"/scopes":        input.Scopes != nil,
		"/options":       input.Options != nil,
	}

	for _, pointer := range pointers {
		if present[pointer] {
			return pointer
		}
	}

	return ""
}

func issuerOf(input Input) string {
	if input.Preset == PresetTelegram {
		return telegramIssuer
	}

	return value(input.Issuer)
}

func connectorOf(
	id string,
	kind string,
	name string,
	config map[string]any,
) (dex.Connector, error) {
	encoded, err := json.Marshal(config)
	if err != nil {
		return dex.Connector{}, fmt.Errorf("writing the config of %s: %w", id, err)
	}

	return dex.Connector{ID: id, Type: kind, Name: name, Config: encoded}, nil
}

// setJSON writes the value into the config as JSON.
func setJSON(config map[string]json.RawMessage, key string, value any) error {
	encoded, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("writing %s into the config: %w", key, err)
	}

	config[key] = encoded

	return nil
}

func value(pointer *string) string {
	if pointer == nil {
		return ""
	}

	return *pointer
}
