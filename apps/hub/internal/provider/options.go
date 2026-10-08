package provider

import (
	"bytes"
	"encoding/json"
	"errors"
	"maps"
	"reflect"
	"slices"
	"strings"
)

// FieldError names the member of a request that a provider refuses, as a JSON
// Pointer into the body.
type FieldError struct {
	Pointer string
	Detail  string
}

func (e *FieldError) Error() string {
	return "provider: " + e.Pointer + ": " + e.Detail
}

func fieldError(pointer string, detail string) *FieldError {
	return &FieldError{Pointer: pointer, Detail: detail}
}

// oidcOptions mirrors each field of the OIDC config of Dex v2.46.0 that no preset
// fixes. A decode into it refuses a key that Dex lacks and a value of the wrong type.
//
//nolint:tagliatelle // the keys of the config of Dex, which Dex names.
type oidcOptions struct {
	IssuerAlias                string        `json:"issuerAlias"`
	ProviderDiscoveryOverrides *discovery    `json:"providerDiscoveryOverrides"`
	BasicAuthUnsupported       *bool         `json:"basicAuthUnsupported"`
	HostedDomains              []string      `json:"hostedDomains"`
	RootCAs                    []string      `json:"rootCAs"`
	InsecureSkipEmailVerified  bool          `json:"insecureSkipEmailVerified"`
	InsecureEnableGroups       bool          `json:"insecureEnableGroups"`
	AllowedGroups              []string      `json:"allowedGroups"`
	AcrValues                  []string      `json:"acrValues"`
	InsecureSkipVerify         bool          `json:"insecureSkipVerify"`
	GetUserInfo                bool          `json:"getUserInfo"`
	UserNameKey                string        `json:"userNameKey"`
	PromptType                 *string       `json:"promptType"`
	PKCEChallenge              string        `json:"pkceChallenge"`
	OverrideClaimMapping       bool          `json:"overrideClaimMapping"`
	ClaimMapping               *claimMapping `json:"claimMapping"`
	ClaimModifications         *claimChanges `json:"claimModifications"`
}

//nolint:tagliatelle // the keys of the config of Dex, which Dex names.
type discovery struct {
	TokenURL      string `json:"tokenURL"`
	AuthURL       string `json:"authURL"`
	JWKSURL       string `json:"jwksURL"`
	UserInfoURL   string `json:"userInfoURL"`
	DeviceAuthURL string `json:"deviceAuthURL"`
	EndSessionURL string `json:"endSessionURL"`
}

type claimMapping struct {
	PreferredUsername string `json:"preferred_username"`
	Email             string `json:"email"`
	Groups            string `json:"groups"`
}

// claimChanges keeps newGroupFromClaims raw, because Dex answered no item of it and
// the shape of one is unverified.
//
//nolint:tagliatelle // the keys of the config of Dex, which Dex names.
type claimChanges struct {
	NewGroupFromClaims json.RawMessage `json:"newGroupFromClaims"`
	FilterGroupClaims  *struct {
		GroupsFilter string `json:"groupsFilter"`
	} `json:"filterGroupClaims"`
	ModifyGroupNames *struct {
		Prefix string `json:"prefix"`
		Suffix string `json:"suffix"`
	} `json:"modifyGroupNames"`
}

// OptionKeys answers every key that the options take.
func OptionKeys() []string {
	options := reflect.TypeFor[oidcOptions]()
	keys := make([]string, 0, options.NumField())

	for field := range options.Fields() {
		keys = append(keys, strings.Split(field.Tag.Get("json"), ",")[0])
	}

	return keys
}

// FixedKeys answers every key of the config that a field of a preset covers, the
// mark of the preset included.
func FixedKeys() []string {
	return []string{
		MarkerKey, keyIssuer, keyClientID, keyClientSecret, keyRedirectURI, keyUserIDKey, keyScopes,
	}
}

// ValidateOptions refuses a key that a preset fixes, a key that Dex lacks, and a
// value of the wrong type. The error names the key as a pointer into the body.
func ValidateOptions(options map[string]json.RawMessage) error {
	known := OptionKeys()

	for _, key := range slices.Sorted(maps.Keys(options)) {
		switch {
		case slices.Contains(FixedKeys(), key):
			return fieldError("/options/"+key, "a field of the preset sets this key")
		case !slices.Contains(known, key):
			return fieldError("/options/"+key, "Dex v2.46.0 has no option with this key")
		}

		if err := decodeStrict(key, options[key]); err != nil {
			return err
		}
	}

	return nil
}

// decodeStrict decodes one option alone, so a failure names its key.
func decodeStrict(key string, value json.RawMessage) error {
	single, err := json.Marshal(map[string]json.RawMessage{key: value})
	if err != nil {
		return fieldError("/options/"+key, "the value is not JSON")
	}

	decoder := json.NewDecoder(bytes.NewReader(single))
	decoder.DisallowUnknownFields()

	err = decoder.Decode(&oidcOptions{})
	if err == nil {
		return nil
	}

	var typeErr *json.UnmarshalTypeError
	if errors.As(err, &typeErr) && typeErr.Field != "" {
		return fieldError(
			"/options/"+strings.ReplaceAll(typeErr.Field, ".", "/"),
			"the value is a "+typeErr.Value+", and Dex reads a "+typeErr.Type.String(),
		)
	}

	return fieldError("/options/"+key, "Dex does not read this value: "+err.Error())
}
