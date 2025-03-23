package loadbalancer

import (
	"sync"

	"github.com/spaolacci/murmur3"
	"github.com/zeebo/xxh3"
)

const (
	// 查找表大小，应该是质数
	lookupTableSize = 65537
)

// MaglevHashLoadBalancer Maglev一致性哈希负载均衡器
type MaglevHashLoadBalancer struct {
	*BaseLoadBalancer
	lookupTable []int
	tableSize   int
	mu          sync.RWMutex
}

// NewMaglevHashLoadBalancer 创建Maglev一致性哈希负载均衡器
func NewMaglevHashLoadBalancer() *MaglevHashLoadBalancer {
	return &MaglevHashLoadBalancer{
		BaseLoadBalancer: NewBaseLoadBalancer(),
		tableSize:        lookupTableSize, // 使用质数作为表大小
		lookupTable:      make([]int, lookupTableSize),
	}
}

// AddServer 添加服务器
func (lb *MaglevHashLoadBalancer) AddServer(server *Server) {
	lb.mu.Lock()
	defer lb.mu.Unlock()

	// 确保EffectiveWeight初始化
	if server.EffectiveWeight == 0 {
		server.EffectiveWeight = server.Weight
	}

	lb.Servers = append(lb.Servers, server)
	lb.updateLookupTable()
}

// RemoveServer 移除服务器
func (lb *MaglevHashLoadBalancer) RemoveServer(server *Server) {
	lb.mu.Lock()
	defer lb.mu.Unlock()
	for i, s := range lb.Servers {
		if s == server {
			lb.Servers = append(lb.Servers[:i], lb.Servers[i+1:]...)
			break
		}
	}
	lb.updateLookupTable()
}

// permutation 计算服务器在查找表中的位置
func (lb *MaglevHashLoadBalancer) permutation(serverIndex int) []int {
	// 使用服务器地址和索引结合计算哈希值，增加多样性
	server := lb.Servers[serverIndex]
	uniqueKey := server.Address + ":" + string(rune(serverIndex))

	// 优化哈希种子计算
	offset := murmur3.Sum64([]byte(uniqueKey)) % uint64(lb.tableSize)
	skip := xxh3.Hash([]byte(uniqueKey))%uint64(lb.tableSize-1) + 1 // 确保skip至少为1且不超过tableSize

	weight := server.Weight
	if weight <= 0 {
		weight = 1 // 确保至少有权重1，避免除零错误
	}

	// 考虑权重因素，影响排列的生成
	perm := make([]int, lb.tableSize)
	for i := 0; i < lb.tableSize; i++ {
		// 根据权重调整偏移量，权重大的服务器有更多机会被选中
		adjustedOffset := (offset + uint64(i*weight)) % uint64(lb.tableSize)
		perm[i] = int((adjustedOffset + uint64(i)*skip) % uint64(lb.tableSize))
	}
	return perm
}

