# distsql

A minimal and simple implementation of a distributed database written in Go. The underlying storage engine is sqlite as it provides a simple yet robust implementation of an relational database.

## Directories

- `/engine` Contains the implementation for a node's internal storage. In this case, it contains methods to interact with a sqlite connection.
- `/proto` Contains different definitions of protobuf types, such that the types can be transferred using gRPC
- `/consensus` Contains logic for distributed consensus using the raft algorithm.
- `/discovery` Contains logic for keeping track of nodes in a distributed system.
- `/service` A HTTP interface for interacting with a store and a cluster manager
- `/manager` An interface for managing cluster communication and getting API addresses of nodes in the cluster.
- `/coordinator` This package handles logic to combine gRPC and HTTP servers.

## Running tests

Simply run

```
$ make test
```

## API Endpoints

- `/join` Add a given node to the cluster
- `/leave` Remove a given node from the cluster
- `/execute` Execute given statements
- `/query` Query given statement
- `/metric` Get node metrics
