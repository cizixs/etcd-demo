# etcd-demo

This repo demostrates how to use go etcd client, both v2 api and v3 api.
Operations include:

- create a etcd client
- set and get value
- set ttl to a value
- watch changes

## How to run

Before start, please install etcd client library:

```
go get github.com/coreos/etcd/client
go get github.com/coreos/etcd/clientv3
```
