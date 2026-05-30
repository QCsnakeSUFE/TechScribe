package main

import (
	"context"
	"fmt"
	"log"
	"time"

	clientv3 "go.etcd.io/etcd/client/v3"
)

func registerService(ctx context.Context, endpoints []string, serviceName, addr string) error {
	cli, err := clientv3.New(clientv3.Config{
		Endpoints:   endpoints,
		DialTimeout: 3 * time.Second,
	})
	if err != nil {
		return err
	}

	lease, err := cli.Grant(ctx, 10)
	if err != nil {
		return err
	}

	key := fmt.Sprintf("/services/%s/%s", serviceName, addr)
	if _, err := cli.Put(ctx, key, addr, clientv3.WithLease(lease.ID)); err != nil {
		return err
	}

	ch, err := cli.KeepAlive(ctx, lease.ID)
	if err != nil {
		return err
	}

	go func() {
		for range ch {

		}
		log.Printf("etcd keepalive stopped for %s", key)
	}()
	log.Printf("registered service in etcd: %s -> %s", key, addr)
	return nil
}
