package shard

import (
	"testing"

	"github.com/google/uuid"
)

func TestForRowKey_Deterministic(t *testing.T) {
	key := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")
	numShards := 64

	first := ForRowKey(key, numShards)
	for i := 0; i < 100; i++ {
		got := ForRowKey(key, numShards)
		if got != first {
			t.Fatalf("iteration %d: got shard %d, want %d", i, got, first)
		}
	}
}

func TestForRowKey_InRange(t *testing.T) {
	shardCounts := []int{1, 2, 4, 8, 16, 32, 64, 128, 256}
	for _, numShards := range shardCounts {
		for i := 0; i < 100; i++ {
			key := uuid.New()
			got := ForRowKey(key, numShards)
			if int(got) < 0 || int(got) >= numShards {
				t.Errorf("numShards=%d key=%s: got shard %d out of range [0,%d)", numShards, key, got, numShards)
			}
		}
	}
}

func TestForRowKey_DifferentKeysDistribute(t *testing.T) {
	numShards := 16
	seen := make(map[ID]bool)

	// Generate enough keys that we expect to see multiple distinct shards
	for i := 0; i < 1000; i++ {
		key := uuid.New()
		s := ForRowKey(key, numShards)
		seen[s] = true
	}

	// With 1000 random keys and 16 shards, we should see most shards hit
	if len(seen) < numShards/2 {
		t.Errorf("poor distribution: only %d/%d shards seen with 1000 keys", len(seen), numShards)
	}
}

func TestForRowKey_SingleShard(t *testing.T) {
	key := uuid.New()
	got := ForRowKey(key, 1)
	if got != 0 {
		t.Errorf("with 1 shard, expected 0 but got %d", got)
	}
}

func TestForRowKey_SameKeyDifferentShardCounts(t *testing.T) {
	key := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")

	s64 := ForRowKey(key, 64)
	s128 := ForRowKey(key, 128)

	// Different shard counts may produce different results — just verify they're valid
	if int(s64) < 0 || int(s64) >= 64 {
		t.Errorf("shard64 out of range: %d", s64)
	}
	if int(s128) < 0 || int(s128) >= 128 {
		t.Errorf("shard128 out of range: %d", s128)
	}
}

func TestForRowKey_NilUUID(t *testing.T) {
	got := ForRowKey(uuid.Nil, 64)
	if int(got) < 0 || int(got) >= 64 {
		t.Errorf("nil UUID: shard %d out of range [0,64)", got)
	}
}

func TestID_Type(t *testing.T) {
	var id ID = 42
	if int(id) != 42 {
		t.Errorf("expected 42, got %d", id)
	}
}

func BenchmarkForRowKey(b *testing.B) {
	key := uuid.New()
	for i := 0; i < b.N; i++ {
		ForRowKey(key, 64)
	}
}
