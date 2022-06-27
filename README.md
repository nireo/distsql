# distsql

This is a fairly simple implementation of a distributed relational database. Some parts of the projects take inspiration from `rqlite`, since the topic of distributed relational databases is quite complicated.

## Directories

* `/engine` Contains the implementation for a node's internal storage. In this case, it contains methods to interact with a sqlite connection.
* `/proto` Contains different definitions of protobuf types, such that the types can be transferred using gRPC
* `/server` Contains logic for how a given node accepts gRPC connections
* `/consensus` Contains logic for distributed consensus using the raft algorithm.
* `/discovery` Contains logic for keeping track of nodes in a distributed system.

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
