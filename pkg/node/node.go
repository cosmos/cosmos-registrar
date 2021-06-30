package node

import (
	"bufio"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/md5"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/jackzampolin/cosmos-registrar/pkg/utils"
	"github.com/tendermint/tendermint/libs/log"
	rpchttp "github.com/tendermint/tendermint/rpc/client/http"
	ctypes "github.com/tendermint/tendermint/rpc/core/types"
	libclient "github.com/tendermint/tendermint/rpc/jsonrpc/client"
	"github.com/tendermint/tendermint/types"
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

func SaveLightRoots(basePath, chainID string, lr *LightRoot, logger log.Logger) (err error) {
	repoRoot := repoDir{basePath, chainID}
	f, err := os.Open(repoRoot.heights())
	if err != nil {
		return
	}
	r := bufio.NewReader(f)
	lrh, err := parseLightRootHistory(r)
	if err != nil {
		return
	}

	lrh = append(lrh, *lr)
	err = utils.ToJSON(repoRoot.heights(), lrh)
	return
}

func contactPeer(p *Peer, np *NodePool, wg *sync.WaitGroup, logger log.Logger) {
	defer wg.Done()
	client, e := Client(p.Address)
	ctx := context.Background()

	if e != nil {
		logger.Error("error creating tendermint client: %s", e)
		return
	}

	// TODO: in a more advanced version of this tool,
	// this would crawl the network a couple of hops
	// and find more peers

	netInfo, err := client.NetInfo(ctx)
	if err != nil {
		return
	}
	logger.Debug("GET /net_info", "rpc-addr", p.Address)
	for _, p := range netInfo.Peers {
		peer := &Peer{
			ID:                string(p.NodeInfo.DefaultNodeID),
			Address:           fmt.Sprintf("http://%s:26657", p.RemoteIP),
			IsSeed:            false,
			LastContactHeight: 0,
			LastContactDate:   time.Time{},
			UpdatedAt:         time.Now(),
			Reachable:         false,
		}
		// reach out to the peers of the peer and record if they're at
		// least up
		go up(ctx, peer, np, wg, logger)
		wg.Add(1)
	}
}

func up(ctx context.Context, peer *Peer, np *NodePool, wg *sync.WaitGroup, logger log.Logger) {
	ctx, cancel := context.WithTimeout(ctx, 1*time.Second)
	defer cancel()
	defer wg.Done()

	peer.Contact(ctx, logger)
	if peer.Reachable {
		np.AddNode(peer.ID, peer)
	}
}

// RefreshPeers asks a peer to give its list of peers, then tries to contact
// them on 26657 to see if they're up.
func RefreshPeers(peers map[string]*Peer, logger log.Logger) (peersReachable map[string]*Peer) {
	// for each peer available
	// in the list, contact the known peers
	// and add them to the channel
	np := NewNodePool()
	wg := sync.WaitGroup{}

	for _, p := range peers {
		go contactPeer(p, np, &wg, logger)
		wg.Add(1)
	}
	wg.Wait()

	// np contains all peers that had a reachable 26657. Simply add them into
	// the answer.
	peersReachable = np.nodes
	return
}

// getLatestBlockHeight retries GET /status until the node reports a correct latest block height
func getLatestBlockHeight(peers map[string]*Peer, chainID string, logger log.Logger) (latestBlockHeight int64, err error) {
	// Transform map into list of peers
	vs := []*Peer{}
	for _, v := range peers {
		vs = append(vs, v)
	}

	for i := 0; i < len(vs); i++ {
		latestBlockHeight, err = mustGetLatestBlockHeight(vs[i], chainID, logger)
		if err == nil {
			return latestBlockHeight, err
		} else {
			logger.Debug("Peer could not tell us the latest block height", "peer", vs[i].Address)
		}
	}
	return 0, fmt.Errorf("update light roots: no peer could tell us the latest block height")
}

func mustGetLatestBlockHeight(peer *Peer, chainID string, logger log.Logger) (latestBlockHeight int64, err error) {
	ctx := context.Background()
	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()

	client, e := Client(peer.Address)
	if e != nil {
		logger.Error("error creating tendermint client: %s", e)
	}

	for {
		logger.Debug("GET /status to get latest block height", "peer", peer.Address)
		stat, err := client.Status(ctx)
		switch {
		case err != nil:
			logger.Error("error fetching client status: %s", "error", err)
			return 0, err
		case stat.NodeInfo.Network != chainID:
			logger.Error("node(%s) is on chain(%s) not configured chain(%s)", peer.Address, stat.NodeInfo.Network, chainID)
			return 0, err
		}
		if !stat.SyncInfo.CatchingUp {
			return stat.SyncInfo.LatestBlockHeight, err
		}
		time.Sleep(time.Duration(200) * time.Millisecond)
	}
}

// UpdateLightRoots asks a set a reachable peers for the blockhash at a
// specific height, and ensures they all agree. If it cannot find a majority
// answer, it returns an error
func UpdateLightRoots(chainID string, peers map[string]*Peer, logger log.Logger) (lr *LightRoot, err error) {
	// Choose a random peer to provide the latest block height
	h, err := getLatestBlockHeight(peers, chainID, logger)
	if err != nil {
		logger.Error("Couldn't get latest block height to update light root history", err)
		return nil, err
	}

	wg := sync.WaitGroup{}
	nlr := NewLightRootResults()
	for _, peer := range peers {
		go func(peer *Peer, lrr *LightRootResults, logger log.Logger) {
			defer wg.Done()
			ctx := context.Background()
			ctx, cancel := context.WithTimeout(ctx, 1*time.Second)
			defer cancel()

			client, e := Client(peer.Address)
			if e != nil {
				logger.Error("error creating tendermint client: %s", e)
			}
			logger.Debug("Asking peer for commit at", "peer", peer.Address, "height", h)
			commit, err = client.Commit(ctx, &h)
			if err != nil {
				logger.Error("error getting light roots from", "peer", peer.Address, "error", err)
				return
			}
			lr := NewLightRoot(commit.SignedHeader)
			lrr.AddResult(peer.ID, lr)
			logger.Debug("Updated light roots from", "peer", peer.Address, "peerID", peer.ID, "lightroot", lr)
		}(peer, nlr, logger)
		wg.Add(1)
	}
	wg.Wait()

	if !nlr.Same() {
		return nil, fmt.Errorf("peers reported different lightroot hashes for height %v, peerMap: %v", h, nlr)
	}

	// Return a random LightRootResult (they're all the same at this point)
	return nlr.RandomElement(), nil
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

	old, err := regexp.MatchString("v?0.3[0-3].", stat.NodeInfo.Version)
	if err != nil {
		return fmt.Errorf("error checking tendermint version: %s", err)
	}
	if old {
		return fmt.Errorf("cosmos-registrar only supports nodes with tendermint version 0.34 and up, this node is running %s", stat.NodeInfo.Version)
	}
	if stat.NodeInfo.Version == "" {
		logger.Info("Node did not report its Tendermint version, there may be compatibility problems")
	}

	eg.Go(func() error {
		if stat.NodeInfo.Network == "cosmoshub-4" {
			gen, err = cosmoshub4Workaround(ctx, client)
		} else {
			gen, err = client.Genesis(ctx)
		}
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

	// Initialize a list of historical LightRoots
	lrh := make([]*LightRoot, 1)
	lrh[0] = NewLightRoot(commit.SignedHeader)
	lrhBytes, err := json.MarshalIndent(lrh, "", "  ")
	if err != nil {
		return err
	}
	eg.Go(updateFileGo(repoRoot.heights(), lrhBytes, logger))

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
			if err = writeGzFile(repoRoot.genesisPath(), write, logger); err != nil {
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

func (p *Peer) Contact(ctx context.Context, logger log.Logger) {
	client, err := Client(p.Address)
	if err != nil {
		p.UpdatedAt = time.Now()
		p.Reachable = false
		return
	}

	res, err := client.Status(ctx)
	if err != nil {
		p.UpdatedAt = time.Now()
		p.Reachable = false
		return
	}
	logger.Debug("Confirmed reachable", "peer", p.Address)
	p.LastContactHeight = res.SyncInfo.LatestBlockHeight
	p.LastContactDate = time.Now()
	p.UpdatedAt = time.Now()
	p.Reachable = true
}

type repoDir struct {
	dir     string
	chainID string
}

func (r repoDir) chainPath() string      { return path.Join(r.dir, r.chainID) }
func (r repoDir) genesisPath() string    { return path.Join(r.chainPath(), "genesis.json") }
func (r repoDir) genesisSumPath() string { return path.Join(r.chainPath(), "genesis.json.sum") }
func (r repoDir) lrpath() string         { return path.Join(r.chainPath(), "light-roots") }
func (r repoDir) heights() string        { return path.Join(r.lrpath(), "heights.json") }

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

func writeGzFile(pth string, payload []byte, log log.Logger) (err error) {
	pth = strings.Join([]string{pth, ".gz"}, "")
	log.Debug(fmt.Sprintf("writing path %s", path.Base(pth)))
	f, err := os.Create(pth)
	if err != nil {
		return
	}

	zw := gzip.NewWriter(f)
	_, err = zw.Write(payload)
	if err != nil {
		return fmt.Errorf("writing %s: %s", pth, err)
	}
	defer zw.Close()
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

func sortedGenesis(gen *types.GenesisDoc) (sum string, indented []byte, err error) {
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

// cosmoshub4Workaround is a workaround for not being able to get cosmoshub-4's
// genesis from a node, because it is too large (problem with tendermint 0.34,
// should be fixed with 0.35)
func cosmoshub4Workaround(ctx context.Context, client *rpchttp.HTTP) (gen *ctypes.ResultGenesis, err error) {
	expected := []struct {
		Height  int64
		AppHash string
	}{
		{5200791, "E3B0C44298FC1C149AFBF4C8996FB92427AE41E4649B934CA495991B7852B855"},
		{6000000, "DCBA58D3825AE20BA8FA836AAF386497D8D18A837F4B06D51D67BD372763D4FB"},
		{6282992, "101FCD443AAEDDE4904971810EC08EF44CA06C490E8C520483E02A55C6987FF7"},
	}
	for _, e := range expected {
		commit, err := client.Commit(ctx, &e.Height)
		if err != nil {
			return nil, err
		}

		if commit.Header.AppHash.String() != e.AppHash {
			return nil, fmt.Errorf("height %b had app hash %s, expected %s", e.Height, commit.Header.AppHash.String(), e.AppHash)
		}
	}

	// Find and verify genesis.cosmoshub-4.json.gz in the current folder
	dir, err := os.Getwd()
	if err != nil {
		return
	}
	println("cosmoshub4Workaround: looking for genesis.cosmoshub-4.json.gz in", dir)
	gPath := path.Join(dir, "genesis.cosmoshub-4.json.gz")
	f, err := os.Open(gPath)
	if err != nil {
		return
	}
	defer f.Close()
	b, err := ioutil.ReadAll(f) // so we can read the file contents twice

	hReader := bytes.NewReader(b)
	h := md5.New()
	if _, err := io.Copy(h, hReader); err != nil {
		return nil, err
	}
	hActual := hex.EncodeToString(h.Sum(nil))
	hExpected := "a4216a3cae68e9190d0757c90bcb1f1b"
	if string(hActual) != hExpected {
		return nil, fmt.Errorf("%s has md5 of %s, expected %s", gPath, hActual, hExpected)
	}

	gzReader := bytes.NewReader(b)
	gr, err := gzip.NewReader(gzReader)
	if err != nil {
		return
	}
	defer gr.Close()

	j, err := ioutil.ReadAll(gr)
	if err != nil {
		return
	}
	gDoc, err := types.GenesisDocFromJSON(j)
	if err != nil {
		return
	}
	gen = &ctypes.ResultGenesis{Genesis: gDoc}
	return
}

func parseLightRootHistory(r io.Reader) (lrh LightRootHistory, err error) {
	b, err := ioutil.ReadAll(r)
	if err != nil {
		return
	}
	lrh = LightRootHistory{}
	err = json.Unmarshal(b, &lrh)
	return
}
