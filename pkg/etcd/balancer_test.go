package etcd

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var testInstances = []ServiceInstance{
	{ServiceName: "svc", InstanceID: "a", Addr: "10.0.0.1:8080", Weight: 10},
	{ServiceName: "svc", InstanceID: "b", Addr: "10.0.0.2:8080", Weight: 20},
	{ServiceName: "svc", InstanceID: "c", Addr: "10.0.0.3:8080", Weight: 70},
}

func TestRoundRobinBalancer(t *testing.T) {
	b := &RoundRobinBalancer{}

	inst0, err := b.Pick(testInstances)
	require.NoError(t, err)
	assert.Equal(t, "10.0.0.1:8080", inst0.Addr)

	inst1, err := b.Pick(testInstances)
	require.NoError(t, err)
	assert.Equal(t, "10.0.0.2:8080", inst1.Addr)

	inst2, err := b.Pick(testInstances)
	require.NoError(t, err)
	assert.Equal(t, "10.0.0.3:8080", inst2.Addr)

	inst3, err := b.Pick(testInstances)
	require.NoError(t, err)
	assert.Equal(t, "10.0.0.1:8080", inst3.Addr)
}

func TestRoundRobinBalancer_Empty(t *testing.T) {
	b := &RoundRobinBalancer{}
	_, err := b.Pick(nil)
	assert.ErrorIs(t, err, ErrNoInstances)
}

func TestRandomBalancer(t *testing.T) {
	b := &RandomBalancer{}

	for range 100 {
		inst, err := b.Pick(testInstances)
		require.NoError(t, err)
		assert.Contains(t, []string{"10.0.0.1:8080", "10.0.0.2:8080", "10.0.0.3:8080"}, inst.Addr)
	}
}

func TestRandomBalancer_Empty(t *testing.T) {
	b := &RandomBalancer{}
	_, err := b.Pick(nil)
	assert.ErrorIs(t, err, ErrNoInstances)
}

func TestWeightedRandomBalancer(t *testing.T) {
	b := &WeightedRandomBalancer{}

	counts := map[string]int{}
	total := 10000
	for range total {
		inst, err := b.Pick(testInstances)
		require.NoError(t, err)
		counts[inst.Addr]++
	}

	// Weight 70 should get ~70% of picks
	ratio := float64(counts["10.0.0.3:8080"]) / float64(total)
	assert.InDelta(t, 0.70, ratio, 0.05)

	// Weight 20 should get ~20%
	ratio = float64(counts["10.0.0.2:8080"]) / float64(total)
	assert.InDelta(t, 0.20, ratio, 0.05)

	// Weight 10 should get ~10%
	ratio = float64(counts["10.0.0.1:8080"]) / float64(total)
	assert.InDelta(t, 0.10, ratio, 0.05)
}

func TestWeightedRandomBalancer_Empty(t *testing.T) {
	b := &WeightedRandomBalancer{}
	_, err := b.Pick(nil)
	assert.ErrorIs(t, err, ErrNoInstances)
}

func TestWeightedRandomBalancer_ZeroWeight(t *testing.T) {
	b := &WeightedRandomBalancer{}
	instances := []ServiceInstance{
		{Addr: "10.0.0.1:8080", Weight: 0},
		{Addr: "10.0.0.2:8080", Weight: 0},
	}
	for range 100 {
		inst, err := b.Pick(instances)
		require.NoError(t, err)
		assert.Contains(t, []string{"10.0.0.1:8080", "10.0.0.2:8080"}, inst.Addr)
	}
}
