# http2raft Examples

This folder provides examples how to use http2raft package and build your own Raft-clasters.

Examples available at the moment:

- [incr](http://github.com/decoded/http2raft/examples) - implements atomic counters as Raft cluster available over HTTP

## incr

It is Raft-cluster storage for counters available over HTTP. The path in request acts as key for counter value:
- to get current counter value you will need to do GET-request
- to increment counter value  you will need to do POST request
- if you need to increment counter and get the latest current value as reply to your POST - you will need to add yo our request query string special parameter `return_value=true`
- to delete  counter you will need to do DELETE request

### Building

http2raft is based on [dragonboat](https://github.com/lni/dragonboat) which uses different storage backends for Raft-logs persistence. So 1st thing you will need to do is to install storage. You can choose between `RocksDB` and `LevelDB`. RocksDB is recommended so to install it you will need to follow [Install RocksDB](https://github.com/lni/dragonboat/blob/master/README.md#use-dragonboat): 
```bash
$ cd $HOME/src
$ git clone https://github.com/lni/dragonboat
$ cd $HOME/src/dragonboat
$ make install-rocksdb-ull
$ GO111MODULE=on make dragonboat-test
```

The you will need to build example binary. Run this command in examples folder:
```bash
make incr
```
As result you will get binary `example-incr-server` in current folder.

### Running Raft-cluster
Now you are ready to go. Lets run cluster of 3 nodes.
Initial node to bootstrap cluster:
```bash
./example-incr-server -http_addr=localhost:3001 -raft_cluster_id=1 -raft_node_id=1 -peers=1:localhost:5001,2:localhost:5002,3:localhost:5003 -data_dir=/tmp
```
- `http_addr` is the address to run HTTP-server for this node
- `raft_cluster_id` - should be the same for all nodes in Raft-cluster (unless you run multi-group Raft, this feature is coming soon)
- `raft_node_id` - is integer to uniquely identify this Raft-node
- `peers` is a comma separated list of all nodes in cluster expected to join (including current initial one). The format for list items is `nodeID:node:raftPort`
- `data_dir` - this is where Raft will persist snapshots with data on disk
Join second node to cluster:
```bash
./example-incr-server -http_addr=localhost:3002 -join=true -raft_cluster_id=1 -raft_node_id=2 -raft_addr=localhost:5002 -data_dir=/tmp
```
- note special `-join=true` flag
- `peers` are not provided
- `raft_addr` is mandatory and should match to one from peers-list in 1st command
Join third node to cluster:
```bash
./example-incr-server -http_addr=localhost:3003 -join=true -raft_cluster_id=1 -raft_node_id=3 -raft_addr=localhost:5003 -data_dir=/tmp
```

Now all three node are running with possible quorum. Let's try to access http endpoint to increment some counters:
```bash
curl -X GET http://localhost:3001/abc   
0

# here we ask to return value right after increment by providing return_value=true
curl -X POST http://localhost:3001/abc\?return_value\=true
1

curl -X GET http://localhost:3001/abc                     
1

curl -X POST http://localhost:3001/abc\?return_value\=true
2

# lets delete this counter
curl -X DELETE http://localhost:3001/abc

# now it is gone and zero
curl -X GET http://localhost:3001/abc
0
```
