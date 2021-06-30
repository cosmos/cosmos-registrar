package node

import (
	"bytes"
	"context"
	"encoding/json"
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
	peer1 := &Peer{
		ID:                "12033793a528b55aa40ed9d8354bb5b19520a718",
		Address:           "http://hub.technofractal.com:26657",
		IsSeed:            true,
		LastContactHeight: 6325797,
		LastContactDate:   time.Now(),
		UpdatedAt:         time.Now(),
		Reachable:         true,
	}
	peer2 := &Peer{
		ID:                "cfd785a4224c7940e9a10f6c1ab24c343e923bec",
		Address:           "http://164.68.107.188:26657",
		IsSeed:            true,
		LastContactHeight: 6325797,
		LastContactDate:   time.Now(),
		UpdatedAt:         time.Now(),
		Reachable:         true,
	}
	peer3 := &Peer{
		ID:                "a6f325ea73533648fd3176e612915a83e2a2572f",
		Address:           "http://139.59.70.20:26657",
		IsSeed:            true,
		LastContactHeight: 6325797,
		LastContactDate:   time.Now(),
		UpdatedAt:         time.Now(),
		Reachable:         true,
	}
	pm := map[string]*Peer{peer1.ID: peer1, peer2.ID: peer2, peer3.ID: peer3}
	peersReachable := RefreshPeers(pm, logger)
	fmt.Println("original peers map", pm)

	raw, err := json.MarshalIndent(peersReachable, "", "  ")
	assert.Nil(t, err)
	fmt.Println("New peers map", string(raw))
}

func TestUpdateLightRoots(t *testing.T) {
	logger := log.NewTMLogger(log.NewSyncWriter(os.Stdout))
	peer1 := &Peer{
		ID:                "12033793a528b55aa40ed9d8354bb5b19520a718",
		Address:           "http://hub.technofractal.com:26657",
		IsSeed:            true,
		LastContactHeight: 6325797,
		LastContactDate:   time.Now(),
		UpdatedAt:         time.Now(),
		Reachable:         true,
	}
	peer2 := &Peer{
		ID:                "cfd785a4224c7940e9a10f6c1ab24c343e923bec",
		Address:           "http://164.68.107.188:26657",
		IsSeed:            true,
		LastContactHeight: 6325797,
		LastContactDate:   time.Now(),
		UpdatedAt:         time.Now(),
		Reachable:         true,
	}
	peer3 := &Peer{
		ID:                "a6f325ea73533648fd3176e612915a83e2a2572f",
		Address:           "http://139.59.70.20:26657",
		IsSeed:            true,
		LastContactHeight: 6325797,
		LastContactDate:   time.Now(),
		UpdatedAt:         time.Now(),
		Reachable:         true,
	}
	pm := map[string]*Peer{peer1.ID: peer1, peer2.ID: peer2, peer3.ID: peer3}
	_, err := UpdateLightRoots("cosmoshub-4", pm, logger)
	assert.Nil(t, err)
}

func TestParseLightRootHistory(t *testing.T) {
	testJSON := []byte(`[{"trust-height":6766717,"trust-hash":"2DD62FE8D0358822D245C8C4AD91489DC0031CF35BE193CC561868DB79A51904"},{"trust-height":6766806,"trust-hash":"C21AFE7C2AB4927497DD239568FB9F819120D70EAF254E970185F88502806F1E"},{"trust-height":6766886,"trust-hash":"A6B573B433D69ABC6F07216B572E21BE8DCF83CD037E8898B379F6978D3085DE"}]`)
	r := bytes.NewReader(testJSON)
	_, err := parseLightRootHistory(r)
	assert.Nil(t, err)
}
