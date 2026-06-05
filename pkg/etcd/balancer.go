package etcd

import (
	"math/rand/v2"
	"sync/atomic"

	"github.com/pkg/errors"
)

var ErrNoInstances = errors.New("no available instances")

type Balancer interface {
	Pick(instances []ServiceInstance) (ServiceInstance, error)
}

type RoundRobinBalancer struct {
	counter atomic.Uint64
}

func (b *RoundRobinBalancer) Pick(instances []ServiceInstance) (ServiceInstance, error) {
	if len(instances) == 0 {
		return ServiceInstance{}, ErrNoInstances
	}
	idx := b.counter.Add(1) - 1
	return instances[idx%uint64(len(instances))], nil
}

type RandomBalancer struct{}

func (b *RandomBalancer) Pick(instances []ServiceInstance) (ServiceInstance, error) {
	if len(instances) == 0 {
		return ServiceInstance{}, ErrNoInstances
	}
	return instances[rand.IntN(len(instances))], nil
}

type WeightedRandomBalancer struct{}

func (b *WeightedRandomBalancer) Pick(instances []ServiceInstance) (ServiceInstance, error) {
	if len(instances) == 0 {
		return ServiceInstance{}, ErrNoInstances
	}
	totalWeight := 0
	for _, inst := range instances {
		w := inst.Weight
		if w <= 0 {
			w = 1
		}
		totalWeight += w
	}
	r := rand.IntN(totalWeight)
	for _, inst := range instances {
		w := inst.Weight
		if w <= 0 {
			w = 1
		}
		r -= w
		if r < 0 {
			return inst, nil
		}
	}
	return instances[len(instances)-1], nil
}
