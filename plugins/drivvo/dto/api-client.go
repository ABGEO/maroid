package dto

import (
	"encoding/json"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/abgeo/maroid/plugins/drivvo/errs"
	"github.com/abgeo/maroid/plugins/drivvo/model"
)

// The forms of a day and time in an answer of the car log service. The first is the form
// that the service writes. The others are read too, so a change of form does not stop a
// collection.
//
//nolint:gochecknoglobals // a fixed list, read only.
var momentLayouts = []string{"2006-01-02 15:04:05", "2006-01-02T15:04:05", "2006-01-02"}

const (
	dayLayout = "2006-01-02"

	// CodeCredentialsRejected is the code of the exception that answers a sign in with a
	// wrong email address or password.
	CodeCredentialsRejected = 1908

	// LanguageEnglish is the language of the messages of the service.
	LanguageEnglish = "en"

	// fallbackTokenLife is the life of a token whose answer names no end.
	fallbackTokenLife = 24 * time.Hour
)

// The codes of a unit in the car log settings and in a vehicle.
const (
	unitKilometre      = 1
	unitMile           = 2
	unitLitre          = 1
	unitGallonUS       = 2
	unitGallonImperial = 3
)

// The codes of the unit of the interval of a reminder. Any other code counts months.
const (
	periodDay  = 1
	periodYear = 3
)

// LogInRequest is the body of the sign in. Senha is the lowercase hex MD5 of the password.
type LogInRequest struct {
	Email  string `json:"email"`
	Senha  string `json:"senha"`
	Idioma string `json:"idioma"`
}

// LogInAnswer is the answer of the sign in. It names four members alone, so the name, the
// email address, the phone number, and the driver licence of the holder never reach a
// value of Go.
type LogInAnswer struct {
	Token            string `json:"token"`
	TokenValidUntil  string `json:"token_validade"` //nolint:misspell // a member of the car log service.
	ExpiraEmSegundos int64  `json:"expira_em_segundos"`
	AlterarSenha     bool   `json:"alterar_senha"`
}

// ServiceError is the body of an answer that reports a failure.
type ServiceError struct {
	Exception *struct {
		Code int `json:"code"`
	} `json:"exception"`
}

// Code answers the code of the exception, or zero when the body holds none.
func (e ServiceError) Code() int {
	if e.Exception == nil {
		return 0
	}

	return e.Exception.Code
}

// Expiry answers the moment the token ends: now plus ExpiraEmSegundos, else TokenValidUntil
// read in UTC, else one day after now.
func (a LogInAnswer) Expiry(now time.Time) time.Time {
	if a.ExpiraEmSegundos > 0 {
		return now.Add(time.Duration(a.ExpiraEmSegundos) * time.Second)
	}

	if end, err := parseMoment(a.TokenValidUntil, time.UTC); err == nil {
		return end
	}

	return now.Add(fallbackTokenLife)
}

// Vehicle is one vehicle of the answer of GET /veiculo/web.
type Vehicle struct {
	ID               int32  `json:"id_veiculo"`
	Nome             string `json:"nome"`
	Marca            string `json:"marca"`
	Modelo           string `json:"modelo"`
	Ano              int32  `json:"ano"`
	UnidadeDistancia int    `json:"unidade_distancia"`
	Principal        bool   `json:"principal"`
	Ativo            bool   `json:"ativo"`
}

// Read answers the vehicle. A unit outside the list of the service is an error.
func (v Vehicle) Read() (model.Vehicle, error) {
	vehicle := model.Vehicle{
		SourceID:  v.ID,
		Name:      v.Nome,
		Make:      v.Marca,
		Model:     v.Modelo,
		Principal: v.Principal,
		Active:    v.Ativo,
	}

	if v.Ano > 0 {
		year := v.Ano
		vehicle.Year = &year
	}

	switch v.UnidadeDistancia {
	case unitKilometre:
		vehicle.DistanceUnit = model.DistanceKilometre
	case unitMile:
		vehicle.DistanceUnit = model.DistanceMile
	default:
		return model.Vehicle{}, fmt.Errorf(
			"%w: vehicle %d: unidade_distancia %d",
			errs.ErrRequestFailed,
			v.ID,
			v.UnidadeDistancia,
		)
	}

	return vehicle, nil
}

