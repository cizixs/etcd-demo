# etcd v2 API demo

For simplicity, we use `http://127.0.0.1:2379` as etcd server, 
please make sure etcd is running and listening at the right ip and port.

## Run

Before start, please install etcd client library:

```
go get github.com/coreos/etcd/client
```

Then, just run:
```
go run main.go
```

## Demos

Include: 

- set a simple key
- get value of a simple key
- delete a simple key
- list a directory values
- set ttl to key
- atomic change: CAS(compare and set)
- watch changes

Not included(but easy to implement):

- create a directory
- delete a directory
- update ttl of a key/dir
- compare and delete
