package manifestutils

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestName(t *testing.T) {
	for _, test := range []struct {
		name      string
		component string
		expected  string
	}{
		{
			name:      "foo",
			component: "ingester",
			expected:  "tempo-foo-ingester",
		},
		{
			name:     "bar",
			expected: "tempo-bar",
		},
	} {
		t.Run(test.expected, func(t *testing.T) {
			got := Name(test.component, test.name)
			assert.Equal(t, test.expected, got)
		})
	}
}
