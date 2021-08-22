package global

import (
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestVersionNameCompare(t *testing.T) {
	tests := [...]struct {
		current  string
		remote   string
		expected bool
	}{
		// Normal Tests:
		{"v0.9.29-fix2", "v0.9.29-fix2", false},
		{"v0.9.29-fix1", "v0.9.29-fix2", true},
		{"v0.9.29-fix2", "v0.9.29-fix1", false},
		{"v0.9.29-fix2", "v0.9.30", true},
		{"v1.0.0-alpha", "v1.0.0-alpha2", true},
		{"v1.0.0-alpha2", "v1.0.0-beta1", true},
		{"v1.0.0", "v1.0.0-beta1", false},
		{"v1.0.0-alpha", "v1.0.0", true},
		{"v1.0.0", "v1.0.0", false},
		{"v1.0.0-alpha", "v1.0.0-rc1", true},

		// Issue Fixes:
		{"v1.0.0-beta1", "v0.9.40-fix5", false}, // issue #877
	}
	for i := 0; i < len(tests); i++ {
		t.Run("test case "+strconv.Itoa(i), func(t *testing.T) {
			assert.Equal(t, tests[i].expected, VersionNameCompare(tests[i].current, tests[i].remote))
		})
	}
}
