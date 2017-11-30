package main

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	etcd "github.com/coreos/etcd/client"
	"github.com/coreos/etcd/pkg/transport"
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

// newEtcdClient creates and returns etcdClient
func newEtcdClient(c *EtcdConfig) (etcd.KeysAPI, error) {
	tlsInfo := transport.TLSInfo{
		CertFile: c.CertFile,
		KeyFile:  c.KeyFile,
		CAFile:   c.CAFile,
	}

	t, err := transport.NewTransport(tlsInfo, time.Second)
	if err != nil {
		return nil, err
	}

	client, err := etcd.New(etcd.Config{
		Endpoints: c.Endpoints,
		Transport: t,
		Username:  c.Username,
		Password:  c.Password,
	})
	if err != nil {
		return nil, err
	}

	return etcd.NewKeysAPI(client), nil
}

var client etcd.KeysAPI

func init() {
	var err error
	client, err = newEtcdClient(&EtcdConfig{
		Endpoints: []string{"http://127.0.0.1:2379"},
	})

	if err != nil {
		panic(fmt.Sprintf("Unable to create etcd client: %v", err))
	}
}

func setWinner(who, what string, ttl int) error {
	key := turingPrefix + who
	fmt.Printf("Set %s to [%s]\n", key, what)
	_, err := client.Set(context.Background(), key, what,
		&etcd.SetOptions{
			TTL: time.Duration(ttl) * time.Second,
		},
	)

	if err != nil {
		fmt.Printf("Set value error: %v", err)
		return err
	}
	return nil
}

func getWinner(who string) error {
	key := turingPrefix + who

	resp, err := client.Get(context.Background(), key, nil)
	if err != nil {
		if etcd.IsKeyNotFound(err) {
			fmt.Printf("%s does not win Turing Awards\n", who)
			return nil
		}
		return err
	}

	tokens := strings.Split(resp.Node.Key, "/")
	who = tokens[len(tokens)-1]
	what := resp.Node.Value
	fmt.Printf("%s won turing awards for %s\n", who, what)
	return nil
}

func getWinners() error {
	key := turingPrefix

	resp, err := client.Get(context.Background(), key, nil)
	if err != nil {
		if etcd.IsKeyNotFound(err) {
			fmt.Printf("There is no data yet.")
			return nil
		}
		return err
	}

	for _, node := range resp.Node.Nodes {
		tokens := strings.Split(node.Key, "/")
		who := tokens[len(tokens)-1]
		what := node.Value
		fmt.Printf("%s won turing awards for %s\n", who, what)
	}

	return nil
}

// updateWinner only updates winner description if you provide
// the correct previous description
func updateWinner(who string, prev, new string) error {
	key := turingPrefix + who
	fmt.Printf("Update %s to [%s]\n", key, new)
	_, err := client.Set(context.Background(), key, new,
		&etcd.SetOptions{
			PrevValue: prev,
		},
	)

	if err != nil {
		fmt.Printf("Set value error: %v", err)
		return err
	}
	return nil
}

func deleteWinner(who string) error {
	key := "/turing-awards/" + who
	fmt.Printf("Delete %s\n", key)

	_, err := client.Delete(context.Background(), key, nil)
	if err != nil {
		fmt.Printf("Delete value error: %v", err)
		return err
	}
	return nil
}

// watchWinners will watch winner create, update and delete
// num is how many results so we can exit, we don't want to run
// forever
func watchWinners(num int) error {
	key := turingPrefix
	fmt.Printf("Watch all winners\n")

	watcher := client.Watcher(key, &etcd.WatcherOptions{
		Recursive: true,
	})

	count := 0
	for {
		resp, err := watcher.Next(context.Background())
		if err != nil {
			fmt.Printf("watch error: %v", err)
			return err
		}
		fmt.Printf("%s at %s, modified index: %d\n", resp.Action, resp.Node.Key, resp.Node.ModifiedIndex)

		count++
		if count >= num {
			return nil
		}
	}
}

func deleteAllWinners() {
	key := turingPrefix
	_, err := client.Delete(context.Background(), key, &etcd.DeleteOptions{
		Recursive: true,
		Dir:       true,
	})
	if err != nil {
		fmt.Printf("delete error: %v", err)
	}
}

func demoGetSet() {
	err := setWinner("john-mccarthy", "Artificial Interlligence", 0)
	if err != nil {
		return
	}

	err = getWinner("john-mccarthy")
	if err != nil {
		return
	}

	err = deleteWinner("john-mccarthy")
	if err != nil {
		return
	}

	err = getWinner("john-mccarthy")
	if err != nil {
		return
	}
}

func demoTTL() {
	// Say to flatter my vanity, I want to "win" the Turing Awards,
	// but only temporarily
	err := setWinner("cizixs", "nothing", 1)
	if err != nil {
		return
	}

	// Get the key immediately, it'll be there
	err = getWinner("cizixs")
	if err != nil {
		return
	}

	// After 3 seconds, the key will be gone
	time.Sleep(2 * time.Second)

	err = getWinner("cizixs")
	if err != nil {
		return
	}
}

type winner struct {
	name string
	what string
}

func demoDir() {
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
		err := setWinner(w.name, w.what, 0)
		if err != nil {
			return
		}
	}

	// Get values of a directory
	if err := getWinners(); err != nil {
		return
	}
}

func demoCAS() {
	// add a winner Tim Berners Lee
	who := "Tim-Berners-Lee"
	err := setWinner(who, "WWW", 0)
	if err != nil {
		return
	}

	// try update winner with more descriptive info
	// returned error is expected, ignore it
	_ = updateWinner(who, "lisp", "Inventing World Wide Web")

	// update with the correct information
	err = updateWinner(who, "WWW", "Inventing World Wide Web")
	if err != nil {
		return
	}

	// print the updated information
	getWinner(who)
}

func demoWatch() {
	// run watch in background
	// notice print result can be out of order
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		watchWinners(2)
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
	}

	// wait for watch is ready
	time.Sleep(time.Millisecond * 500)
	for _, w := range winners {
		err := setWinner(w.name, w.what, 0)
		if err != nil {
			return
		}
	}

	wg.Wait()
}

type demo struct {
	action      func()
	description string
}

func main() {
	demos := []demo{
		demo{
			action:      demoGetSet,
			description: "simple set and get",
		},
		demo{
			action:      demoDir,
			description: "get directory values",
		},
		demo{
			action:      demoCAS,
			description: "compare and set ",
		},
		demo{
			action:      demoWatch,
			description: "watch winners",
		},
		demo{
			action:      demoTTL,
			description: "set ttl to a key",
		},
	}

	for i, d := range demos {
		fmt.Printf("\n------------- DEMO %d: %s --------------\n", i+1, d.description)
		d.action()
	}

	deleteAllWinners()
}
