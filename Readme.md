# Script for talk
## Problem definition
- you need to do smth exactly once in your system
	- log rotation, DB schema migration, give range of unique keys, running a cron job, saving steps as step function analogy, etc
- for some reason there is no specifically dedicated solution in your system that can solve it
- you have etcd (ex: k8s cluster) or(for some reason) you want to install etcd and not other tools

## Aim
- quick overview of etcd API for leader election
- work with transactions


## how many operations etcd can support?
Etcd uses Raft for consensus and leader election.
Let's check latency numbers and see how many operations we can confirm through the consensus

It depends very much on how durable you want to be but if we consider these numbers we can see that theoretical limits will be:

| configuration     | RTT(round trip time) | theoretical limit for confirmed op/s |
| ----------------- | -------------------- | ------------------------------------ |
| inside DC         | 1-2ms                | 1000-500                             |
| inside one region | 2-5ms                | 500-200                              |
| cross region      | 30-200ms             | 35-5                                 |
Even with ideal conditions (no failures, no retries), consensus-based confirmation is fundamentally limited by network latency. Even inside a single data center, the best case is 500-1000 ops/sec

If your load is not high - simply use kv store.
But if you need more you need to batch transactions per read and implement this logic.

## How to get more
- batching
	- batch with etcd tools(individual k-v writes but groupped in one Tx)
	- batch with 1 tx - but in value - write batched/compressed Tx ids(self-implemented mechanism)

# Demo plan

- up the stack
	- `make up`
- check member list
	- `etcdctl --endpoints=http://localhost:2379 member list`
- check etcd leader
	- `docker exec etcd-etcd-00-1 etcdctl endpoint status --endpoints=http://etcd-etcd-00-1:2379,http://etcd-etcd-01-1:2379,http://etcd-etcd-02-1:2379 -w table`
- leader election. Lease
	- check cmd/main.go implementation
	- return after leader "election"
	- check how to take it, how node looses it
	- watch `etcdctl --endpoints=http://localhost:2379 watch /leader` for followers to get reaction where to start
- change lease time - see how watch updates deleted quicker/slower
- start to write each record individually
	- `etcdctl --endpoints=http://localhost:2379 watch --prefix /individual`
	- `etcdctl --endpoints=http://localhost:2379 get --prefix /individual`
	- `etcdctl --endpoints=http://localhost:2379 del --prefix /individual`
- start to write batched etcd records by using etcd api