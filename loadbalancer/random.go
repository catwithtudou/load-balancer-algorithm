package loadbalancer

import (
	"math/rand"
)

// RandomLoadBalancer 随机选择负载均衡器
type RandomLoadBalancer struct {
	*BaseLoadBalancer
	rng *rand.Rand
}

// NewRandomLoadBalancer 创建随机选择负载均衡器
func NewRandomLoadBalancer() *RandomLoadBalancer {
	return &RandomLoadBalancer{
		BaseLoadBalancer: NewBaseLoadBalancer(),
		rng:              rand.New(rand.NewSource(rand.Int63())),
	}
}

// GetServer 随机选择一个服务器
func (r *RandomLoadBalancer) GetServer(key string) *Server {
	r.mu.RLock()
	defer r.mu.RUnlock()

	// 过滤出可用的服务器
	availableServers := make([]*Server, 0)
	for _, server := range r.Servers {
		if server.Weight > 0 {
			availableServers = append(availableServers, server)
		}
	}

	if len(availableServers) == 0 {
		return nil
	}

	// 计算总权重
	totalWeight := 0
	for _, server := range availableServers {
		totalWeight += server.Weight
	}

	// 随机选择一个服务器（考虑权重）
	randomWeight := r.rng.Intn(totalWeight)
	currentWeight := 0
	for _, server := range availableServers {
		currentWeight += server.Weight
		if randomWeight < currentWeight {
			return server
		}
	}

	// 如果因为浮点数精度问题没有选中任何服务器，返回最后一个
	return availableServers[len(availableServers)-1]
}
