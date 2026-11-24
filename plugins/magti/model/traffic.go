package model

import "time"

// TrafficDay is the traffic of one mobile subscriber on one day of Tbilisi. A series that
// the portal answered no row of that day for leaves its fields nil.
type TrafficDay struct {
	ID               string    `db:"id"`
	SubscriberNumber string    `db:"subscriber_number"`
	Day              time.Time `db:"day"`
	DataMB           *float64  `db:"data_mb"`
	CallInSeconds    *int      `db:"call_in_seconds"`
	CallOutSeconds   *int      `db:"call_out_seconds"`
	SMSReceived      *int      `db:"sms_received"`
	SMSSent          *int      `db:"sms_sent"`
}

// Collection is the result of one collection: the count of the traffic days that it
// wrote, and the count of the money entries that it stored for the first time.
type Collection struct {
	Days       int
	NewEntries int
}
