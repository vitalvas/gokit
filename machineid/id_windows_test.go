//go:build windows

package machineid

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMachineID_Windows(t *testing.T) {
	t.Run("returns non-empty id", func(t *testing.T) {
		id, err := machineID()
		require.NoError(t, err)
		assert.NotEmpty(t, id)
	})
}