// updateLookupTable 更新查找表
func (lb *MaglevHashLoadBalancer) updateLookupTable() {
	if len(lb.Servers) == 0 {
		// 如果没有服务器，清空查找表
		for i := range lb.lookupTable {
			lb.lookupTable[i] = -1
		}
		return
	}

	// 筛选可用的服务器
	availableServers := make([]*Server, 0)
	serverIndexMap := make(map[*Server]int)
	weightSum := 0

	for i, server := range lb.Servers {
		if server.Weight > 0 { // 只考虑权重大于0的服务器为可用
			availableServers = append(availableServers, server)
			serverIndexMap[server] = i
			weightSum += server.Weight
		}
	}

	// 如果没有可用服务器，清空查找表
	if len(availableServers) == 0 {
		for i := range lb.lookupTable {
			lb.lookupTable[i] = -1
		}
		return
	}

	// 初始化查找表
	for i := range lb.lookupTable {
		lb.lookupTable[i] = -1
	}

	// 计算每个可用服务器的排列
	perms := make([][]int, len(availableServers))
	for i, server := range availableServers {
		// 使用原始索引计算排列
		origIndex := serverIndexMap[server]
		perms[i] = lb.permutation(origIndex)
	}

	// 填充查找表 - 考虑权重因素
	next := make([]int, len(availableServers))
	filled := 0

	// 确保表至少有75%填满
	for filled < lb.tableSize*3/4 {
		// 找到所有服务器中下一个未使用的位置
		minPos := lb.tableSize
		for i := range availableServers {
			if next[i] < lb.tableSize && perms[i][next[i]] < minPos {
				minPos = perms[i][next[i]]
			}
		}
		if minPos == lb.tableSize {
			break // 所有位置都已填满
		}

		// 按权重比例选择服务器
		selectedIndex := -1

		// 首先尝试按权重选择第一个到达该位置的服务器
		candidates := make([]int, 0)
		for i := range availableServers {
			if next[i] < lb.tableSize && perms[i][next[i]] == minPos {
				candidates = append(candidates, i)
			}
		}

		if len(candidates) > 0 {
			// 如果有多个候选服务器，根据权重选择
			if len(candidates) > 1 {
				// 权重大的服务器优先
				maxWeight := 0
				for _, idx := range candidates {
					if availableServers[idx].Weight > maxWeight {
						maxWeight = availableServers[idx].Weight
						selectedIndex = idx
					}
				}
			} else {
				selectedIndex = candidates[0]
			}

			// 使用原始索引填充查找表
			lb.lookupTable[minPos] = serverIndexMap[availableServers[selectedIndex]]
			next[selectedIndex]++
			filled++
		}
	}

	// 确保表完全填满
	for i := range lb.lookupTable {
		if lb.lookupTable[i] == -1 {
			// 如果某个位置未分配，随机选择一个可用服务器
			// 但倾向于选择权重更高的服务器
			totalWeight := 0
			for _, server := range availableServers {
				totalWeight += server.Weight
			}

			if totalWeight > 0 {
				// 按权重选择
				randomWeight := int(murmur3.Sum32([]byte(string(rune(i))))) % totalWeight
				cumulativeWeight := 0
				for _, server := range availableServers {
					cumulativeWeight += server.Weight
					if randomWeight < cumulativeWeight {
						lb.lookupTable[i] = serverIndexMap[server]
						break
					}
				}
			} else {
				// 权重和为0，直接选择第一个服务器
				lb.lookupTable[i] = serverIndexMap[availableServers[0]]
			}
		}
	}
}

// GetServer 根据key获取服务器
func (lb *MaglevHashLoadBalancer) GetServer(key string) *Server {
	lb.mu.RLock()
	defer lb.mu.RUnlock()

	if len(lb.Servers) == 0 {
		return nil
	}

	// 使用key计算哈希值
	hash := murmur3.Sum64([]byte(key))
	index := int(hash % uint64(lb.tableSize))
	serverIndex := lb.lookupTable[index]

	if serverIndex == -1 || serverIndex >= len(lb.Servers) || lb.Servers[serverIndex].Weight <= 0 {
		// 如果查找表中没有对应的服务器，或者服务器不可用，
		// 尝试查找表中的其他位置
		for offset := 1; offset < 20; offset++ {
			newIndex := (index + offset) % lb.tableSize
			serverIndex = lb.lookupTable[newIndex]
			if serverIndex >= 0 && serverIndex < len(lb.Servers) && lb.Servers[serverIndex].Weight > 0 {
				return lb.Servers[serverIndex]
			}
		}

		// 如果仍未找到，回退到简单哈希
		availableServers := make([]*Server, 0)
		for _, server := range lb.Servers {
			if server.Weight > 0 {
				availableServers = append(availableServers, server)
			}
		}

		if len(availableServers) > 0 {
			return availableServers[int(hash%uint64(len(availableServers)))]
		}
		return nil
	}

	return lb.Servers[serverIndex]
}
