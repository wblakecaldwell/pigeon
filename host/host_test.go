package host_test

import (
	"github.com/stretchr/testify/assert"
	"github.com/wblakecaldwell/pigeon/host"
	"testing"
)

// TestRegisterAndFetchAction tests different scenarios of registering and fetching actions
func TestRegisterAndFetchAction(t *testing.T) {
	h, err := host.NewHost()
	assert.NoError(t, err)
	assert.NotNil(t, h)

	runner := func() ([]byte, error) {
		return nil, nil
	}

	// create an action
	action := host.NewAction("fOO", runner)
	assert.NotNil(t, action)

	// register it
	err = h.RegisterAction(action)
	assert.NoError(t, err)

	// assert it's stored
	foundAction := h.GetAction("fOO")
	assert.Equal(t, foundAction, action)

	// assert name keeps their casing
	assert.Equal(t, "fOO", foundAction.Name())

	// assert it can be found with different casing
	assert.Equal(t, h.GetAction("Foo"), action)

	// assert can't register again
	err = h.RegisterAction(action)
	assert.Error(t, err)
	assert.Equal(t, "There is already an action named foo", err.Error())

	// assert can't be registered again with different casing
	err = h.RegisterAction(host.NewAction("FOO", runner))
	assert.Error(t, err)
	assert.Equal(t, "There is already an action named foo", err.Error())

	// assert unregistered actions aren't found
	action = h.GetAction("not")
	assert.Nil(t, action)
}
