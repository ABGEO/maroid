package model

import (
	"fmt"
	"math"
	"time"
)

// Direction names whether a money entry adds to the balance or takes from it.
type Direction string

const (
	// DirectionIn marks a top up.
	DirectionIn Direction = "in"
	// DirectionOut marks a charge.
	DirectionOut Direction = "out"
)

// tetriOfLari is the count of tetri in one lari.
const tetriOfLari = 100

// Tetri is an amount of lari as a count of tetri, the hundredth of a lari. A sum of
// tetri holds no error of a binary fraction.
type Tetri int64

// TetriOf answers the amount of lari as tetri, rounded to the nearest tetri.
func TetriOf(lari float64) Tetri {
	return Tetri(math.Round(lari * tetriOfLari))
}

// String answers the amount in lari with two decimals, for example 17.30.
func (t Tetri) String() string {
	sign, count := "", int64(t)
	if count < 0 {
		sign, count = "-", -count
	}

	return fmt.Sprintf("%s%d.%02d", sign, count/tetriOfLari, count%tetriOfLari)
}

// MoneyEntry is one top up or one charge on the balance of a mobile subscriber.
type MoneyEntry struct {
	ID               string    `db:"id"`
	SubscriberNumber string    `db:"subscriber_number"`
	OccurredAt       time.Time `db:"occurred_at"`
	Direction        Direction `db:"direction"`
	Kind             string    `db:"kind"`
	// Amount is never below zero. Direction gives the sign.
	Amount Tetri `db:"amount"`
}

// MonthTotal is the sum of the top ups and of the charges of one month of Tbilisi.
type MonthTotal struct {
	Month   Month
	TopUps  Tetri
	Charges Tetri
}
