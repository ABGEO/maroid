package figures

import (
	"time"

	"github.com/abgeo/maroid/plugins/drivvo/model"
)

// Entry is one log entry of a vehicle, as the car log copy holds it.
type Entry struct {
	ID           string
	Kind         model.Kind
	OccurredAt   time.Time
	Odometer     float64
	Amount       *model.Amount
	MissedRefuel bool
	Lines        []Line
	Instalments  []model.Instalment
}

// Line is one fuel of a refuel, or one type of a maintenance or an expense. Item is the
// identifier of the row of the fuel or the type, and is empty when the copy holds none.
type Line struct {
	Item     string
	Name     string
	Amount   model.Amount
	Fuel     bool
	Volume   float64
	FullTank bool
	Free     bool
}

// full reports whether one fuel line of the refuel fills the tank.
func (e Entry) full() bool {
	for _, line := range e.Lines {
		if line.Fuel && line.FullTank {
			return true
		}
	}

	return false
}

// volume answers the sum of the volumes of the fuel lines.
func (e Entry) volume() float64 {
	var total float64

	for _, line := range e.Lines {
		if line.Fuel {
			total += line.Volume
		}
	}

	return total
}

// names reports whether a line of the entry names the item.
func (e Entry) names(item string) bool {
	for _, line := range e.Lines {
		if line.Item != "" && line.Item == item {
			return true
		}
	}

	return false
}
