package provider

// Preset is a kind of provider that the hub offers.
type Preset string

// The presets of the hub.
const (
	PresetLocal    Preset = "local"
	PresetTelegram Preset = "telegram"
	PresetOIDC     Preset = "oidc"
)

// MarkerKey names the preset inside the config of a connector that the hub stored.
// Dex ignores a key that it does not know, and a connector without it is static.
const MarkerKey = "maroidPreset"

// The keys of the OIDC config of Dex that a field of a preset covers. Every other
// key belongs to the options.
const (
	keyIssuer       = "issuer"
	keyClientID     = "clientID"
	keyClientSecret = "clientSecret"
	keyRedirectURI  = "redirectURI"
	keyUserIDKey    = "userIDKey"
	keyScopes       = "scopes"
)

func (p Preset) known() bool {
	switch p {
	case PresetLocal, PresetTelegram, PresetOIDC:
		return true
	default:
		return false
	}
}
