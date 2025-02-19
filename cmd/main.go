package main

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"time"

	clientv3 "go.etcd.io/etcd/client/v3"
)

const (
	LeaderNodeID = "node1"
	LeaderKey    = "/leader"
)

func main() {
	etcdEndpoints := []string{
		"http://localhost:2379",
		"http://localhost:2381",
		"http://localhost:2383",
	}

	cli, err := clientv3.New(clientv3.Config{
		Endpoints:   etcdEndpoints,
		DialTimeout: 5 * time.Second,
	})
	if err != nil {
		log.Fatal(err)
	}
	defer cli.Close()

	if _, _, err := campaign(cli); err != nil {
		log.Fatal(err)
	}

	log.Println("got leader position")

	//
	// For leader demo/check
	// return

	// start writing individual txs
	writeIndividualTransactions(cli)

	// batch etcd api
	writeBatchedTransactionsETCDapi(cli)
}

func nextKV(prefix string, seed int64, counter *int) (string, string) {
	val := *counter
	*counter++
	return fmt.Sprintf("%v/%v/[%d]", prefix, seed, val), "value" + strconv.Itoa(val)
}

func writeIndividualTransactions(cli *clientv3.Client) {
	const prefix = "/individual"

	start := time.Now()
	seed := start.Unix()

	var counter int

	for {
		nextK, nextV := nextKV(prefix, seed, &counter)

		if _, err := cli.Txn(context.Background()).If(
			clientv3.Compare(clientv3.Value(LeaderKey), "=", LeaderNodeID),
		).Then(
			clientv3.OpPut(nextK, nextV),
		).Commit(); err != nil {
			log.Fatal(err)
		}

		mod := 100
		if counter < 100 {
			mod = 10
		}

		if counter%mod == 0 {
			log.Printf("wrote %d keys in %v", counter, time.Since(start))
		}

		// time.Sleep(1 * time.Second)
	}
}

func writeBatchedTransactionsETCDapi(cli *clientv3.Client) {
	const prefix = "/etcdBatch"

	// ETCD_MAX_TXN_OPS
	const txPerBatch = 100

	start := time.Now()
	seed := start.Unix()

	var counter int
	for {

		ops := make([]clientv3.Op, 0, txPerBatch)
		for i := 0; i < txPerBatch; i++ {
			nextK, nextV := nextKV(prefix, seed, &counter)
			ops = append(ops, clientv3.OpPut(nextK, nextV))
		}

		if _, err := cli.Txn(context.Background()).
			If(
				clientv3.Compare(clientv3.Value(LeaderKey), "=", LeaderNodeID),
			).
			Then(ops...).
			Commit(); err != nil {
			log.Fatal(err)
		}

		if counter%100 == 0 {
			log.Printf("wrote %d keys in %v", counter, time.Since(start))
		}

		// time.Sleep(1 * time.Second)
	}
}

func campaign(cli *clientv3.Client) (clientv3.LeaseID, chan struct{}, error) {

	lease, err := cli.Grant(context.Background(), 3)
	if err != nil {
		return 0, nil, fmt.Errorf("failed to create lease: %v", err)
	}

	resp, err := cli.Txn(context.Background()).
		If(clientv3.Compare(clientv3.Version(LeaderKey), "=", 0)).
		Then(clientv3.OpPut(LeaderKey, LeaderNodeID, clientv3.WithLease(lease.ID))).
		Else(clientv3.OpGet(LeaderKey)).
		Commit()
	if err != nil {
		return 0, nil, fmt.Errorf("failed to execute leader transaction: %v", err)
	}

	if !resp.Succeeded {
		currentLeader := string(resp.Responses[0].GetResponseRange().Kvs[0].Value)
		return 0, nil, fmt.Errorf("failed to become leader, current leader is: %s", currentLeader)
	}

	keepAliveCh, err := cli.KeepAlive(context.Background(), lease.ID)
	if err != nil {
		return 0, nil, fmt.Errorf("failed to keep lease alive: %v", err)
	}

	leaderCh := make(chan struct{})
	go func() {
		for {
			if _, ok := <-keepAliveCh; !ok {
				close(leaderCh)
				return
			}
		}
	}()

	return lease.ID, leaderCh, nil
}
