package node

import (
	"context"
	"flag"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/tendermint/tendermint/libs/log"
)

var online = flag.Bool("online", false, "perform tests that require a network connection")

func TestCosmoshub4Workaround(t *testing.T) {
	ctx := context.Background()
	client, err := Client("https://rpc.cosmos.network:443")
	assert.Nil(t, err)

	gen, err = cosmoshub4Workaround(ctx, client)
	fmt.Println("gen", gen)
	assert.Nil(t, err)
}

// TestRefreshPeers is more of a dev harness, since it requires a network
// connection and has no objectively correct result
func TestRefreshPeers(t *testing.T) {
	// if !*online {
	// 	t.Skip("skipping test in offline mode")
	// }
	logger := log.NewTMLogger(log.NewSyncWriter(os.Stdout))
	sPeer := &Peer{
		ID:                "12033793a528b55aa40ed9d8354bb5b19520a718",
		Address:           "http://hub.technofractal.com:26657",
		IsSeed:            true,
		LastContactHeight: 6325797,
		LastContactDate:   time.Now(),
		UpdatedAt:         time.Now(),
		Reachable:         true,
	}
	pm := map[string]*Peer{sPeer.ID: sPeer}
	RefreshPeers(pm, logger)
	fmt.Println("peers map", pm)
}
