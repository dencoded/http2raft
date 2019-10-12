package main

import (
	"encoding/gob"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"
	"sync"

	"github.com/lni/dragonboat/v3/statemachine"
)

var (
	errInvalidQuerySyntax = errors.New("invalid query syntax")
)

// InMemCounter implements IStateMachine from github.com/lni/dragonboat/v3/statemachine
type InMemCounter struct {
	data      map[string]uint64
	mutex     sync.RWMutex
	keyPrefix string
}

func NewInMemCounter(clusterID, nodeID uint64) statemachine.IStateMachine {
	return &InMemCounter{
		data:      make(map[string]uint64),
		keyPrefix: fmt.Sprintf("%d:%d:", clusterID, nodeID),
	}
}

func (s *InMemCounter) Lookup(query interface{}) (interface{}, error) {
	// read query in format "GET key"
	// query is always []byte so it is safe to cast without check
	queryStr := query.([]byte)
	queryParts := strings.Split(string(queryStr), " ")
	if len(queryParts) < 2 || queryParts[0] != "GET" {
		return nil, errInvalidQuerySyntax
	}
	key := queryParts[1]

	// read map and return result as []byte
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	result := strconv.FormatUint(s.data[s.keyPrefix+key], 10)
	return []byte(result), nil
}

// Update increments counter using the specified key in committed raft entry
func (s *InMemCounter) Update(data []byte) (statemachine.Result, error) {
	// read query in format "VERB key\nBODY" (VERN can be POST/PUT/PATCH/DELETE BODY is optional)
	// here our counter state machine implementation ignores body as it doesn't need it for INCR operation
	dataParts := strings.Split(string(data), "\n")
	if len(dataParts) == 0 {
		return statemachine.Result{}, errInvalidQuerySyntax
	}
	headerParts := strings.Split(dataParts[0], " ")
	if len(headerParts) < 2 {
		return statemachine.Result{}, errInvalidQuerySyntax
	}

	key := s.keyPrefix + headerParts[1]
	s.mutex.Lock()
	defer s.mutex.Unlock()
	switch headerParts[0] {
	case "DELETE":
		// delete counter operation
		delete(s.data, key)
	default:
		// increment counter operation
		s.data[key] = s.data[key] + 1
	}

	// do not return new value as proposal based queries are not recommended
	return statemachine.Result{}, nil
}

// SaveSnapshot saves the current IStateMachine state into a snapshot
func (s *InMemCounter) SaveSnapshot(w io.Writer, fc statemachine.ISnapshotFileCollection, done <-chan struct{}) error {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	enc := gob.NewEncoder(w)
	if err := enc.Encode(s.data); err != nil {
		return err
	}

	return nil
}

// RecoverFromSnapshot recovers the state using the provided snapshot
func (s *InMemCounter) RecoverFromSnapshot(r io.Reader, files []statemachine.SnapshotFile, done <-chan struct{}) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	dec := gob.NewDecoder(r)
	if err := dec.Decode(&s.data); err != nil {
		return err
	}

	return nil
}

// Close closes the IStateMachine instance
func (s *InMemCounter) Close() error { return nil }