// Settings is one item of the answer of GET /configuracao.
type Settings struct {
	UnidadeMedida         int    `json:"unidade_medida"`
	EficienciaCombustivel int    `json:"eficiencia_combustivel"`
	FormatoValor          string `json:"formato_valor"`
}

// Read answers the car log settings. A unit or a format outside the lists of the service
// is an error. An empty currency is USD, as the service shows it.
func (s Settings) Read() (model.CarLogSettings, error) {
	settings := model.CarLogSettings{
		ConsumptionFormat: model.ConsumptionFormat(s.EficienciaCombustivel),
		Currency:          strings.ToUpper(strings.TrimSpace(s.FormatoValor)),
	}

	switch s.UnidadeMedida {
	case unitLitre:
		settings.VolumeUnit = model.VolumeLitre
	case unitGallonUS:
		settings.VolumeUnit = model.VolumeGallonUS
	case unitGallonImperial:
		settings.VolumeUnit = model.VolumeGallonImperial
	default:
		return model.CarLogSettings{}, fmt.Errorf(
			"%w: unidade_medida %d",
			errs.ErrRequestFailed,
			s.UnidadeMedida,
		)
	}

	if !settings.ConsumptionFormat.Valid() {
		return model.CarLogSettings{}, fmt.Errorf(
			"%w: eficiencia_combustivel %d",
			errs.ErrRequestFailed,
			s.EficienciaCombustivel,
		)
	}

	if settings.Currency == "" {
		settings.Currency = "USD"
	}

	return settings, nil
}

// ReferenceList names one list of reference items: its path and the member that holds
// the identifier of an item.
type ReferenceList struct {
	Kind     model.ReferenceKind
	Path     string
	IDMember string
}

// ReferenceLists names every list of reference items that a collection reads.
func ReferenceLists() []ReferenceList {
	return []ReferenceList{
		{Kind: model.ReferenceFuel, Path: "/combustivel", IDMember: "id_combustivel"},
		{Kind: model.ReferenceMaintenanceType, Path: "/tipoServico", IDMember: "id_tipo_servico"},
		{Kind: model.ReferenceExpenseType, Path: "/tipoDespesa", IDMember: "id_tipo_despesa"},
		{Kind: model.ReferenceIncomeType, Path: "/tipoReceita", IDMember: "id_tipo_receita"},
		{Kind: model.ReferenceReason, Path: "/tipoMotivo", IDMember: "id_tipo_motivo"},
		{
			Kind:     model.ReferencePaymentMethod,
			Path:     "/formaPagamento",
			IDMember: "id_forma_pagamento",
		},
		{Kind: model.ReferenceStation, Path: "/postoCombustivel", IDMember: "id_posto_combustivel"},
		{Kind: model.ReferencePlace, Path: "/local", IDMember: "id_local"},
	}
}

// ReadReferenceItems reads the answer of one list of reference items. An item with no
// identifier is skipped.
func ReadReferenceItems(body []byte, list ReferenceList) ([]model.ReferenceItem, error) {
	var raw []map[string]json.RawMessage
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, fmt.Errorf("%w: reading %s: %w", errs.ErrRequestFailed, list.Path, err)
	}

	items := make([]model.ReferenceItem, 0, len(raw))

	for _, member := range raw {
		var (
			id   int32
			name string
		)

		if err := decodeMember(member, list.IDMember, &id); err != nil {
			return nil, fmt.Errorf("%w: reading %s: %w", errs.ErrRequestFailed, list.Path, err)
		}

		if err := decodeMember(member, "nome", &name); err != nil {
			return nil, fmt.Errorf("%w: reading %s: %w", errs.ErrRequestFailed, list.Path, err)
		}

		if id > 0 {
			items = append(items, model.ReferenceItem{Kind: list.Kind, SourceID: id, Name: name})
		}
	}

	return items, nil
}

