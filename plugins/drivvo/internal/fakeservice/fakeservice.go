// Package fakeservice plays the car log service for the tests of the plugin. No production
// code imports it.
//
//nolint:mnd,errchkjson // a fixture of the tests, never served.
package fakeservice

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
)

// The values that the fake service answers unless a test sets others.
const (
	Token         = "T"
	TokenSeconds  = 86400
	Hatchback     = 11
	OldSedan      = 12
	PathLogIn     = "/autenticacao/login"
	PathVehicles  = "/veiculo/web"
	PathSettings  = "/configuracao"
	HolderName    = "Holder Name"
	HolderEmail   = "holder@example.com"
	HolderPhone   = "+995 555 000 000"
	HolderLicence = "DL-0001"

	memberName = "nome"
)

// Call is one request that the fake service received.
type Call struct {
	Method      string
	Path        string
	Token       string
	Platform    string
	Version     string
	VersionName string
	Body        []byte
}

// Service is the state of the fake service. A test changes a field through Set.
type Service struct {
	mu sync.Mutex

	// SignInStatus and SignInBody answer the sign in.
	SignInStatus int
	SignInBody   any
	// Valid holds the tokens that a data call accepts. A data call with another token
	// answers 401.
	Valid map[string]bool
	// Answers holds the JSON answer of each data path.
	Answers map[string]any
	// Status answers a path with a status in place of its answer.
	Status map[string]int

	calls  []Call
	server *httptest.Server
}

// New starts a fake service whose account holds "Hatchback" (11, principal, active, km)
// and "Old sedan" (12, inactive, km), with litres, the format 4, and GEL, and no log entry.
func New(t *testing.T) *Service {
	t.Helper()

	service := &Service{
		SignInStatus: http.StatusOK,
		SignInBody: map[string]any{
			"token":              Token,
			"token_validade":     "2026-11-26 10:00:00", //nolint:misspell // a member of the service.
			"expira_em_segundos": TokenSeconds,
			"alterar_senha":      false,
			memberName:           HolderName,
			"email":              HolderEmail,
			"telefone":           HolderPhone,
			"cnh":                HolderLicence,
		},
		Valid: map[string]bool{Token: true},
		Answers: map[string]any{
			PathVehicles: []any{
				Vehicle(Hatchback, "Hatchback", true, true),
				Vehicle(OldSedan, "Old sedan", false, false),
			},
			PathSettings: []any{map[string]any{
				"unidade_medida":         1,
				"eficiencia_combustivel": 4,
				"formato_valor":          "GEL",
				"idioma":                 "en",
			}},
			"/combustivel": []any{
				Named("id_combustivel", 1, "Petrol"),
				Named("id_combustivel", 2, "LPG"),
			},
			"/tipoServico":      []any{Named("id_tipo_servico", 5, "Oil change")},
			"/tipoDespesa":      []any{Named("id_tipo_despesa", 7, "Insurance")},
			"/tipoReceita":      []any{Named("id_tipo_receita", 9, "Ride")},
			"/tipoMotivo":       []any{},
			"/formaPagamento":   []any{Named("id_forma_pagamento", 3, "Card")},
			"/postoCombustivel": []any{Named("id_posto_combustivel", 4, "Gulf")},
			"/local":            []any{Named("id_local", 6, "Garage")},
		},
		Status: map[string]int{},
	}

	for _, vehicle := range []int{Hatchback, OldSedan} {
		for _, path := range VehiclePaths(vehicle) {
			service.Answers[path] = []any{}
		}
	}

	service.server = httptest.NewServer(service)
	t.Cleanup(service.server.Close)

	return service
}

// VehiclePaths answers the six paths that a collection reads for one vehicle.
func VehiclePaths(vehicle int) []string {
	return []string{
		fmt.Sprintf("/veiculo/%d/abastecimento/web", vehicle),
		fmt.Sprintf("/veiculo/%d/servico/completo", vehicle),
		fmt.Sprintf("/veiculo/%d/despesa/completo", vehicle),
		fmt.Sprintf("/veiculo/%d/receita/completo", vehicle),
		fmt.Sprintf("/veiculo/%d/leitura/web", vehicle),
		fmt.Sprintf("/veiculo/%d/lembrete/web", vehicle),
	}
}

// Vehicle answers one vehicle of the answer of /veiculo/web, in kilometres.
func Vehicle(id int, name string, principal, active bool) map[string]any {
	return map[string]any{
		"id_veiculo":        id,
		memberName:          name,
		"marca":             "Make " + name,
		"modelo":            "Model " + name,
		"ano":               2019,
		"placa":             "AA-000-AA",
		"unidade_distancia": 1,
		"principal":         principal,
		"ativo":             active,
	}
}

// Named answers one reference item.
func Named(idMember string, id int, name string) map[string]any {
	return map[string]any{idMember: id, memberName: name, "ativo": true}
}

// URL answers the origin of the fake service.
func (s *Service) URL() string {
	return s.server.URL
}

// Set changes the state of the fake service under its lock.
func (s *Service) Set(change func(*Service)) {
	s.mu.Lock()
	defer s.mu.Unlock()

	change(s)
}

// Calls answers every request, in order.
func (s *Service) Calls() []Call {
	s.mu.Lock()
	defer s.mu.Unlock()

	return append([]Call(nil), s.calls...)
}

// CallsTo answers the requests to one path, in order.
func (s *Service) CallsTo(path string) []Call {
	var matched []Call

	for _, call := range s.Calls() {
		if call.Path == path {
			matched = append(matched, call)
		}
	}

	return matched
}

// ServeHTTP answers one request.
func (s *Service) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	body, _ := io.ReadAll(r.Body)

	s.mu.Lock()
	defer s.mu.Unlock()

	s.calls = append(s.calls, Call{
		Method:      r.Method,
		Path:        r.URL.Path,
		Token:       r.Header.Get("X-Token"),
		Platform:    r.Header.Get("App-Platform"),
		Version:     r.Header.Get("App-Version"),
		VersionName: r.Header.Get("App-Version-Name"),
		Body:        body,
	})

	w.Header().Set("Content-Type", "application/json")

	if r.Method == http.MethodPost && r.URL.Path == PathLogIn {
		w.WriteHeader(s.SignInStatus)
		_ = json.NewEncoder(w).Encode(s.SignInBody)

		return
	}

	if status, found := s.Status[r.URL.Path]; found {
		w.WriteHeader(status)
		_, _ = io.WriteString(w, `{"result":false,"exception":{"code":500,"message":"failure"}}`)

		return
	}

	if !s.Valid[r.Header.Get("X-Token")] {
		w.WriteHeader(http.StatusUnauthorized)

		return
	}

	answer, found := s.Answers[r.URL.Path]
	if r.Method != http.MethodGet || !found {
		w.WriteHeader(http.StatusNotFound)

		return
	}

	_ = json.NewEncoder(w).Encode(answer)
}
