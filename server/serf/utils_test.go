package serf

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetKappaEventName(t *testing.T) {

	// GetKappaEventName should prepend Kappa event prefix to event type
	assert.Equal(t, "kappa-event:some-event", GetKappaEventName("some-event"))
	assert.Equal(t, "kappa-event:", GetKappaEventName(""))
}

func TestIsKappaEvent(t *testing.T) {

	// IsKappaEvent should return true if the event name starts with the Kappa event prefix.
	assert.True(t, IsKappaEvent("kappa-event:some-event"))

	// IsKappaEvent should return false if the event name does not start with the Kappa event prefix.
	assert.False(t, IsKappaEvent("some-event"))
}

func TestGetRawEventName(t *testing.T) {

	// GetRawEventName strips the Kappa event prefix
	assert.Equal(t, "some-event", GetRawEventName("kappa-event:some-event"))
	assert.Equal(t, "some-event", GetRawEventName("some-event"))
	assert.Equal(t, "", GetRawEventName("kappa-event:"))
}
