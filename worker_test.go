package pigeon_test

import (
	"github.com/stretchr/testify/assert"
	"github.com/wblakecaldwell/pigeon/worker"
	"testing"
)

// TestRegisterAndFetchAction tests different scenarios of registering and fetching actions
func TestRegisterAndFetchAction(t *testing.T) {
	w, err := worker.NewWorker()
	assert.NoError(t, err)
	assert.NotNil(t, w)

	runner := func(req string) (string, error) {
		return "", nil
	}

	// create an action
	action := worker.NewAction("fOO", runner)
	assert.NotNil(t, action)

	// register it
	err = w.RegisterAction(action)
	assert.NoError(t, err)

	// assert it's stored
	foundAction := w.GetAction("fOO")
	assert.Equal(t, foundAction, action)

	// assert name keeps their casing
	assert.Equal(t, "fOO", foundAction.Name())

	// assert it can be found with different casing
	assert.Equal(t, w.GetAction("Foo"), action)

	// assert can't register again
	err = w.RegisterAction(action)
	assert.Error(t, err)
	assert.Equal(t, "There is already an action named foo", err.Error())

	// assert can't be registered again with different casing
	err = w.RegisterAction(worker.NewAction("FOO", runner))
	assert.Error(t, err)
	assert.Equal(t, "There is already an action named foo", err.Error())

	// assert unregistered actions aren't found
	action = w.GetAction("not")
	assert.Nil(t, action)
}
