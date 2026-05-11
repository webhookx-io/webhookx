package uid

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test(t *testing.T) {
	t.Run("attempt prefix", func(t *testing.T) {
		id := Generate(AttemptPrefix)
		assert.True(t, strings.HasPrefix(id, "at_"))
	})
	t.Run("endpoint prefix", func(t *testing.T) {
		id := Generate(EndpointPrefix)
		assert.True(t, strings.HasPrefix(id, "end_"))
	})
	t.Run("event prefix", func(t *testing.T) {
		id := Generate(EventPrefix)
		assert.True(t, strings.HasPrefix(id, "evt_"))
	})
	t.Run("plugin prefix", func(t *testing.T) {
		id := Generate(PluginPrefix)
		assert.True(t, strings.HasPrefix(id, "plg_"))
	})
	t.Run("source prefix", func(t *testing.T) {
		id := Generate(SourcePrefix)
		assert.True(t, strings.HasPrefix(id, "src_"))
	})
	t.Run("workspace prefix", func(t *testing.T) {
		id := Generate(WorkspacePrefix)
		assert.True(t, strings.HasPrefix(id, "ws_"))
	})
}
