package node

import (
	"reflect"
	"sync"

	tmtypes "github.com/tendermint/tendermint/types"
)

// LightRoot is the format for a light client root file which
// will be used for state sync
type LightRoot struct {
	TrustHeight int64  `json:"trust-height"`
	TrustHash   string `json:"trust-hash"`
}

// NewLightRoot returns a new light root
func NewLightRoot(sh tmtypes.SignedHeader) *LightRoot {
	return &LightRoot{
		TrustHeight: sh.Header.Height,
		TrustHash:   sh.Commit.BlockID.Hash.String(),
	}
}

type LightRootHistory = []LightRoot

type result struct {
	PeerID    string
	LightRoot *LightRoot
}

// LightRootResults is a map with a Mutex wrapped around it so that only one goroutine
// can write to it at any one time. goroutines created by RefreshPeers() write
// their final results here.
type LightRootResults struct {
	rw         sync.RWMutex
	lightroots map[string]*LightRoot
}

// Size returns the size of the pool.
func (h *LightRootResults) Size() int {
	h.rw.RLock()
	defer h.rw.RUnlock()
	return len(h.lightroots)
}

func (h *LightRootResults) AddResult(peerID string, lr *LightRoot) {
	h.rw.Lock()
	defer h.rw.Unlock()
	h.lightroots[peerID] = lr
}

func (h *LightRootResults) RandomElement() *LightRoot {
	h.rw.Lock()
	defer h.rw.Unlock()
	var val *LightRoot
	for _, v := range h.lightroots {
		val = v
	}
	return val
}

func (h *LightRootResults) Same() bool {
	h.rw.RLock()
	defer h.rw.RUnlock()

	// transform values into a list of results
	r := []result{}
	for peerID, lr := range h.lightroots {
		r = append(r, result{peerID, lr})
	}

	// compare the LightRoots in the list
	for i := 1; i < len(r); i++ {
		if !reflect.DeepEqual(r[i].LightRoot, r[0].LightRoot) {
			return false
		}
	}
	return true
}

func NewLightRootResults() *LightRootResults {
	n := new(LightRootResults)
	n.lightroots = make(map[string]*LightRoot)
	return n
}
