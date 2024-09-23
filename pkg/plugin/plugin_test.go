package plugin

import (
	"github.com/stretchr/testify/assert"
	"github.com/webhookx-io/webhookx/db/entities"
	"testing"
)

func Test(t *testing.T) {
	instance := &entities.Plugin{
		Name: "notfound",
	}
	err := ExecutePlugin(instance, nil, nil)
	assert.NotNil(t, err)
	assert.Equal(t, "unknown plugin: notfound", err.Error())
}
