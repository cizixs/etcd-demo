package main

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io/ioutil"
	"strings"
	"sync"
	"time"

	etcd "github.com/coreos/etcd/clientv3"
)

const (
	turingPrefix = "/turing-awards/"
)

// EtcdConfig stores important fileds to connect to etcd.
// Mainly endpoints and verification related info
type EtcdConfig struct {
	Endpoints []string
	KeyFile   string
	CertFile  string
	CAFile    string
	Username  string
	Password  string
}

func newEtcdClient(c *EtcdConfig) (*etcd.Client, error) {
	var tlsInfo *tls.Config
	var pool *x509.CertPool

	if c.CertFile != "" && c.KeyFile != "" {
		// load key and cert
		cert, err := tls.LoadX509KeyPair(c.CertFile, c.KeyFile)
		if err != nil {
			return nil, fmt.Errorf("Unable to load cert: %v", err)
		}

		if c.CAFile != "" {
			// load root CA info
			caData, err := ioutil.ReadFile(c.CAFile)
			if err != nil {
				return nil, fmt.Errorf("Unable to load ca file context: %v", caData)
			}
			pool = x509.NewCertPool()
			pool.AppendCertsFromPEM(caData)
		}

		tlsInfo = &tls.Config{
			Certificates: []tls.Certificate{cert},
			RootCAs:      pool,
		}
	}

	client, err := etcd.New(etcd.Config{
		Endpoints: c.Endpoints,
		Username:  c.Username,
		Password:  c.Password,
		TLS:       tlsInfo,
	})
	if err != nil {
		return nil, err
	}

	return client, nil
}

var client *etcd.Client

func init() {
	var err error
	client, err = newEtcdClient(&EtcdConfig{
		Endpoints: []string{"http://127.0.0.1:2379"},
	})

	if err != nil {
		panic(fmt.Sprintf("Unable to create etcd client: %v", err))
	}
}

func setWinner(who, what string) error {
	key := turingPrefix + who
	fmt.Printf("Set %s to [%s]\n", key, what)

	_, err := client.Put(context.Background(), key, what)
	if err != nil {
		fmt.Printf("Put error: %v", err)
	}
	return err
}

func setWinnerWithLease(who, what string, ttl int) error {
	key := turingPrefix + who
	fmt.Printf("Set %s to [%s] with lease %d\n", key, what, ttl)

	// create a lease first
	resp, err := client.Grant(context.Background(), int64(ttl))
	if err != nil {
		fmt.Printf("Unable to create lease")
	}

	// Use WithLease to add lease to key
	_, err = client.Put(context.Background(), key, what, etcd.WithLease(resp.ID))
	if err != nil {
		fmt.Printf("Put error: %v", err)
	}
	return err
}
func deleteWinner(who string) error {
	key := turingPrefix + who
	fmt.Printf("Delete %s \n", key)

	_, err := client.Delete(context.Background(), key)
	if err != nil {
		fmt.Printf("Delete error: %v", err)
	}
	return err
}

func getWinner(who string) error {
	key := turingPrefix + who

	resp, err := client.Get(context.Background(), key)
	if err != nil {
		return err
	}

	if len(resp.Kvs) == 0 {
		fmt.Printf("%s does not win Turing Awards\n", who)
		return nil
	}

	tokens := strings.Split(string(resp.Kvs[0].Key), "/")
	who = tokens[len(tokens)-1]
	what := resp.Kvs[0].Value
	fmt.Printf("%s won turing awards for %s\n", who, what)

	return nil
}

func getWinners() error {
	key := turingPrefix

	resp, err := client.Get(context.Background(), key, etcd.WithPrefix())
	if err != nil {
		return err
	}

	for _, ev := range resp.Kvs {
		fmt.Printf("%s won turing awards for %s\n", ev.Key, ev.Value)
	}
	return nil
}

