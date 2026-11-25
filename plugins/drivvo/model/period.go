package model

import "time"

// tbilisiOffset is the offset of Tbilisi from UTC. Tbilisi keeps no summer time.
const tbilisiOffset = 4 * 60 * 60

//nolint:gochecknoglobals // one zone value, read only, shared by every reader.
var tbilisi = time.FixedZone("Asia/Tbilisi", tbilisiOffset)

// Tbilisi answers the zone in which the car log service reports each day and time. A fixed
// zone needs no zone database in the image of the hub.
func Tbilisi() *time.Location {
	return tbilisi
}
