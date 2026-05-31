package main

import (
	"context"
	"fmt"
	"time"

	clientv3 "go.etcd.io/etcd/client/v3"
)

func servicePrefix(serviceName string) string {
	return fmt.Sprintf("/services/%s/", serviceName)
}

func discoverService(ctx context.Context, endpoints []string, serviceName string) (string, error) {
	cli, err := clientv3.New(clientv3.Config{
		Endpoints:   endpoints,
		DialTimeout: 3 * time.Second,
	})
	if err != nil {
		return "", err
	}
	defer cli.Close()

	resp, err := cli.Get(ctx, servicePrefix(serviceName), clientv3.WithPrefix())
	if err != nil {
		return "", err
	}

	if len(resp.Kvs) == 0 {
		return "", fmt.Errorf("no available instance for service %s", serviceName)
	}

	return string(resp.Kvs[0].Value), nil
}
