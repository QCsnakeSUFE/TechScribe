package main

import (
	"context"
	"fmt"
	"log"
	"reflect"
	"sync"
	"time"

	clientv3 "go.etcd.io/etcd/client/v3"
	"google.golang.org/grpc/resolver"
)

const etcdResolverScheme = "etcd"

type etcdResolverBuilder struct {
	endpoints []string
}

func registerEtcdResolver(endpoints []string) {
	resolver.Register(&etcdResolverBuilder{endpoints: endpoints})
}

func (b *etcdResolverBuilder) Scheme() string {
	return etcdResolverScheme
}

func (b *etcdResolverBuilder) Build(target resolver.Target, cc resolver.ClientConn, opts resolver.BuildOptions) (resolver.Resolver, error) {
	serviceName := target.Endpoint()
	if serviceName == "" {
		return nil, fmt.Errorf("missing etcd resolver service name")
	}
	log.Printf("[Resolver] build service name: %s", serviceName)

	ctx, cancel := context.WithCancel(context.Background())
	cli, err := clientv3.New(clientv3.Config{
		Endpoints:   b.endpoints,
		DialTimeout: 3 * time.Second,
	})
	if err != nil {
		cancel()
		return nil, err
	}

	r := &etcdResolver{
		cc:          cc,
		cli:         cli,
		serviceName: serviceName,
		ctx:         ctx,
		cancel:      cancel,
	}

	if err := r.updateState(ctx); err != nil {
		_ = cli.Close()
		cancel()
		return nil, err
	}

	go r.watch()
	return r, nil
}

type etcdResolver struct {
	cc          resolver.ClientConn
	cli         *clientv3.Client
	serviceName string
	ctx         context.Context
	cancel      context.CancelFunc
	lastAddrs   []string
	mu          sync.Mutex
}

func (r *etcdResolver) ResolveNow(opts resolver.ResolveNowOptions) {
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()

		if err := r.updateState(ctx); err != nil {
			r.cc.ReportError(err)
		}
	}()
}

func (r *etcdResolver) Close() {
	r.cancel()
	_ = r.cli.Close()
}

func (r *etcdResolver) watch() {
	prefix := servicePrefix(r.serviceName)
	ch := r.cli.Watch(r.ctx, prefix, clientv3.WithPrefix())
	for resp := range ch {
		if err := resp.Err(); err != nil {
			log.Printf("[Resolver] watch error: %v", err)
			r.cc.ReportError(err)
			continue
		}

		for _, ev := range resp.Events {
			log.Printf("[Resolver] watch event type=%v key=%s value=%s",
				ev.Type,
				string(ev.Kv.Key),
				string(ev.Kv.Value),
			)
		}

		if err := r.updateState(r.ctx); err != nil {
			r.cc.ReportError(err)
		}
	}
}

func (r *etcdResolver) updateState(ctx context.Context) error {
	addrs, err := r.resolveAddresses(ctx)
	if err != nil {
		return err
	}

	current := addressStrings(addrs)

	r.mu.Lock()
	defer r.mu.Unlock()

	if reflect.DeepEqual(current, r.lastAddrs) {
		return nil
	}

	r.lastAddrs = current

	log.Printf("[Resolver] update addresses=%v", current)
	return r.cc.UpdateState(resolver.State{Addresses: addrs})
}

func (r *etcdResolver) resolveAddresses(ctx context.Context) ([]resolver.Address, error) {
	resp, err := r.cli.Get(ctx, servicePrefix(r.serviceName), clientv3.WithPrefix())
	if err != nil {
		return nil, err
	}
	if len(resp.Kvs) == 0 {
		return nil, fmt.Errorf("no available instance for service %s", r.serviceName)
	}

	addrs := make([]resolver.Address, 0, len(resp.Kvs))
	for _, kv := range resp.Kvs {
		addrs = append(addrs, resolver.Address{Addr: string(kv.Value)})
	}
	return addrs, nil
}

func addressStrings(addrs []resolver.Address) []string {
	result := make([]string, 0, len(addrs))
	for _, addr := range addrs {
		result = append(result, addr.Addr)
	}
	return result
}
