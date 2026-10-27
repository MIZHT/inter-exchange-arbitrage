package websocket_manager

import (
	"fmt"
	"sync"
)

type ProxyPool struct {
	mu      sync.Mutex
	proxies []string
	cursor  int
}

func NewProxyPool(proxies []string) *ProxyPool {
	return &ProxyPool{proxies: append([]string(nil), proxies...)}
}

func PoolOrProxy(proxy string, pools ...*ProxyPool) *ProxyPool {
	if len(pools) > 0 && pools[0] != nil {
		return pools[0]
	}
	return NewProxyPool([]string{proxy})
}

func (p *ProxyPool) Next() string {
	p.mu.Lock()
	defer p.mu.Unlock()
	if len(p.proxies) == 0 {
		return ""
	}

	proxy := p.proxies[p.cursor]
	p.cursor = (p.cursor + 1) % len(p.proxies)
	return proxy
}

type Group struct {
	ID    string
	Proxy string
	Coins map[string]struct{}
}

type GroupPlanner struct {
	mu        sync.Mutex
	pool      *ProxyPool
	prefix    string
	capacity  int
	nextID    uint64
	groups    []*Group
	coinGroup map[string]*Group
}

func NewGroupPlanner(pool *ProxyPool, exchange, market string, capacity int) *GroupPlanner {
	if capacity < 1 {
		capacity = 1
	}
	return &GroupPlanner{
		pool:      pool,
		prefix:    exchange + ":" + market,
		capacity:  capacity,
		coinGroup: make(map[string]*Group),
	}
}

func (p *GroupPlanner) Add(coins []string) {
	p.mu.Lock()
	defer p.mu.Unlock()

	var newCoins []string
	for _, coin := range coins {
		if _, exists := p.coinGroup[coin]; !exists {
			newCoins = append(newCoins, coin)
		}
	}
	for len(newCoins) > 0 {
		n := p.capacity
		if n > len(newCoins) {
			n = len(newCoins)
		}
		group := &Group{
			ID:    fmt.Sprintf("%s:%d", p.prefix, p.nextID),
			Proxy: p.pool.Next(),
			Coins: make(map[string]struct{}, n),
		}
		p.nextID++
		batch := newCoins[:n]
		for _, coin := range batch {
			group.Coins[coin] = struct{}{}
			p.coinGroup[coin] = group
		}
		p.groups = append(p.groups, group)
		newCoins = newCoins[n:]
	}
}

func (p *GroupPlanner) Remove(coins []string) {
	p.mu.Lock()
	defer p.mu.Unlock()

	for _, coin := range coins {
		group, exists := p.coinGroup[coin]
		if !exists {
			continue
		}
		delete(group.Coins, coin)
		delete(p.coinGroup, coin)
	}
}

func (p *GroupPlanner) Set(coins []string) {
	desired := make(map[string]struct{}, len(coins))
	for _, coin := range coins {
		desired[coin] = struct{}{}
	}
	p.mu.Lock()
	var add, remove []string
	for _, coin := range coins {
		if _, exists := p.coinGroup[coin]; !exists {
			add = append(add, coin)
		}
	}
	for coin := range p.coinGroup {
		if _, exists := desired[coin]; !exists {
			remove = append(remove, coin)
		}
	}
	p.mu.Unlock()
	p.Remove(remove)
	p.Add(add)
}

func (p *GroupPlanner) All() []Group {
	p.mu.Lock()
	defer p.mu.Unlock()
	result := make([]Group, 0, len(p.groups))
	for _, group := range p.groups {
		result = append(result, cloneGroup(group))
	}
	return result
}

func cloneGroup(group *Group) Group {
	copy := Group{ID: group.ID, Proxy: group.Proxy, Coins: make(map[string]struct{}, len(group.Coins))}
	for coin := range group.Coins {
		copy.Coins[coin] = struct{}{}
	}
	return copy
}
