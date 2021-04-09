package node

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/url"
	"os"
	"path"
	"strings"

	"github.com/tendermint/tendermint/libs/log"
	"github.com/tendermint/tendermint/p2p"
	rpchttp "github.com/tendermint/tendermint/rpc/client/http"
	ctypes "github.com/tendermint/tendermint/rpc/core/types"
	libclient "github.com/tendermint/tendermint/rpc/jsonrpc/client"
	tmtypes "github.com/tendermint/tendermint/types"
	"golang.org/x/sync/errgroup"
)

var (
	gen     *ctypes.ResultGenesis
	commit  *ctypes.ResultCommit
	netInfo *ctypes.ResultNetInfo
	rdir    repoDir
	eg      errgroup.Group
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

// LoadPeers load the information about the chain nodes
func LoadPeers(basepath, chainID, rpcAddress string, logger log.Logger) (peers []string, err error) {
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
			return fmt.Errorf("commit: %s", err)
		}
		logger.Debug(fmt.Sprintf("GET /commit?height=%d", h), "rpc-addr", rpcAddress)
		return nil
	})

	// TODO: in a more advanced version of this tool,
	// this would crawl the network a couple of hops
	// and find more peers
	eg.Go(func() error {
		netInfo, err = client.NetInfo(ctx)
		if err != nil {
			return fmt.Errorf("net-info: %s", err)
		}
		logger.Debug("GET /net_info", "rpc-addr", rpcAddress)
		return nil
	})

	if err = eg.Wait(); err != nil {
		err = fmt.Errorf("fetching: %s", err)
		return
	}
	// fetch data
	rdir := repoDir{basePath, chainID}
	if err = createDirIfNotExist(rdir.chainPath(), logger); err != nil {
		return
	}
	if err = createDirIfNotExist(rdir.lrpath(), logger); err != nil {
		return
	}
	// TODO: sanity checks on the genesis file returned from the chain compared with repo
	eg.Go(updateFileGo(rdir.latestPath(), NewLightRoot(commit.SignedHeader), logger))
	eg.Go(updateFileGo(rdir.heightPath(commit.SignedHeader.Header.Height), NewLightRoot(commit.SignedHeader), logger))
	// TODO: not sure about this one, but we should be able to get the node version from the rpc
	// eg.Go(updateFileGo(rdir.binariesPath(), config.Binary(), logger))
	eg.Go(func() error {
		if _, err = os.Stat(rdir.genesisPath()); os.IsNotExist(err) {
			sum, write, err := sortedGenesis(gen.Genesis)
			if err != nil {
				return fmt.Errorf("sorting genesis file: %s", err)
			}

			if err = writeFile(rdir.genesisSumPath(), []byte(sum), logger); err != nil {
				return err
			}
			if err = writeFile(rdir.genesisPath(), write, logger); err != nil {
				return err
			}
		}
		return nil
	})
	eg.Go(func() error {
		// add the current node
		u, _ := url.Parse(rpcAddress)
		entryNodeURL := nodeCoordinatesToStr(stat.NodeInfo.ID(), u.Hostname(), u.Port())
		qp := []string{entryNodeURL}
		// add the peer nodes
		qp = append(qp, stringsFromPeers(netInfo.Peers)...)
		if _, err = os.Stat(rdir.peersPath()); os.IsNotExist(err) {
			logger.Debug("no peers file, populating from /net_info", "num", len(qp))
			out, err := json.MarshalIndent(qp, "", "  ")
			if err != nil {
				return fmt.Errorf("marshaling peers: %s", err)
			}
			return writeFile(rdir.peersPath(), out, logger)
		}

		var fp []string
		pf, err := os.Open(rdir.peersPath())
		if err != nil {
			return fmt.Errorf("opening peer file: %s", err)
		}
		pfb, err := ioutil.ReadAll(pf)
		if err != nil {
			pf.Close()
			return fmt.Errorf("reading peer file: %s", err)
		}
		if err = json.Unmarshal(pfb, &fp); err != nil {
			pf.Close()
			return fmt.Errorf("unmarshaling peer strings: %s", err)
		}
		pf.Close()
		ps := dedupe(append(fp, qp...))
		// TODO: we should check peer liveness here
		logger.Debug(fmt.Sprintf("added %d new peers to %s", len(ps)-len(fp), path.Base(rdir.peersPath())))
		w, err := json.MarshalIndent(ps, "", "  ")
		if err != nil {
			return fmt.Errorf("marshaling peers: %s", err)
		}
		return updateFile(rdir.peersPath(), w, logger)
	})

	err = eg.Wait()
	return
}

type repoDir struct {
	dir     string
	chainID string
}

func (r repoDir) chainPath() string         { return path.Join(r.dir) }
func (r repoDir) genesisPath() string       { return path.Join(r.chainPath(), "genesis.json") }
func (r repoDir) genesisSumPath() string    { return path.Join(r.chainPath(), "genesis.json.sum") }
func (r repoDir) lrpath() string            { return path.Join(r.chainPath(), "light-roots") }
func (r repoDir) latestPath() string        { return path.Join(r.lrpath(), "latest.json") }
func (r repoDir) heightPath(h int64) string { return path.Join(r.lrpath(), fmt.Sprintf("%d.json", h)) }
func (r repoDir) binariesPath() string      { return path.Join(r.chainPath(), "binaries.json") }
func (r repoDir) peersPath() string         { return path.Join(r.chainPath(), "peers.json") }

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

func stringsFromPeers(ni []ctypes.Peer) (qp []string) {
	for _, p := range ni {
		port := strings.Split(p.NodeInfo.ListenAddr, ":")
		qp = append(qp, nodeCoordinatesToStr(p.NodeInfo.ID(), p.RemoteIP, port[len(port)-1]))
	}
	return
}

// stringify a node rpc address and id
func nodeCoordinatesToStr(id p2p.ID, ip, port string) string {
	if port == "" {
		// avoid having a blank port
		port = "26657"
	}
	return fmt.Sprintf("%s@%s:%s", id, ip, port)
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

func dedupe(ele []string) (out []string) {
	e := map[string]bool{}
	for v := range ele {
		e[ele[v]] = true
	}
	for k := range e {
		out = append(out, k)
	}
	return
}
