package node

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path"
	"regexp"
	"sort"
	"sync"
	"time"

	"github.com/jackzampolin/cosmos-registrar/pkg/utils"
	"github.com/tendermint/tendermint/libs/log"
	rpchttp "github.com/tendermint/tendermint/rpc/client/http"
	ctypes "github.com/tendermint/tendermint/rpc/core/types"
	libclient "github.com/tendermint/tendermint/rpc/jsonrpc/client"
	tmtypes "github.com/tendermint/tendermint/types"
	"golang.org/x/sync/errgroup"
)

var (
	gen    *ctypes.ResultGenesis
	commit *ctypes.ResultCommit
	eg     errgroup.Group
)

// Client returns a tendermint client to work against the configured chain
func Client(rpcAddress string) (*rpchttp.HTTP, error) {
	httpClient, err := libclient.DefaultHTTPClient(rpcAddress)
	if err != nil {
		return nil, err
	}

	rpcClient, err := rpchttp.NewWithClient(rpcAddress, "/websocket", httpClient)
	if err != nil {
		return nil, err
	}

	return rpcClient, nil
}

// FetchChainID - retrieve the chain ID from a rpc endpoint
func FetchChainID(rpcAddress string) (chainID string, err error) {
	client, err := Client(rpcAddress)
	if err != nil {
		err = fmt.Errorf("error creating tendermint client: %s", err)
		return
	}
	ctx := context.Background()
	stat, err := client.Status(ctx)
	if err != nil {
		err = fmt.Errorf("error fetching client status: %s", err)
		return
	}
	chainID = stat.NodeInfo.Network
	return
}

// LoadGenesisSum load the genesis checksum file
func LoadGenesisSum(basePath, chainID string) (sum string, err error) {
	repoRoot := repoDir{basePath, chainID}
	if !utils.PathExists(repoRoot.genesisSumPath()) {
		err = fmt.Errorf("path %s does not exists", repoRoot.genesisSumPath())
		return
	}
	fp, err := os.Open(repoRoot.genesisSumPath())
	if err != nil {
		return
	}
	defer fp.Close()
	raw, err := io.ReadAll(fp)
	if err != nil {
		return
	}
	sum = string(raw)
	return
}

// LoadPeers load the information about the chain nodes
func LoadPeers(basePath, chainID, rpcAddress string, logger log.Logger) (peers map[string]*Peer, err error) {
	repoRoot := repoDir{basePath, chainID}
	if !utils.PathExists(repoRoot.peersPath()) {
		return
	}
	// read the list of peers
	peerList := []Peer{}
	err = utils.FromJSON(repoRoot.peersPath(), &peerList)
	if err != nil {
		return
	}
	peers = make(map[string]*Peer)
	// map them to a map
	for _, p := range peerList {
		peers[p.ID] = &p
	}
	return
}

func SavePeers(basePath, chainID string, peers map[string]*Peer, logger log.Logger) (err error) {
	// sort the peer keys
	peerKeys := make([]string, 0, len(peers))
	for k := range peers {
		peerKeys = append(peerKeys, k)
	}
	sort.Strings(peerKeys)
	// create a slice of peers sorted by ID
	peerData := make([]*Peer, 0, len(peerKeys))
	for _, k := range peerKeys {
		peerData = append(peerData, peers[k])
	}
	// write the list to disk
	repoRoot := repoDir{basePath, chainID}
	err = utils.ToJSON(repoRoot.peersPath(), peerData)
	return
}

func RefreshPeers(peers map[string]*Peer, genesisSum string, logger log.Logger) (err error) {
	// for each pear available
	// in the list, contact the known peers
	// and add them to the list
	wg := sync.WaitGroup{}

	for _, p := range peers {
		go func(peer *Peer) {
			defer wg.Done()
			client, e := Client(p.Address)
			ctx := context.Background()

			if e != nil {
				err = fmt.Errorf("error creating tendermint client: %s", err)
				return
			}
			gen, err = client.Genesis(ctx)

			// TODO: in a more advanced version of this tool,
			// this would crawl the network a couple of hops
			// and find more peers

			netInfo, err := client.NetInfo(ctx)
			if err != nil {
				return
			}
			logger.Debug("GET /net_info", "rpc-addr", p.Address)
			for _, p := range netInfo.Peers {
				fmt.Println(p)
			}
		}(p)
		wg.Add(1)
	}
	wg.Wait()
	return
}

