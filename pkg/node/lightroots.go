package node

import (
	"encoding/json"

	tmtypes "github.com/tendermint/tendermint/types"
)

// LightRoot is the format for a light client root file which
// will be used for state sync
type LightRoot struct {
	TrustHeight int64  `json:"trust-height"`
	TrustHash   string `json:"trust-hash"`
}

// NewLightRoot returns a new light root
func NewLightRoot(sh tmtypes.SignedHeader) []byte {
	out, _ := json.MarshalIndent(&LightRoot{
		TrustHeight: sh.Header.Height,
		TrustHash:   sh.Commit.BlockID.Hash.String(),
	}, "", "  ")
	return out
}
