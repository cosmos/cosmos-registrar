package node

import "sync"

// NodePool is a list with a Mutex wrapped around it so that only one goroutine
// can write to it at any one time. goroutines created by RefreshPeers() write
// their final results here.
type NodePool struct {
	rw    sync.RWMutex
	nodes []*Peer
}

// Size returns the size of the pool.
func (np *NodePool) Size() int {
	np.rw.RLock()
	defer np.rw.RUnlock()
	return len(np.nodes)
}

func (np *NodePool) AddNode(p *Peer) {
	np.rw.Lock()
	defer np.rw.Unlock()
	np.nodes = append(np.nodes, p)
}
