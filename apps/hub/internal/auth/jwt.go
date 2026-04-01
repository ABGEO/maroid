package auth

import (
	"crypto/rsa"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"

	"github.com/abgeo/maroid/apps/hub/internal/config"
)

// ErrJWTUnexpectedClaims indicates that the JWT token claims have an unexpected type.
var ErrJWTUnexpectedClaims = errors.New("jwt: unexpected claims type")

// Claims represents the JWT claims.
//
//nolint:tagliatelle
type Claims struct {
	jwt.RegisteredClaims

	Username string `json:"preferred_username,omitempty"`
	Name     string `json:"name,omitempty"`
	Picture  string `json:"picture,omitempty"`
}

// JWTService handles JWT signing and verification.
type JWTService struct {
	issuer string
	expiry time.Duration

	privateKey *rsa.PrivateKey
	publicKey  *rsa.PublicKey
}

// NewJWTService creates a new JWTService from PEM file paths in config.
func NewJWTService(cfg *config.Config) (*JWTService, error) {
	privateBytes, err := os.ReadFile(cfg.JWT.PrivateKey)
	if err != nil {
		return nil, fmt.Errorf("reading private key: %w", err)
	}

	privateKey, err := jwt.ParseRSAPrivateKeyFromPEM(privateBytes)
	if err != nil {
		return nil, fmt.Errorf("parsing private key: %w", err)
	}

	publicBytes, err := os.ReadFile(cfg.JWT.PublicKey)
	if err != nil {
		return nil, fmt.Errorf("reading public key: %w", err)
	}

	publicKey, err := jwt.ParseRSAPublicKeyFromPEM(publicBytes)
	if err != nil {
		return nil, fmt.Errorf("parsing public key: %w", err)
	}

	return &JWTService{
		issuer: cfg.JWT.Issuer,
		expiry: cfg.JWT.TokenExpiry,

		privateKey: privateKey,
		publicKey:  publicKey,
	}, nil
}

// Sign creates a signed JWT from the given claims.
// It sets Issuer, IssuedAt, and ExpiresAt internally.
func (s *JWTService) Sign(claims Claims) (string, error) {
	now := time.Now()

	claims.Issuer = s.issuer
	claims.IssuedAt = jwt.NewNumericDate(now)
	claims.ExpiresAt = jwt.NewNumericDate(now.Add(s.expiry))

	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)

	signed, err := token.SignedString(s.privateKey)
	if err != nil {
		return "", fmt.Errorf("signing token: %w", err)
	}

	return signed, nil
}

// Verify parses and validates a JWT, returning the claims.
func (s *JWTService) Verify(tokenString string) (*Claims, error) {
	keyFunc := func(_ *jwt.Token) (any, error) {
		return s.publicKey, nil
	}

	token, err := jwt.ParseWithClaims(
		tokenString,
		&Claims{},
		keyFunc,
		jwt.WithValidMethods([]string{jwt.SigningMethodRS256.Name}),
		jwt.WithIssuedAt(),
		jwt.WithExpirationRequired(),
		jwt.WithIssuer(s.issuer),
	)
	if err != nil {
		return nil, fmt.Errorf("parsing token: %w", err)
	}

	claims, ok := token.Claims.(*Claims)
	if !ok {
		return nil, ErrJWTUnexpectedClaims
	}

	return claims, nil
}
