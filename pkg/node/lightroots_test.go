package node

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLightRootResultsSame(t *testing.T) {
	lrr := NewLightRootResults()
	lrr.AddResult("cfd785a4224c7940e9a10f6c1ab24c343e923bec", &LightRoot{6765853, "E37F5936731F7F0FF35497255C666B91E719896A2E1E2F55A778A970AF92157E"})
	lrr.AddResult("a6f325ea73533648fd3176e612915a83e2a2572f", &LightRoot{6765853, "E37F5936731F7F0FF35497255C666B91E719896A2E1E2F55A778A970AF92157E"})

	assert.True(t, lrr.Same())
}
