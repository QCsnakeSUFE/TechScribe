package main

import (
	"context"
	"fmt"
	"time"

	clientv3 "go.etcd.io/etcd/client/v3"
)

func discoverService(ctx context.Context, endpoints []string, serviceName string) (string, error) {
	cli, err := clientv3.New(clientv3.Config{
		Endpoints:   endpoints,
		DialTimeout: 3 * time.Second,
	})
	if err != nil {
		return "", err
	}
	defer cli.Close()

	prefix := fmt.Sprintf("/services/%s/", serviceName)
	resp, err := cli.Get(ctx, prefix, clientv3.WithPrefix())
	if err != nil {
		return "", err
	}

	if len(resp.Kvs) == 0 {
		return "", fmt.Errorf("no available instance for service %s", serviceName)
	}

	return string(resp.Kvs[0].Value), nil
}
