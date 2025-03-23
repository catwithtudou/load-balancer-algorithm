package loadbalancer

import (
	"sync"
)

// Server 表示一个后端服务器
type Server struct {
	Address string
	Weight  int
	// 用于最小连接算法的当前连接数
	CurrentConnections int32
	// 用于轮询算法的当前权重
	CurrentWeight int
	// 用于轮询算法的有效权重
	EffectiveWeight int
}

// LoadBalancer 定义负载均衡器接口
type LoadBalancer interface {
	// AddServer 添加服务器
	AddServer(server *Server)
	// RemoveServer 移除服务器
	RemoveServer(address string)
	// GetServer 获取下一个服务器
	GetServer(key string) *Server
}

// BaseLoadBalancer 基础负载均衡器结构
type BaseLoadBalancer struct {
	Servers []*Server
	mu      sync.RWMutex
}

// NewBaseLoadBalancer 创建基础负载均衡器
func NewBaseLoadBalancer() *BaseLoadBalancer {
	return &BaseLoadBalancer{
		Servers: make([]*Server, 0),
	}
}

// AddServer 添加服务器
func (b *BaseLoadBalancer) AddServer(server *Server) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.Servers = append(b.Servers, server)
}

// RemoveServer 移除服务器
func (b *BaseLoadBalancer) RemoveServer(address string) {
	b.mu.Lock()
	defer b.mu.Unlock()
	for i, server := range b.Servers {
		if server.Address == address {
			b.Servers = append(b.Servers[:i], b.Servers[i+1:]...)
			break
		}
	}
}

// GetServerCount 获取服务器数量
func (b *BaseLoadBalancer) GetServerCount() int {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return len(b.Servers)
}