func decodeMember(member map[string]json.RawMessage, name string, target any) error {
	value, found := member[name]
	if !found || string(value) == "null" {
		return nil
	}

	if err := json.Unmarshal(value, target); err != nil {
		return fmt.Errorf("member %s: %w", name, err)
	}

	return nil
}

// Instalment is one item of parcelas.
type Instalment struct {
	Numero         int16   `json:"numero"`
	DataVencimento string  `json:"data_vencimento"`
	Valor          float64 `json:"valor"`
}

// Refuel is one item of the answer of GET /veiculo/{id}/abastecimento/web.
type Refuel struct {
	ID                int32    `json:"id_abastecimento"`
	VehicleID         int32    `json:"id_veiculo"`
	Data              string   `json:"data"`
	Odometro          float64  `json:"odometro"`
	IDCombustivel     int32    `json:"id_combustivel"`
	Preco             float64  `json:"preco"`
	Volume            float64  `json:"volume"`
	ValorTotal        float64  `json:"valor_total"`
	TanqueCheio       bool     `json:"tanque_cheio"`
	SemCusto          bool     `json:"sem_custo"`
	IDCombustivelDois int32    `json:"id_combustivel_dois"`
	PrecoDois         float64  `json:"preco_dois"`
	VolumeDois        float64  `json:"volume_dois"`
	ValorTotalDois    float64  `json:"valor_total_dois"`
	TanqueCheioDois   bool     `json:"tanque_cheio_dois"`
	SemCustoDois      bool     `json:"sem_custo_dois"`
	IDCombustivelTres int32    `json:"id_combustivel_tres"`
	PrecoTres         float64  `json:"preco_tres"`
	VolumeTres        float64  `json:"volume_tres"`
	ValorTotalTres    float64  `json:"valor_total_tres"`
	TanqueCheioTres   bool     `json:"tanque_cheio_tres"`
	SemCustoTres      bool     `json:"sem_custo_tres"`
	EsqueceuAnterior  bool     `json:"esqueceu_anterior"`
	IDPosto           int32    `json:"id_posto_combustivel"`
	IDFormaPagamento  int32    `json:"id_forma_pagamento"`
	IDTipoMotivo      int32    `json:"id_tipo_motivo"`
	BateriaInicialPct *float64 `json:"bateria_inicial_pct"`
	BateriaFinalPct   *float64 `json:"bateria_final_pct"`
	DuracaoMinutos    *int32   `json:"duracao_minutos"`
	Observacao        string   `json:"observacao"`
}

// Read answers the refuel with a line for each fuel whose identifier is above zero. The
// amount sums the lines, and a free line counts zero.
func (r Refuel) Read() (model.LogEntry, error) {
	moment, err := parseMoment(r.Data, model.Tbilisi())
	if err != nil {
		return model.LogEntry{}, entryFailure(model.KindRefuel, r.ID, err)
	}

	entry := model.LogEntry{
		Kind:            model.KindRefuel,
		SourceID:        r.ID,
		VehicleSource:   r.VehicleID,
		OccurredAt:      moment,
		Odometer:        r.Odometro,
		Note:            r.Observacao,
		Station:         r.IDPosto,
		PaymentMethod:   r.IDFormaPagamento,
		Reason:          r.IDTipoMotivo,
		MissedRefuel:    r.EsqueceuAnterior,
		BatteryStartPct: positive(r.BateriaInicialPct),
		BatteryEndPct:   positive(r.BateriaFinalPct),
		ChargeMinutes:   positive(r.DuracaoMinutos),
	}

	entry.Lines = r.fuelLines()
	entry.Amount = sumLines(entry.Lines)

	return entry, nil
}

// litres answers the volume of a fuel as the web site of the car log service reads it. The
// service sends the volume of a free fuel alone, so a paid fuel takes its amount over its price.
func litres(free bool, volume, total, price float64) float64 {
	const decimals = 1000

	switch {
	case free:
		return volume
	case price > 0:
		return math.Round(total/price*decimals) / decimals
	default:
		return 0
	}
}