// DumpInfo connect to ad node and dumps the info about
// that chain into a folder
func DumpInfo(basePath, chainID, rpcAddress string, logger log.Logger) (err error) {
	client, err := Client(rpcAddress)
	if err != nil {
		err = fmt.Errorf("error creating tendermint client: %s", err)
		return
	}
	ctx := context.Background()
	stat, err := client.Status(ctx)
	switch {
	case err != nil:
		err = fmt.Errorf("error fetching client status: %s", err)
		return
	case stat.NodeInfo.Network != chainID:
		err = fmt.Errorf("node(%s) is on chain(%s) not configured chain(%s)", rpcAddress, stat.NodeInfo.Network, chainID)
		return
	case stat.SyncInfo.CatchingUp:
		err = fmt.Errorf("node(%s) on chain(%s) still catching up", rpcAddress, chainID)
		return
	default:
		logger.Debug("GET /status", "rpc-addr", rpcAddress)
	}

	old, err := regexp.MatchString("[0-9].3[0-3]*", stat.NodeInfo.Version)
	if err != nil {
		return fmt.Errorf("error checking tendermint version: %s", err)
	}
	if old {
		return fmt.Errorf("cosmos-registrar only supports nodes with tendermint version 0.34 and up, this node is running %s", stat.NodeInfo.Version)
	}

	eg.Go(func() error {
		gen, err = client.Genesis(ctx)
		if err != nil {
			return fmt.Errorf("genesis file: %s", err)
		}
		logger.Debug("GET /genesis", "rpc-addr", rpcAddress)
		return nil
	})

	eg.Go(func() error {
		h := stat.SyncInfo.LatestBlockHeight
		commit, err = client.Commit(ctx, &h)
		if err != nil {
			return err
		}
		logger.Debug(fmt.Sprintf("GET /commit?height=%d", h), "rpc-addr", rpcAddress)
		return nil
	})

	if err = eg.Wait(); err != nil {
		err = fmt.Errorf("fetching: %s", err)
		return
	}
	// fetch data
	repoRoot := repoDir{basePath, chainID}
	if err = createDirIfNotExist(repoRoot.chainPath(), logger); err != nil {
		return
	}
	if err = createDirIfNotExist(repoRoot.lrpath(), logger); err != nil {
		return
	}
	// TODO: sanity checks on the genesis file returned from the chain compared with repo
	eg.Go(updateFileGo(repoRoot.latestPath(), NewLightRoot(commit.SignedHeader), logger))
	eg.Go(updateFileGo(repoRoot.heightPath(commit.SignedHeader.Header.Height), NewLightRoot(commit.SignedHeader), logger))
	// TODO: not sure about this one, but we should be able to get the node version from the rpc
	// eg.Go(updateFileGo(repoRoot.binariesPath(), config.Binary(), logger))
	eg.Go(func() error {
		if _, err = os.Stat(repoRoot.genesisPath()); os.IsNotExist(err) {
			sum, write, err := sortedGenesis(gen.Genesis)
			if err != nil {
				return fmt.Errorf("sorting genesis file: %s", err)
			}

			if err = writeFile(repoRoot.genesisSumPath(), []byte(sum), logger); err != nil {
				return err
			}
			if err = writeFile(repoRoot.genesisPath(), write, logger); err != nil {
				return err
			}
		}
		return nil
	})
	eg.Go(func() error {
		updateTime := time.Now()
		seedNode := Peer{
			IsSeed:            true,
			Reachable:         true,
			Address:           rpcAddress,
			ID:                fmt.Sprint(stat.NodeInfo.ID()),
			LastContactHeight: stat.SyncInfo.LatestBlockHeight,
			LastContactDate:   updateTime,
			UpdatedAt:         updateTime,
		}
		out, err := json.Marshal([]Peer{seedNode})
		if err != nil {
			return fmt.Errorf("marshaling peers: %s", err)
		}
		return writeFile(repoRoot.peersPath(), out, logger)
	})

	err = eg.Wait()
	return
}

// Peer structure to keep track of the status of a peer
type Peer struct {
	ID                string    `json:"id,omitempty"`
	Address           string    `json:"address,omitempty"`
	IsSeed            bool      `json:"is_seed,omitempty"`
	LastContactHeight int64     `json:"last_contact_height,omitempty"`
	LastContactDate   time.Time `json:"last_contact_date,omitempty"`
	UpdatedAt         time.Time `json:"updated_at,omitempty"`
	Reachable         bool      `json:"reachable,omitempty"`
}

type repoDir struct {
	dir     string
	chainID string
}

func (r repoDir) chainPath() string         { return path.Join(r.dir, r.chainID) }
func (r repoDir) genesisPath() string       { return path.Join(r.chainPath(), "genesis.json") }
func (r repoDir) genesisSumPath() string    { return path.Join(r.chainPath(), "genesis.json.sum") }
func (r repoDir) lrpath() string            { return path.Join(r.chainPath(), "light-roots") }
func (r repoDir) latestPath() string        { return path.Join(r.lrpath(), "latest.json") }
func (r repoDir) heightPath(h int64) string { return path.Join(r.lrpath(), fmt.Sprintf("%d.json", h)) }

func (r repoDir) peersPath() string { return path.Join(r.chainPath(), "peers.json") }

func updateFileGo(pth string, payload []byte, log log.Logger) func() error {
	return func() (err error) {
		return updateFile(pth, payload, log)
	}
}

func updateFile(pth string, payload []byte, log log.Logger) error {
	log.Debug(fmt.Sprintf("deleting path %s", path.Base(pth)))
	os.Remove(pth)
	return writeFile(pth, payload, log)
}

func writeFile(pth string, payload []byte, log log.Logger) (err error) {
	log.Debug(fmt.Sprintf("writing path %s", path.Base(pth)))
	if err = ioutil.WriteFile(pth, payload, 0644); err != nil {
		return fmt.Errorf("writing %s: %s", pth, err)
	}
	return nil
}

func createDirIfNotExist(pth string, log log.Logger) (err error) {
	if _, err = os.Stat(pth); os.IsNotExist(err) {
		log.Debug("creating directory", "dir", path.Base(pth))
		if err = os.Mkdir(pth, os.ModePerm); err != nil {
			return fmt.Errorf("making dir %s: %s", pth, err)
		}
	}
	return nil
}

func sortedGenesis(gen *tmtypes.GenesisDoc) (sum string, indented []byte, err error) {
	// prepare to sort
	if indented, err = json.Marshal(gen); err != nil {
		return
	}

	// sort
	var c interface{}
	if err = json.Unmarshal(indented, &c); err != nil {
		return
	}

	// indent
	if indented, err = json.MarshalIndent(c, "", "  "); err != nil {
		return
	}

	// sum
	sum = fmt.Sprintf("%x", sha256.Sum256(indented))
	return
}

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
