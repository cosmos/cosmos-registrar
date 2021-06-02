package cmd

import (
	"strings"
	"testing"

	registrar "github.com/jackzampolin/cosmos-registrar/pkg/config"
	"github.com/noandrea/go-codeowners"
	"github.com/stretchr/testify/assert"
)

const testCO = `*                         @okwme @jackzampolin @cosmos/ape-unit_registry_write
/akashnet-1/              @jackzampolin
/cosmoshub-3/             @jackzampolin
/pooltoy-2/               @okwme
/cosmoshub-4/             @randomshinichi
`

func TestMyChains(t *testing.T) {
	co, err := codeowners.FromReader(strings.NewReader(testCO), "/test/reporoot")
	assert.Nil(t, err)

	chainIDs := myChains(co, &registrar.Config{GitName: "randomshinichi"})
	assert.Equal(t, []string{"cosmoshub-4"}, chainIDs)

	assert.Equal(t, []string{}, myChains(co, &registrar.Config{GitName: "nobody"}))

	assert.Equal(t, []string{"akashnet-1", "cosmoshub-3", "pooltoy-2", "cosmoshub-4"}, myChains(co, &registrar.Config{GitName: "jackzampolin"}))

	assert.Equal(t, []string{"akashnet-1", "cosmoshub-3", "pooltoy-2", "cosmoshub-4"}, myChains(co, &registrar.Config{GitName: "okwme"}))

	assert.Equal(t, []string{"akashnet-1", "cosmoshub-3", "pooltoy-2", "cosmoshub-4"}, myChains(co, &registrar.Config{GitName: "cosmos/ape-unit_registry_write"}))

}
