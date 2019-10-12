package http2raft

import (
	"errors"
	"flag"
	"fmt"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/lni/dragonboat/v3"
	"github.com/lni/dragonboat/v3/config"
	"github.com/lni/dragonboat/v3/logger"
	"github.com/lni/dragonboat/v3/statemachine"
)

const (
	defaultNodeID    = 1
	defaultClusterID = 1
)

var (
	errBadPeersSyntax = errors.New("bad syntax for 'peers' param")
)

var (
	nodeID    = flag.Int("raft_node_id", defaultNodeID, "raft node ID to use")
	clusterID = flag.Int("raft_cluster_id", defaultClusterID, "raft cluster ID to use")
	raftAddr  = flag.String("raft_addr", "", "raft node address")
	dataDir   = flag.String("data_dir", "./", "raft node data directory")
	peersStr  = flag.String("peers", "",
		"raft peers list, comma separated list of values in format nodeID:host:port")
	join         = flag.Bool("join", false, "joining a new node")
	readTimeout  = flag.Duration("read_timeout", 3*time.Second, "time out for read operations")
	writeTimeout = flag.Duration("write_timeout", 3*time.Second, "time out for write operations")
)

// Start starts taft-node, sets up routing amd starts HTTP listener
func Start(addr string, smFactoryFunc func(clusterID, nodeID uint64) statemachine.IStateMachine) {
	// start raft node
	raftNode, err := startRaftNode(smFactoryFunc)
	if err != nil {
		// there is no reason to continue if no raft
		panic(err)
	}
	defer raftNode.Stop()

	// prepare controller
	keys := keysController{
		readTimeOut:   *readTimeout,
		writeTimeOut:  *writeTimeout,
		clusterID:     uint64(*clusterID),
		raftNode:      raftNode,
		clientSession: raftNode.GetNoOPSession(uint64(*clusterID)),
	}

	// setup routing
	// catch all traffic as for now, the URL path will act as key name (query string is ignored)
	keyHandler := func(w http.ResponseWriter, req *http.Request) {
		switch req.Method {
		case http.MethodGet:
			// GET requests translates to raft SyncRead
			keys.readKey(w, req)
		case http.MethodPost, http.MethodPut, http.MethodDelete, http.MethodPatch:
			// POST, PUT, DELETE, PATCH requests translates to raft SyncPropose
			keys.writeKey(w, req)
		default:
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
		}
	}
	http.HandleFunc("/", keyHandler)

	// start http listener
	http.ListenAndServe(addr, nil)
}

func startRaftNode(smFactoryFunc func(clusterID, nodeID uint64) statemachine.IStateMachine) (*dragonboat.NodeHost, error) {
	// adjust logging level
	logger.GetLogger("raft").SetLevel(logger.WARNING)
	logger.GetLogger("rsm").SetLevel(logger.WARNING)
	logger.GetLogger("transport").SetLevel(logger.WARNING)

	// populate peers list (if any specified in param)
	peers := make(map[uint64]string)
	if *peersStr != "" {
		for _, pStr := range strings.Split(*peersStr, ",") {
			peerParts := strings.Split(pStr, ":")
			if len(peerParts) < 3 || peerParts[0] == "" || peerParts[1] == "" || peerParts[2] == "" {
				flag.Usage()
				return nil, errBadPeersSyntax
			}
			peerNodeID, err := strconv.Atoi(peerParts[0])
			if err != nil {
				flag.Usage()
				return nil, errBadPeersSyntax
			}
			peers[uint64(peerNodeID)] = peerParts[1] + ":" + peerParts[2]
		}
		// set current raft node addr from peers if it wasn't passed via raft_addr param
		if *raftAddr == "" && len(peers) > 0 {
			*raftAddr = peers[uint64(*nodeID)]
		}
	}

	if *raftAddr == "" {
		return nil, errors.New("raft_addr parameter is required")
	}

	// prepare configs
	// TODO: move hardcoded numbers to CLI params
	raftConfig := config.Config{
		NodeID:             uint64(*nodeID),
		ClusterID:          uint64(*clusterID),
		ElectionRTT:        10,
		HeartbeatRTT:       1,
		CheckQuorum:        true,
		SnapshotEntries:    10,
		CompactionOverhead: 5,
	}
	dataDirPath := filepath.Join(
		*dataDir,
		"http2raft",
		fmt.Sprintf("cluster-%d", *clusterID),
		fmt.Sprintf("node-%d", *nodeID),
	)
	nodeConfig := config.NodeHostConfig{
		WALDir:         dataDirPath,
		NodeHostDir:    dataDirPath,
		RTTMillisecond: 200,
		RaftAddress:    *raftAddr,
		EnableMetrics:  true,
	}

	// create node and start/join the cluster
	raftNode, err := dragonboat.NewNodeHost(nodeConfig)
	if err != nil {
		return nil, err
	}
	if err := raftNode.StartCluster(peers, *join, smFactoryFunc, raftConfig); err != nil {
		return nil, err
	}

	return raftNode, nil
}
