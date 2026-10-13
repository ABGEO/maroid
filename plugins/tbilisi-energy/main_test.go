package main

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/abgeo/maroid/libs/pluginapi"
	"github.com/abgeo/maroid/plugins/tbilisi-energy/service"
)

// GASACC-SC-022: The plugin declares the bill as the notification topic monthly_bill.
func TestTheBillIsANotificationTopic(t *testing.T) {
	t.Parallel()

	topics, err := (&TbilisiEnergyPlugin{}).NotificationTopics()

	require.NoError(t, err)
	require.Equal(t, []pluginapi.NotificationTopic{{
		ID:          service.MonthlyBillTopic,
		Name:        "Monthly bill",
		Description: "The gas bill of the last month, with its receipt.",
	}}, topics)
}
