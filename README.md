# http2raft - Raft based clusters exposed as HTTP server

http2raft is a simple framework to build fault-tolerant key-value clusters controlled by Raft consensus and available over HTTP. It allows to implement easily Raft-cluster with custom implementation of your own state machine. Raft will "feed" your state machine with committed log entries and your implementation is responsible to process these entries.  

## Implementation
The idea is simple - http2raft runs HTTP server and Raft-node at the same time. All `GET` requests to HTTP server turn into reads of your Raft-cluster state. All `POST`, `PUT`, `PATCH` and `DELETE` HTTP requests will turn in to change state of your Raft-cluster.

Please have a look at [Examples](http://github.com/decoded/http2raft/examples)

NOTE: http2raft is an experiment and IS NOT supposed to be used in production!
