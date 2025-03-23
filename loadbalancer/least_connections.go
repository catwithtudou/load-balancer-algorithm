package loadbalancer

import (
	"sync"
	"sync/atomic"
)

// LeastConnectionsLoadBalancer 最小连接负载均衡器
type LeastConnectionsLoadBalancer struct {
	*BaseLoadBalancer
	connections map[*Server]*int64
	weighted    bool
	mu          sync.RWMutex
}

// NewLeastConnectionsLoadBalancer 创建最小连接负载均衡器
func NewLeastConnectionsLoadBalancer(weighted bool) *LeastConnectionsLoadBalancer {
	return &LeastConnectionsLoadBalancer{
		BaseLoadBalancer: NewBaseLoadBalancer(),
		connections:      make(map[*Server]*int64),
		weighted:         weighted,
	}
}

// GetServer 获取连接数最少的服务器
func (lb *LeastConnectionsLoadBalancer) GetServer(key string) *Server {
	lb.mu.Lock()
	defer lb.mu.Unlock()

	// 过滤出可用的服务器 - 检查Weight大于0的服务器（表示可用）
	availableServers := make([]*Server, 0)
	for _, server := range lb.Servers {
		if server.Weight > 0 { // 使用Weight > 0作为可用性判断
			availableServers = append(availableServers, server)
		}
	}

	if len(availableServers) == 0 {
		return nil
	}

	// 找到连接数最少的服务器
	var selectedServer *Server
	minValue := float64(1<<63 - 1)

	for _, server := range availableServers {
		connPtr := lb.connections[server]
		if connPtr == nil {
			var conn int64
			connPtr = &conn
			lb.connections[server] = connPtr
		}
		connections := atomic.LoadInt64(connPtr)

		var currentValue float64
		if lb.weighted {
			// 加权最小连接：考虑权重因素
			if server.Weight > 0 {
				// 权重越大，加权值越小，越容易被选中
				currentValue = float64(connections) / float64(server.Weight)
			} else {
				// 如果权重为0，则使用最大值，确保不会被选中
				currentValue = float64(1<<63 - 1)
			}
		} else {
			// 非加权最小连接
			currentValue = float64(connections)
		}

		// 如果当前值小于最小值，选择此服务器
		if currentValue < minValue {
			minValue = currentValue
			selectedServer = server
		} else if currentValue == minValue && selectedServer != nil {
			// 如果加权值相同，优先选择权重更高的服务器
			if server.Weight > selectedServer.Weight {
				selectedServer = server
			}
		}
	}

	if selectedServer != nil {
		// 增加选中服务器的连接数
		atomic.AddInt64(lb.connections[selectedServer], 1)
		// 同时更新Server结构体中的CurrentConnections字段，方便外部查看
		atomic.AddInt32(&selectedServer.CurrentConnections, 1)
	}

	return selectedServer
}

// ReleaseConnection 释放连接
func (lb *LeastConnectionsLoadBalancer) ReleaseConnection(server *Server) {
	if server == nil {
		return
	}

	lb.mu.Lock()
	defer lb.mu.Unlock()

	if connPtr, exists := lb.connections[server]; exists && atomic.LoadInt64(connPtr) > 0 {
		atomic.AddInt64(connPtr, -1)
		// 同时更新Server结构体中的CurrentConnections字段
		if server.CurrentConnections > 0 {
			atomic.AddInt32(&server.CurrentConnections, -1)
		}
	}
}

// AddServer 添加服务器
func (lb *LeastConnectionsLoadBalancer) AddServer(server *Server) {
	lb.mu.Lock()
	defer lb.mu.Unlock()
	lb.Servers = append(lb.Servers, server)
	var conn int64
	lb.connections[server] = &conn
}

// RemoveServer 移除服务器
func (lb *LeastConnectionsLoadBalancer) RemoveServer(server *Server) {
	lb.mu.Lock()
	defer lb.mu.Unlock()
	for i, s := range lb.Servers {
		if s == server {
			lb.Servers = append(lb.Servers[:i], lb.Servers[i+1:]...)
			delete(lb.connections, server)
			break
		}
	}
}
