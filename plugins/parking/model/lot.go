package model

import "github.com/paulmach/orb"

// ParkingLot is a place for parking that the parking service names with a number and
// an address. The points of the outline hold the longitude, then the latitude.
type ParkingLot struct {
	Number  string
	Address string
	Outline orb.Polygon
}

// NearbyLot is a parking lot and its distance in meters from the shared location.
type NearbyLot struct {
	ParkingLot

	Distance float64
}
