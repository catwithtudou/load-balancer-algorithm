package main

import (
	"fmt"
	"time"

	"github.com/load-balancer-algorithm/loadbalancer"
)

func main() {
	// 创建测试服务器，使用不同的权重
	servers := []*loadbalancer.Server{
		{Address: "192.168.1.1:8080", Weight: 1}, // 权重1
		{Address: "192.168.1.2:8080", Weight: 2}, // 权重2
		{Address: "192.168.1.3:8080", Weight: 3}, // 权重3
	}

	// 测试随机选择算法
	fmt.Println("测试随机选择算法:")
	randomLB := loadbalancer.NewRandomLoadBalancer()
	for _, server := range servers {
		randomLB.AddServer(server)
	}
	for i := 0; i < 5; i++ {
		server := randomLB.GetServer("")
		fmt.Printf("第%d次选择: %s (权重: %d)\n", i+1, server.Address, server.Weight)
	}

	// 测试轮询算法
	fmt.Println("\n测试轮询算法:")
	roundRobinLB := loadbalancer.NewRoundRobinLoadBalancer(false) // 非加权轮询
	for _, server := range servers {
		roundRobinLB.AddServer(server)
	}
	for i := 0; i < 5; i++ {
		server := roundRobinLB.GetServer("")
		fmt.Printf("第%d次选择: %s (权重: %d)\n", i+1, server.Address, server.Weight)
	}

	// 测试加权轮询算法
	fmt.Println("\n测试加权轮询算法:")
	weightedRoundRobinLB := loadbalancer.NewRoundRobinLoadBalancer(true) // 加权轮询

	// 创建新的服务器数组，确保CurrentWeight和EffectiveWeight正确初始化
	wrrServers := []*loadbalancer.Server{
		{Address: "192.168.1.3:8080", Weight: 3, EffectiveWeight: 3, CurrentWeight: 0}, // 权重3, 服务器A
		{Address: "192.168.1.2:8080", Weight: 2, EffectiveWeight: 2, CurrentWeight: 0}, // 权重2, 服务器B
		{Address: "192.168.1.1:8080", Weight: 1, EffectiveWeight: 1, CurrentWeight: 0}, // 权重1, 服务器C
	}

	for _, server := range wrrServers {
		weightedRoundRobinLB.AddServer(server)
	}

	fmt.Println("预期序列: [3权重, 2权重, 3权重, 1权重, 2权重, 3权重] - 平滑加权轮询")
	for i := 0; i < 10; i++ {
		server := weightedRoundRobinLB.GetServer("")
		fmt.Printf("第%d次选择: %s (权重: %d, 当前权重: %d)\n", i+1, server.Address, server.Weight, server.CurrentWeight)
	}
	weightedRoundRobinLB.ResetWeights() // 重置权重

	// 测试最小连接算法
	fmt.Println("\n测试最小连接算法:")
	leastConnLB := loadbalancer.NewLeastConnectionsLoadBalancer(false) // 非加权最小连接
	for _, server := range servers {
		leastConnLB.AddServer(server)
	}

	// 先创建一些连接
	serversToRelease := make([]*loadbalancer.Server, 0)
	// 给权重为1的服务器添加1个连接
	s1 := leastConnLB.GetServer("")
	serversToRelease = append(serversToRelease, s1)
	fmt.Printf("预先添加连接: %s (权重: %d, 当前连接数: %d)\n", s1.Address, s1.Weight, s1.CurrentConnections)

	// 给权重为2的服务器添加2个连接
	for i := 0; i < 2; i++ {
		s2 := leastConnLB.GetServer("")
		serversToRelease = append(serversToRelease, s2)
		fmt.Printf("预先添加连接: %s (权重: %d, 当前连接数: %d)\n", s2.Address, s2.Weight, s2.CurrentConnections)
	}

	// 现在测试算法选择
	fmt.Println("\n开始测试最小连接算法选择过程:")
	for i := 0; i < 5; i++ {
		server := leastConnLB.GetServer("")
		fmt.Printf("第%d次选择: %s (权重: %d, 当前连接数: %d)\n", i+1, server.Address, server.Weight, server.CurrentConnections)
		// 每次选择后延迟释放连接
		go func(s *loadbalancer.Server) {
			time.Sleep(500 * time.Millisecond)
			leastConnLB.ReleaseConnection(s)
		}(server)
		time.Sleep(100 * time.Millisecond) // 等待一点时间以便观察选择效果
	}

	// 释放之前添加的连接
	for _, s := range serversToRelease {
		leastConnLB.ReleaseConnection(s)
	}

	time.Sleep(1 * time.Second) // 等待所有连接释放完成

	// 测试加权最小连接算法
	fmt.Println("\n测试加权最小连接算法:")
	weightedLeastConnLB := loadbalancer.NewLeastConnectionsLoadBalancer(true) // 加权最小连接
	for _, server := range servers {
		weightedLeastConnLB.AddServer(server)
	}

	// 先创建一些连接，使每个服务器的连接数/权重比不同
	// 权重1的服务器添加5个连接（5/1=5）
	s1WLC := weightedLeastConnLB.GetServer("")
	for i := 0; i < 4; i++ { // 已经有1个连接，再添加4个
		weightedLeastConnLB.GetServer("")
	}
	fmt.Printf("预先设置: %s (权重: %d, 当前连接数: %d, 比值: %.2f)\n",
		s1WLC.Address, s1WLC.Weight, s1WLC.CurrentConnections, float64(s1WLC.CurrentConnections)/float64(s1WLC.Weight))

	// 权重2的服务器添加8个连接（8/2=4）
	s2WLC := weightedLeastConnLB.GetServer("")
	for i := 0; i < 7; i++ { // 已经有1个连接，再添加7个
		weightedLeastConnLB.GetServer("")
	}
	fmt.Printf("预先设置: %s (权重: %d, 当前连接数: %d, 比值: %.2f)\n",
		s2WLC.Address, s2WLC.Weight, s2WLC.CurrentConnections, float64(s2WLC.CurrentConnections)/float64(s2WLC.Weight))

	// 权重3的服务器添加9个连接（9/3=3）- 应该是最小的比值
	s3WLC := weightedLeastConnLB.GetServer("")
	for i := 0; i < 8; i++ { // 已经有1个连接，再添加8个
		weightedLeastConnLB.GetServer("")
	}
	fmt.Printf("预先设置: %s (权重: %d, 当前连接数: %d, 比值: %.2f)\n",
		s3WLC.Address, s3WLC.Weight, s3WLC.CurrentConnections, float64(s3WLC.CurrentConnections)/float64(s3WLC.Weight))

	fmt.Println("\n开始测试加权最小连接算法选择过程:")
	for i := 0; i < 5; i++ {
		server := weightedLeastConnLB.GetServer("")
		fmt.Printf("第%d次选择: %s (权重: %d, 当前连接数: %d, 比值: %.2f)\n",
			i+1, server.Address, server.Weight, server.CurrentConnections,
			float64(server.CurrentConnections)/float64(server.Weight))

		// 每次选择后延迟释放连接
		go func(s *loadbalancer.Server) {
			time.Sleep(500 * time.Millisecond)
			weightedLeastConnLB.ReleaseConnection(s)
		}(server)
		time.Sleep(100 * time.Millisecond) // 等待一点时间以便观察选择效果
	}

	time.Sleep(1 * time.Second) // 等待所有连接释放完成

	// 测试 Maglev 一致性哈希算法
	fmt.Println("\n测试 Maglev 一致性哈希算法:")
	maglevHashLB := loadbalancer.NewMaglevHashLoadBalancer()
	for _, server := range servers {
		maglevHashLB.AddServer(server)
	}

	// 使用更多样的测试键
	testKeys := []string{
		"user1", "user2", "user3", "user4", "user5",
		"customer10", "customer20", "customer30", "customer40", "customer50",
		"product100", "product200", "product300", "product400", "product500",
		"order1000", "order2000", "order3000", "order4000", "order5000",
		"item10000", "item20000", "item30000", "item40000", "item50000",
	}

	// 服务器选择计数
	serverCount := make(map[string]int)

	fmt.Println("各键映射到的服务器:")
	for _, key := range testKeys {
		server := maglevHashLB.GetServer(key)
		if server != nil {
			fmt.Printf("Key: %-12s 选择的服务器: %s (权重: %d)\n", key, server.Address, server.Weight)
			serverCount[server.Address]++
		} else {
			fmt.Printf("Key: %-12s 无可用服务器\n", key)
		}
	}

	// 展示分布统计
	fmt.Println("\n服务器负载分布统计:")
	totalKeys := len(testKeys)
	for _, server := range servers {
		count := serverCount[server.Address]
		percentage := float64(count) / float64(totalKeys) * 100
		fmt.Printf("服务器: %-16s 权重: %d  分配请求数: %d  百分比: %.2f%%  期望百分比: %.2f%%\n",
			server.Address, server.Weight, count, percentage,
			float64(server.Weight)/float64(1+2+3)*100) // 1+2+3是所有服务器权重之和
	}

	// 在另一个场景测试：删除一个服务器，然后再测试键分布
	fmt.Println("\n移除服务器后的一致性哈希测试:")
	// 移除中间权重的服务器
	maglevHashLB.RemoveServer(servers[1]) // 移除权重为2的服务器

	// 记录移除服务器前的映射，检查变化
	fmt.Println("移除服务器后的键映射变化:")
	changedCount := 0
	newServerCount := make(map[string]int)

	for _, key := range testKeys {
		server := maglevHashLB.GetServer(key)
		if server != nil {
			newServerCount[server.Address]++
			if serverCount[server.Address] != newServerCount[server.Address] {
				changedCount++
			}
		}
	}

	// 展示新的分布统计
	fmt.Println("\n移除服务器后的负载分布:")
	for i, server := range servers {
		if i == 1 { // 跳过已移除的服务器
			continue
		}
		newCount := newServerCount[server.Address]
		newPercentage := float64(newCount) / float64(totalKeys) * 100
		remainingWeight := 1 + 3 // 剩余的权重总和
		fmt.Printf("服务器: %-16s 权重: %d  分配请求数: %d  百分比: %.2f%%  期望百分比: %.2f%%\n",
			server.Address, server.Weight, newCount, newPercentage,
			float64(server.Weight)/float64(remainingWeight)*100)
	}

	// 输出变化率
	changePercentage := float64(changedCount) / float64(totalKeys) * 100
	fmt.Printf("\n移除服务器后，%.2f%% 的请求被重新映射到其他服务器\n", changePercentage)
}