// fuelLines answers a line for each of the three fuels whose identifier is above zero.
func (r Refuel) fuelLines() []model.Line {
	fuels := []struct {
		item                 int32
		price, volume, total float64
		fullTank, free       bool
	}{
		{r.IDCombustivel, r.Preco, r.Volume, r.ValorTotal, r.TanqueCheio, r.SemCusto},
		{
			r.IDCombustivelDois,
			r.PrecoDois,
			r.VolumeDois,
			r.ValorTotalDois,
			r.TanqueCheioDois,
			r.SemCustoDois,
		},
		{
			r.IDCombustivelTres,
			r.PrecoTres,
			r.VolumeTres,
			r.ValorTotalTres,
			r.TanqueCheioTres,
			r.SemCustoTres,
		},
	}

	var lines []model.Line

	for _, fuel := range fuels {
		if fuel.item <= 0 {
			continue
		}

		lines = append(lines, model.Line{
			Position:  len(lines) + 1,
			Item:      fuel.item,
			Amount:    model.AmountOf(fuel.total),
			Fuel:      true,
			UnitPrice: model.AmountOf(fuel.price),
			Volume:    litres(fuel.free, fuel.volume, fuel.total, fuel.price),
			FullTank:  fuel.fullTank,
			Free:      fuel.free,
		})
	}

	return lines
}

// TypeLine is one item of tipos_servico or tipos_despesa.
type TypeLine struct {
	IDTipoServico int32   `json:"id_tipo_servico"`
	IDTipoDespesa int32   `json:"id_tipo_despesa"`
	Valor         float64 `json:"valor"`
}

// Maintenance is one item of the answer of GET /veiculo/{id}/servico/completo.
type Maintenance struct {
	ID               int32        `json:"id_servico"`
	VehicleID        int32        `json:"id_veiculo"`
	Data             string       `json:"data"`
	Odometro         float64      `json:"odometro"`
	Titulo           string       `json:"titulo"`
	IDLocal          int32        `json:"id_local"`
	IDFormaPagamento int32        `json:"id_forma_pagamento"`
	TiposServico     []TypeLine   `json:"tipos_servico"`
	Parcelas         []Instalment `json:"parcelas"`
	Observacao       string       `json:"observacao"`
}

// Read answers the maintenance with a line for each of its types, in the order of the
// service.
func (m Maintenance) Read() (model.LogEntry, error) {
	moment, err := parseMoment(m.Data, model.Tbilisi())
	if err != nil {
		return model.LogEntry{}, entryFailure(model.KindMaintenance, m.ID, err)
	}

	instalments, err := readInstalments(m.Parcelas)
	if err != nil {
		return model.LogEntry{}, entryFailure(model.KindMaintenance, m.ID, err)
	}

	entry := model.LogEntry{
		Kind:          model.KindMaintenance,
		SourceID:      m.ID,
		VehicleSource: m.VehicleID,
		OccurredAt:    moment,
		Odometer:      m.Odometro,
		Title:         m.Titulo,
		Note:          m.Observacao,
		Place:         m.IDLocal,
		PaymentMethod: m.IDFormaPagamento,
		Lines:         typeLines(m.TiposServico, func(l TypeLine) int32 { return l.IDTipoServico }),
		Instalments:   instalments,
	}
	entry.Amount = sumLines(entry.Lines)

	return entry, nil
}

// Expense is one item of the answer of GET /veiculo/{id}/despesa/completo.
type Expense struct {
	ID               int32        `json:"id_despesa"`
	VehicleID        int32        `json:"id_veiculo"`
	Data             string       `json:"data"`
	Odometro         float64      `json:"odometro"`
	Titulo           string       `json:"titulo"`
	IDLocal          int32        `json:"id_local"`
	IDFormaPagamento int32        `json:"id_forma_pagamento"`
	IDTipoMotivo     int32        `json:"id_tipo_motivo"`
	TiposDespesa     []TypeLine   `json:"tipos_despesa"`
	Parcelas         []Instalment `json:"parcelas"`
	Observacao       string       `json:"observacao"`
}

