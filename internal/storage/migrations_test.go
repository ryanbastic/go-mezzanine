package storage

import "testing"

func TestShardTable(t *testing.T) {
	tests := []struct {
		shardID int
		want    string
	}{
		{0, "cells_0000"},
		{1, "cells_0001"},
		{42, "cells_0042"},
		{99, "cells_0099"},
		{100, "cells_0100"},
		{999, "cells_0999"},
		{1000, "cells_1000"},
		{9999, "cells_9999"},
	}

	for _, tt := range tests {
		got := ShardTable(tt.shardID)
		if got != tt.want {
			t.Errorf("ShardTable(%d) = %q, want %q", tt.shardID, got, tt.want)
		}
	}
}

func TestShardTable_ZeroPadding(t *testing.T) {
	// Verify 4-digit zero padding
	got := ShardTable(5)
	if len(got) != len("cells_0005") {
		t.Errorf("unexpected length: %q", got)
	}
	if got != "cells_0005" {
		t.Errorf("got %q, want %q", got, "cells_0005")
	}
}
