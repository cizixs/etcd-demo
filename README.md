# etcd-demo

Code is tested on Linux with `go1.9.2`, go version `1.7.0+` is recommended.

This repo demostrates how to use go etcd client, both v2 api and v3 api. Operations include:

- create a etcd client
- set and get value
- set ttl to a value
- watch changes

**NOTE:** The code is for learning and demostrating purpose only, should not be used in production.

## How to run

Before start, please install etcd client library:

```
go get github.com/coreos/etcd/client
go get github.com/coreos/etcd/clientv3
```
