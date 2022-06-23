# distsql

This is a fairly simple implementation of a distributed relational database. Some parts of the projects take inspiration from `rqlite`, since the topic of distributed relational databases is quite complicated.

## Directories

* `/engine` Contains the implementation for a node's internal storage. In this case, it contains methods to interact with a sqlite connection.
* `/proto` Contains different definitions of protobuf types, such that the types can be transferred using gRPC
* `/server` Contains logic for how a given node accepts gRPC connections