// updateWinner only updates winner description if you provide
// the correct previous description
func updateWinner(who string, prev, new string) error {
	key := turingPrefix + who
	fmt.Printf("Update %s to [%s]\n", key, new)

	resp, err := client.Txn(context.Background()).
		If(etcd.Compare(etcd.Value(key), "=", prev)).
		Then(etcd.OpPut(key, new)).
		Commit()

	if err != nil {
		fmt.Printf("Set value error: %v", err)
	}

	if !resp.Succeeded {
		fmt.Printf("Set value failed: value compare error\n")
	}

	return err
}

func watchWinners(closeCh <-chan struct{}) {
	key := turingPrefix
	resultCh := client.Watch(context.Background(), key, etcd.WithPrefix())

	for {
		select {
		case resp := <-resultCh:
			for _, event := range resp.Events {
				fmt.Printf("Event received: %s %q: %q\n", event.Type, event.Kv.Key, event.Kv.Value)
			}

		case <-closeCh:
			return
		}
	}
}

func demoGetSet() error {
	who := "john-mccarthy"

	err := setWinner(who, "Artificial Interlligence")
	if err != nil {
		return err
	}

	err = getWinner(who)
	if err != nil {
		return err
	}

	// After delete winner from etcd,
	// we should get nothing
	err = deleteWinner(who)
	if err != nil {
		return err
	}

	err = getWinner(who)
	if err != nil {
		return err
	}

	return nil
}

func demoPrefix() error {
	// add more winnders
	winners := []winner{
		winner{
			"Dijkstra",
			"Programming Languages",
		},
		winner{
			"Knuth",
			"analysis of algorithms",
		},
	}

	for _, w := range winners {
		err := setWinner(w.name, w.what)
		if err != nil {
			return err
		}
	}

	// Get values of a directory
	return getWinners()
}

func demoTransaction() error {
	// add a winner Tim Berners Lee
	who := "Tim-Berners-Lee"
	err := setWinner(who, "WWW")
	if err != nil {
		return err
	}

	// try update winner with more descriptive info
	// returned error is expected, ignore it
	_ = updateWinner(who, "lisp", "Inventing World Wide Web")

	getWinner(who)

	// update with the correct information
	err = updateWinner(who, "WWW", "Inventing World Wide Web")
	if err != nil {
		return err
	}

	// print the updated information
	return getWinner(who)
}

func demoWatch() error {
	closeCh := make(chan struct{})

	// run watch in background
	// print can be out of order
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		watchWinners(closeCh)
		wg.Done()
	}()

	// add winners
	winners := []winner{
		winner{
			"Ken-Thompson",
			"Unix operating system",
		},
		winner{
			"Alan-Key",
			"Object-Oriented Programming Languages",
		},
		winner{
			"cizixs",
			"nothing",
		},
	}

	// wait for watch is ready
	time.Sleep(time.Millisecond * 500)
	for _, w := range winners {
		err := setWinner(w.name, w.what)
		if err != nil {
			return err
		}
	}

	deleteWinner("cizixs")

	closeCh <- struct{}{}
	wg.Wait()

	return nil
}

func demoLease() error {
	// NOTE: the minimial ttl is 5 seconds, see the following explanation:
	// https://github.com/coreos/etcd/issues/6025
	err := setWinnerWithLease("cizixs", "nothing", 5)
	if err != nil {
		return err
	}

	if err := getWinner("cizixs"); err != nil {
		return err
	}

	fmt.Printf("wait for lease to expire...\n")
	time.Sleep(7 * time.Second)

	return getWinner("cizixs")
}

type winner struct {
	name string
	what string
}

type demo struct {
	action      func() error
	description string
}

func main() {
	defer client.Close()
	demos := []demo{
		demo{
			action:      demoGetSet,
			description: "simple set and get",
		},
		demo{
			action:      demoPrefix,
			description: "use prefix to get all values",
		},
		demo{
			action:      demoTransaction,
			description: "multiple action with transaction",
		},
		demo{
			action:      demoWatch,
			description: "watch key changes",
		},
		demo{
			action:      demoLease,
			description: "assign lease to keys",
		},
	}

	for i, d := range demos {
		fmt.Printf("\n------------- DEMO %d: %s --------------\n", i+1, d.description)
		err := d.action()
		if err != nil {
			fmt.Printf("Demo failed: %v", err)
		}
	}
}
