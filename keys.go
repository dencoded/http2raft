package http2raft

import (
	"context"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/lni/dragonboat/v3"
	"github.com/lni/dragonboat/v3/client"
)

// keysController implements endpoints to perform key-value operations
type keysController struct {
	clusterID     uint64
	readTimeOut   time.Duration
	writeTimeOut  time.Duration
	raftNode      *dragonboat.NodeHost
	clientSession *client.Session
}

func (c *keysController) readKey(w http.ResponseWriter, r *http.Request) {
	// perform linear read from cluster
	query := []byte(r.Method + " " + r.URL.Path)
	result, err := c.syncRead(query)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Write(result)
}

func (c *keysController) writeKey(w http.ResponseWriter, r *http.Request) {
	// read payload
	data, err := ioutil.ReadAll(r.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer r.Body.Close()

	// do SyncPropose change key
	query := []byte(r.Method + " " + r.URL.Path)
	if len(data) > 0 {
		query = append(query, '\n')
		query = append(query, data...)
	}

	ctx, cancel := context.WithTimeout(context.Background(), c.writeTimeOut)
	// ignore result as for now
	_, err = c.raftNode.SyncPropose(ctx, c.clientSession, query)
	cancel()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// do sync read of just updated value because proposal based queries are not recommended
	// (SyncPropose) could return new value in result
	if r.Method != http.MethodDelete && r.URL.Query().Get("return_value") != "" {
		query = []byte(http.MethodGet + " " + r.URL.Path)
		result, err := c.syncRead(query)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Write(result)
	}
}

func (c *keysController) syncRead(query []byte) ([]byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), c.readTimeOut)
	defer cancel()
	result, err := c.raftNode.SyncRead(ctx, c.clusterID, query)
	if err != nil {
		return nil, err
	}

	return result.([]byte), nil
}