// Read answers the expense with a line for each of its types, in the order of the service.
func (e Expense) Read() (model.LogEntry, error) {
	moment, err := parseMoment(e.Data, model.Tbilisi())
	if err != nil {
		return model.LogEntry{}, entryFailure(model.KindExpense, e.ID, err)
	}

	instalments, err := readInstalments(e.Parcelas)
	if err != nil {
		return model.LogEntry{}, entryFailure(model.KindExpense, e.ID, err)
	}

	entry := model.LogEntry{
		Kind:          model.KindExpense,
		SourceID:      e.ID,
		VehicleSource: e.VehicleID,
		OccurredAt:    moment,
		Odometer:      e.Odometro,
		Title:         e.Titulo,
		Note:          e.Observacao,
		Place:         e.IDLocal,
		PaymentMethod: e.IDFormaPagamento,
		Reason:        e.IDTipoMotivo,
		Lines:         typeLines(e.TiposDespesa, func(l TypeLine) int32 { return l.IDTipoDespesa }),
		Instalments:   instalments,
	}
	entry.Amount = sumLines(entry.Lines)

	return entry, nil
}

// Income is one item of the answer of GET /veiculo/{id}/receita/completo.
type Income struct {
	ID            int32        `json:"id_receita"`
	VehicleID     int32        `json:"id_veiculo"`
	Data          string       `json:"data"`
	Odometro      float64      `json:"odometro"`
	IDTipoReceita int32        `json:"id_tipo_receita"`
	Valor         float64      `json:"valor"`
	Parcelas      []Instalment `json:"parcelas"`
	Observacao    string       `json:"observacao"`
}

// Read answers the income. Its amount is valor.
func (i Income) Read() (model.LogEntry, error) {
	moment, err := parseMoment(i.Data, model.Tbilisi())
	if err != nil {
		return model.LogEntry{}, entryFailure(model.KindIncome, i.ID, err)
	}

	instalments, err := readInstalments(i.Parcelas)
	if err != nil {
		return model.LogEntry{}, entryFailure(model.KindIncome, i.ID, err)
	}

	amount := model.AmountOf(i.Valor)

	return model.LogEntry{
		Kind:          model.KindIncome,
		SourceID:      i.ID,
		VehicleSource: i.VehicleID,
		OccurredAt:    moment,
		Odometer:      i.Odometro,
		Amount:        &amount,
		Note:          i.Observacao,
		IncomeType:    i.IDTipoReceita,
		Instalments:   instalments,
	}, nil
}

// OdometerEntry is one item of the answer of GET /veiculo/{id}/leitura/web.
type OdometerEntry struct {
	ID         int32   `json:"id_leitura"`
	VehicleID  int32   `json:"id_veiculo"`
	Data       string  `json:"data"`
	Odometro   float64 `json:"odometro"`
	Observacao string  `json:"observacao"`
}

// Read answers the odometer entry. It holds no amount.
func (o OdometerEntry) Read() (model.LogEntry, error) {
	moment, err := parseMoment(o.Data, model.Tbilisi())
	if err != nil {
		return model.LogEntry{}, entryFailure(model.KindOdometer, o.ID, err)
	}

	return model.LogEntry{
		Kind:          model.KindOdometer,
		SourceID:      o.ID,
		VehicleSource: o.VehicleID,
		OccurredAt:    moment,
		Odometer:      o.Odometro,
		Note:          o.Observacao,
	}, nil
}

// Reminder is one item of the answer of GET /veiculo/{id}/lembrete/web. Tipo true names an
// expense type. UnicoRepetir true names a reminder that applies one time.
type Reminder struct {
	ID               int32   `json:"id_lembrete"`
	VehicleID        int32   `json:"id_veiculo"`
	Tipo             bool    `json:"tipo"`
	IDTipoServico    int32   `json:"id_tipo_servico"`
	IDTipoDespesa    int32   `json:"id_tipo_despesa"`
	Data             string  `json:"data"`
	Odometro         float64 `json:"odometro"`
	UnicoRepetir     bool    `json:"unico_repetir"`
	RepetirDistancia float64 `json:"repetir_distancia"`
	RepetirTempo     int32   `json:"repetir_tempo"`
	Periodo          int     `json:"periodo"`
	Observacao       string  `json:"observacao"`
}

