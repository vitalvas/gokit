//go:build darwin

package machineid

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExtractID(t *testing.T) {
	t.Run("valid ioreg output", func(t *testing.T) {
		input := `+-o Root  <class IORegistryEntry, id 0x100000100, retain 20>
    {
      "IOKitBuildVersion" = "Darwin Kernel Version 21.6.0"
      "IOPlatformUUID" = "550E8400-E29B-41D4-A716-446655440000"
      "IOPolledInterface" = "SMCPolledInterface is not serializable"
    }`
		id, err := extractID(input)
		require.NoError(t, err)
		assert.Equal(t, "550E8400-E29B-41D4-A716-446655440000", id)
	})

	t.Run("uuid with lowercase", func(t *testing.T) {
		input := `      "IOPlatformUUID" = "550e8400-e29b-41d4-a716-446655440000"`
		id, err := extractID(input)
		require.NoError(t, err)
		assert.Equal(t, "550e8400-e29b-41d4-a716-446655440000", id)
	})

	t.Run("missing IOPlatformUUID", func(t *testing.T) {
		input := `+-o Root  <class IORegistryEntry, id 0x100000100, retain 20>
    {
      "IOKitBuildVersion" = "Darwin Kernel Version 21.6.0"
    }`
		_, err := extractID(input)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "IOPlatformUUID")
	})

	t.Run("empty input", func(t *testing.T) {
		_, err := extractID("")
		assert.Error(t, err)
	})

	t.Run("malformed IOPlatformUUID line", func(t *testing.T) {
		input := `      "IOPlatformUUID" = broken`
		_, err := extractID(input)
		assert.Error(t, err)
	})
}

func TestMachineID_Darwin(t *testing.T) {
	t.Run("returns valid UUID format", func(t *testing.T) {
		id, err := machineID()
		require.NoError(t, err)
		assert.NotEmpty(t, id)
		// UUID format: 8-4-4-4-12 hex characters
		assert.Regexp(t, `^[0-9A-Fa-f]{8}-[0-9A-Fa-f]{4}-[0-9A-Fa-f]{4}-[0-9A-Fa-f]{4}-[0-9A-Fa-f]{12}$`, id)
	})
}

func BenchmarkExtractID(b *testing.B) {
	input := `+-o Root  <class IORegistryEntry, id 0x100000100, retain 20>
    {
      "IOKitBuildVersion" = "Darwin Kernel Version 21.6.0"
      "IOPlatformUUID" = "550E8400-E29B-41D4-A716-446655440000"
      "IOPolledInterface" = "SMCPolledInterface is not serializable"
    }`
	b.ReportAllocs()
	for b.Loop() {
		_, _ = extractID(input)
	}
}
