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

First you need to create test certificates such that authentication tests can run properly.

```
$ make gencert
```

This command creates three certificates, one for the server, one for an authorized client and a third one for an unauthorized client.

After that you just need to run:

```
$ make test
```

Examples of test certificates is stored in the `/test` directory. That directory also contains files for defining actions a given user can access in the `policy.csv` file.

## API Endpoints

- `/join` Add a given node to the cluster
- `/leave` Remove a given node from the cluster
- `/execute` Execute given statements
- `/query` Query given statements
