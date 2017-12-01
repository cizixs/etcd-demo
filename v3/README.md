# etcd v3 API demo

For simplicity, we use `http://127.0.0.1:2379` as etcd server, 
please make sure etcd is running and listening at the right ip and port.

## Run

Before start, please install etcd client library:

```
go get github.com/coreos/etcd/clientv3
```

## Demos

Include: 

- set a simple key
- get value of a simple key
- delete a simple key
- use prefix to list dir-like values
- set ttl to key
- transcation commands
- watch changes

Not included(but easy to implement):

- lease
  - revoke a lease
  - keepalive a lease
  - get lease information
- get history value of a key
- get list of keys with sort and pagination
