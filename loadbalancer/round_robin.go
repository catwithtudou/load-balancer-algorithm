package loadbalancer

import (
	"sync/atomic"
)

// RoundRobinLoadBalancer 轮询负载均衡器
type RoundRobinLoadBalancer struct {
	*BaseLoadBalancer
	currentIndex int64
	weighted     bool
}

// NewRoundRobinLoadBalancer 创建轮询负载均衡器
func NewRoundRobinLoadBalancer(weighted bool) *RoundRobinLoadBalancer {
	return &RoundRobinLoadBalancer{
		BaseLoadBalancer: NewBaseLoadBalancer(),
		weighted:         weighted,
	}
}

// GetServer 获取下一个服务器
func (lb *RoundRobinLoadBalancer) GetServer(key string) *Server {
	lb.mu.Lock()
	defer lb.mu.Unlock()

	if len(lb.Servers) == 0 {
		return nil
	}

	if !lb.weighted {
		// 非加权轮询
		index := atomic.AddInt64(&lb.currentIndex, 1) % int64(len(lb.Servers))
		return lb.Servers[index]
	}

	// 调试输出
	/*
		fmt.Println("当前服务器权重状态:")
		for i, server := range lb.Servers {
			fmt.Printf("Server%d[%s] 当前权重:%d, 有效权重:%d\n",
				i+1, server.Address, server.CurrentWeight, server.EffectiveWeight)
		}
	*/

	// 实现平滑加权轮询（Smooth Weighted Round-Robin）
	totalWeight := 0
	var bestServer *Server

	// 计算总有效权重，并为每个服务器增加当前权重
	for _, server := range lb.Servers {
		// 确保有效权重被初始化
		if server.EffectiveWeight == 0 {
			server.EffectiveWeight = server.Weight
		}
		// 当前权重增加有效权重
		server.CurrentWeight += server.EffectiveWeight
		totalWeight += server.EffectiveWeight

		// 选择当前权重最大的服务器
		if bestServer == nil || server.CurrentWeight > bestServer.CurrentWeight {
			bestServer = server
		}
	}

	// 如果找到了最佳服务器，减少其当前权重
	if bestServer != nil {
		bestServer.CurrentWeight -= totalWeight
	}

	return bestServer
}

// ResetWeights 重置所有服务器的权重
func (lb *RoundRobinLoadBalancer) ResetWeights() {
	lb.mu.Lock()
	defer lb.mu.Unlock()

	// 重置所有服务器的权重
	for _, server := range lb.Servers {
		server.CurrentWeight = 0
		server.EffectiveWeight = server.Weight
	}

	// 重置轮询状态
	atomic.StoreInt64(&lb.currentIndex, 0)
}

// AddServer 添加服务器
func (lb *RoundRobinLoadBalancer) AddServer(server *Server) {
	// 只有在未设置的情况下初始化EffectiveWeight
	if server.EffectiveWeight == 0 {
		server.EffectiveWeight = server.Weight
	}
	lb.BaseLoadBalancer.AddServer(server)
}