// Read answers the reminder. A value of zero is no value, as the car log service reads it.
func (r Reminder) Read() (model.Reminder, error) {
	reminder := model.Reminder{
		SourceID:      r.ID,
		VehicleSource: r.VehicleID,
		Kind:          model.KindMaintenance,
		Item:          r.IDTipoServico,
		Repeats:       !r.UnicoRepetir,
		Note:          r.Observacao,
	}

	if r.Tipo {
		reminder.Kind, reminder.Item = model.KindExpense, r.IDTipoDespesa
	}

	if strings.TrimSpace(r.Data) != "" {
		day, err := parseDay(r.Data)
		if err != nil {
			return model.Reminder{}, fmt.Errorf("reminder %d: %w", r.ID, err)
		}

		reminder.DueOn = &day
	}

	reminder.DueOdometer = positive(&r.Odometro)

	if reminder.Repeats {
		reminder.RepeatDistance = positive(&r.RepetirDistancia)

		if r.RepetirTempo > 0 {
			every, unit := r.RepetirTempo, repeatUnit(r.Periodo)
			reminder.RepeatEvery, reminder.RepeatUnit = &every, &unit
		}
	}

	return reminder, nil
}

func repeatUnit(period int) model.RepeatUnit {
	switch period {
	case periodDay:
		return model.RepeatDay
	case periodYear:
		return model.RepeatYear
	}

	return model.RepeatMonth
}

func typeLines(lines []TypeLine, item func(TypeLine) int32) []model.Line {
	read := make([]model.Line, 0, len(lines))

	for _, line := range lines {
		read = append(read, model.Line{
			Position: len(read) + 1,
			Item:     item(line),
			Amount:   model.AmountOf(line.Valor),
		})
	}

	return read
}

func sumLines(lines []model.Line) *model.Amount {
	var sum model.Amount

	for _, line := range lines {
		if !line.Free {
			sum += line.Amount
		}
	}

	return &sum
}

func readInstalments(parcelas []Instalment) ([]model.Instalment, error) {
	instalments := make([]model.Instalment, 0, len(parcelas))

	for _, parcela := range parcelas {
		if parcela.Numero <= 0 {
			continue
		}

		day, err := parseDay(parcela.DataVencimento)
		if err != nil {
			return nil, fmt.Errorf("instalment %d: %w", parcela.Numero, err)
		}

		instalments = append(instalments, model.Instalment{
			Number: parcela.Numero,
			DueOn:  day,
			Amount: model.AmountOf(parcela.Valor),
		})
	}

	return instalments, nil
}

// parseMoment reads a day and time of the service in the zone, and answers it in UTC.
func parseMoment(value string, zone *time.Location) (time.Time, error) {
	text := strings.TrimSpace(value)

	for _, layout := range momentLayouts {
		if moment, err := time.ParseInLocation(layout, text, zone); err == nil {
			return moment.UTC(), nil
		}
	}

	return time.Time{}, fmt.Errorf(
		"%w: the moment %q has no known form",
		errs.ErrRequestFailed,
		value,
	)
}

// parseDay reads the date part of a day and time of the service, as a day at midnight UTC.
// A day names no moment, so no zone moves it.
func parseDay(value string) (time.Time, error) {
	text := strings.TrimSpace(value)
	if len(text) < len(dayLayout) {
		return time.Time{}, fmt.Errorf(
			"%w: the day %q has no known form",
			errs.ErrRequestFailed,
			value,
		)
	}

	day, err := time.Parse(dayLayout, text[:len(dayLayout)])
	if err != nil {
		return time.Time{}, fmt.Errorf(
			"%w: the day %q has no known form: %w",
			errs.ErrRequestFailed,
			value,
			err,
		)
	}

	return day, nil
}

func positive[T int32 | float64](value *T) *T {
	if value == nil || *value <= 0 {
		return nil
	}

	kept := *value

	return &kept
}

func entryFailure(kind model.Kind, id int32, err error) error {
	return fmt.Errorf("%s %d: %w", kind, id, err)
}
