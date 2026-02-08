package shard

import (
	"hash/fnv"

	"github.com/google/uuid"
)

// ID represents a shard number in [0, NumShards).
type ID int

// ForRowKey computes the shard for a given row_key UUID.
func ForRowKey(rowKey uuid.UUID, numShards int) ID {
	h := fnv.New32a()
	b := [16]byte(rowKey)
	h.Write(b[:])
	return ID(int(h.Sum32()) % numShards)
}

// ForKey computes the shard for an arbitrary string key.
func ForKey(key string, numShards int) ID {
	h := fnv.New32a()
	h.Write([]byte(key))
	return ID(int(h.Sum32()) % numShards)
}
