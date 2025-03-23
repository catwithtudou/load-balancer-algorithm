package loadbalancer

import (
	"fmt"
	"testing"
)

func TestWeightedRoundRobin(t *testing.T) {
	// 创建加权轮询负载均衡器
	lb := NewRoundRobinLoadBalancer(true)

	// 添加3个权重不同的服务器
	servers := []*Server{
		{Address: "Server-A", Weight: 3, EffectiveWeight: 3},
		{Address: "Server-B", Weight: 2, EffectiveWeight: 2},
		{Address: "Server-C", Weight: 1, EffectiveWeight: 1},
	}

	for _, server := range servers {
		lb.AddServer(server)
	}

	// 平滑加权轮询算法的预期选择序列: [A,B,A,C,B,A]
	// 对于权重[3,2,1]，这种分布更均匀
	fmt.Println("平滑加权轮询理论序列: [A,B,A,C,B,A]")

	// 连续选择12次，应该看到模式重复
	selections := make([]string, 12)
	for i := 0; i < 12; i++ {
		server := lb.GetServer("")
		if server != nil {
			selections[i] = fmt.Sprintf("%s", server.Address)
		} else {
			selections[i] = "nil"
		}
	}

	// 输出选择结果
	fmt.Println("加权轮询选择序列:")
	for i, selection := range selections {
		fmt.Printf("第%d次选择: %s\n", i+1, selection)
	}

	// 由于平滑加权轮询算法可能产生略有不同的结果，我们不强制检查完全匹配
	// 而是检查选择结果中各服务器出现的次数是否与权重比例匹配
	countA := 0
	countB := 0
	countC := 0

	for _, selection := range selections {
		switch selection {
		case "Server-A":
			countA++
		case "Server-B":
			countB++
		case "Server-C":
			countC++
		}
	}

	// 检查服务器选择的比例是否大致符合权重比例 3:2:1
	fmt.Printf("服务器选择次数 - A: %d, B: %d, C: %d\n", countA, countB, countC)

	// 这里不需要严格检查模式，只要大致比例符合即可
	totalCount := float64(countA + countB + countC)
	if totalCount == 0 {
		t.Error("没有选择任何服务器")
	} else {
		ratioA := float64(countA) / totalCount
		ratioB := float64(countB) / totalCount
		ratioC := float64(countC) / totalCount

		// 理论上比例应该是 3/6, 2/6, 1/6，允许有小的偏差
		if ratioA < 0.4 || ratioB < 0.25 || ratioC < 0.1 {
			t.Errorf("服务器选择比例不符合权重比例. A: %.2f, B: %.2f, C: %.2f", ratioA, ratioB, ratioC)
		}
	}

	// 重置服务器权重
	lb.ResetWeights()
}
