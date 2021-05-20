package node

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCosmoshub4Workaround(t *testing.T) {
	ctx := context.Background()
	client, err := Client("https://rpc.cosmos.network:443")
	assert.Nil(t, err)

	gen, err = cosmoshub4Workaround(ctx, client)
	fmt.Println("gen", gen)
	assert.Nil(t, err)
}
