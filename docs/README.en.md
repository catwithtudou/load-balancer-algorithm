# Go Load Balancing Algorithm Implementation

> 20250324: I published a blog post about load balancing algorithms
>
> To help everyone understand, interested readers can check the [blog link](https://zhengyua.cn/new_blog/blog/2025/03/24/深入理解四种经典负载均衡算法/) (in Chinese)
>
>

This project implements several common load balancing algorithms in Go, which can be used for load distribution in distributed systems.

This project references [@Zheaoli's Python implementation](https://github.com/Zheaoli/load-balancer-algorithm) and reimplements and optimizes the algorithms in Go.

## Implemented Algorithms

This project implements the following four load balancing algorithms:

1. Random Selection
2. Round Robin
3. Least Connections
4. Maglev Consistent Hashing

Each algorithm supports both weighted and non-weighted versions to meet load balancing requirements in different scenarios.

## Algorithm Details

### 1. Random Selection Algorithm

Random selection algorithm is one of the simplest load balancing strategies, which randomly selects a server from the available server pool to process requests.

**Features**:
- Simple implementation and easy to understand
- Supports weighted random selection, higher weight means higher probability of being selected
- Filters unavailable (weight of 0) servers

**Applicable Scenarios**:
- Servers with similar performance
- Requests with small processing time differences
- Relatively balanced request distribution

### 2. Round Robin Algorithm

Round Robin algorithm selects servers in sequence, achieving average distribution of requests.

**Features**:
- Selects servers in a cyclical order
- Supports weighted round robin (smooth weighted round robin algorithm), servers with higher weights handle more requests
- Can skip unavailable servers

**Applicable Scenarios**:
- Servers with similar configurations
- Balanced request loads
- Suitable for maintaining balanced loads in long-running systems

### 3. Least Connections Algorithm

Least Connections algorithm selects the server with the fewest active connections, optimizing resource utilization.

**Features**:
- Dynamically tracks the number of connections for each server
- Supports weighted least connections algorithm, considering the ratio of server weight to current connections
- Releases resources after connection completion
- Uses atomic operations to ensure concurrent safety

**Applicable Scenarios**:
- Requests with large processing time differences
- Servers with different processing capabilities
- Scenarios requiring dynamic load balancing

### 4. Maglev Consistent Hashing Algorithm

Maglev consistent hashing algorithm is an efficient consistent hashing algorithm designed by Google for large-scale distributed systems.

**Features**:
- Uses hash tables to implement O(1) lookup performance
- Minimizes request remapping when servers change
- Supports weighted consistent hashing, servers with higher weights handle more requests
- Uses dual hash functions (murmur3 and xxh3) to improve randomness and distribution uniformity

**Applicable Scenarios**:
- Systems requiring session persistence or cache consistency
- Environments with frequent server additions and removals
- Large-scale distributed systems

## Usage Example

```go
package main

import (
    "fmt"
    "github.com/load-balancer-algorithm/loadbalancer"
)

func main() {
    // Create servers
    servers := []*loadbalancer.Server{
        {Address: "192.168.1.1:8080", Weight: 1},
        {Address: "192.168.1.2:8080", Weight: 2},
        {Address: "192.168.1.3:8080", Weight: 3},
    }

    // Create load balancer (using consistent hashing as an example)
    lb := loadbalancer.NewMaglevHashLoadBalancer()

    // Add servers
    for _, server := range servers {
        lb.AddServer(server)
    }

    // Use the load balancer to select servers
    for i := 0; i < 5; i++ {
        key := fmt.Sprintf("user%d", i)
        server := lb.GetServer(key)
        fmt.Printf("Request %s assigned to server: %s\n", key, server.Address)
    }
}
```

## Performance Testing and Comparison

Performance comparison of various algorithms in different scenarios:

| Algorithm | Time Complexity | Space Complexity | Consistency | Load Balancing | Suitable Scenarios |
|------|------------|------------|--------|------------|----------|
| Random Selection | O(1) | O(n) | Low | Medium | Simple systems, short connections |
| Round Robin | O(1) | O(n) | Low | High | Server clusters with similar performance |
| Least Connections | O(n) | O(n) | Medium | High | Requests with large processing time differences |
| Maglev Hashing | O(1) | O(m) | High | Medium | Systems requiring session persistence |

## Project Structure

```
loadbalancer/
  ├── base.go           # Basic structures and interface definitions
  ├── random.go         # Random selection algorithm implementation
  ├── round_robin.go    # Round robin algorithm implementation
  ├── least_connections.go  # Least connections algorithm implementation
  └── consistent_hash.go    # Maglev consistent hashing algorithm implementation
```