package tests

import (
	"testing"
	"time"

	"github.com/iamgilwell/aura/internal/ai"
	"github.com/iamgilwell/aura/internal/monitor"
)

func TestProcessSignature(t *testing.T) {
	proc1 := &monitor.ProcessInfo{
		Name:     "firefox",
		User:     "testuser",
		CPU:      23.0, // buckets to 20
		Memory:   12.0, // buckets to 10
		Category: monitor.CategoryUser,
	}

	proc2 := &monitor.ProcessInfo{
		Name:     "firefox",
		User:     "testuser",
		CPU:      24.0, // also buckets to 20
		Memory:   13.0, // also buckets to 10
		Category: monitor.CategoryUser,
	}

	proc3 := &monitor.ProcessInfo{
		Name:     "chrome",
		User:     "testuser",
		CPU:      23.0,
		Memory:   12.0,
		Category: monitor.CategoryUser,
	}

	sig1 := ai.ProcessSignature(proc1)
	sig2 := ai.ProcessSignature(proc2)
	sig3 := ai.ProcessSignature(proc3)

	if sig1 != sig2 {
		t.Error("similar processes should have same signature")
	}
	if sig1 == sig3 {
		t.Error("different processes should have different signatures")
	}
}

func TestCache(t *testing.T) {
	cache := ai.NewCache(3, 5*time.Minute)

	resp := &ai.DecisionResponse{
		ProcessPID:  100,
		ProcessName: "test",
		Action:      ai.ActionKeep,
		Confidence:  0.8,
	}

	// Put and Get
	cache.Put("key1", resp)
	got, ok := cache.Get("key1")
	if !ok {
		t.Fatal("expected cache hit")
	}
	if !got.FromCache {
		t.Error("cached response should have FromCache=true")
	}
	if got.ProcessPID != 100 {
		t.Errorf("wrong PID: got %d, want 100", got.ProcessPID)
	}

	// Cache miss
	_, ok = cache.Get("nonexistent")
	if ok {
		t.Error("expected cache miss")
	}

	// Eviction
	cache.Put("key2", resp)
	cache.Put("key3", resp)
	cache.Put("key4", resp) // should evict key1

	_, ok = cache.Get("key1")
	if ok {
		t.Error("key1 should have been evicted")
	}

	if cache.Size() != 3 {
		t.Errorf("expected cache size 3, got %d", cache.Size())
	}

	// Clear
	cache.Clear()
	if cache.Size() != 0 {
		t.Errorf("expected empty cache after clear, got %d", cache.Size())
	}
}

func TestCacheTTL(t *testing.T) {
	cache := ai.NewCache(10, 50*time.Millisecond)

	resp := &ai.DecisionResponse{
		ProcessPID: 100,
		Action:     ai.ActionKeep,
	}

	cache.Put("key1", resp)

	// Should exist immediately
	_, ok := cache.Get("key1")
	if !ok {
		t.Fatal("expected cache hit")
	}

	// Wait for TTL expiry
	time.Sleep(100 * time.Millisecond)

	_, ok = cache.Get("key1")
	if ok {
		t.Error("expected cache miss after TTL expiry")
	}
}

func TestActionConstants(t *testing.T) {
	if ai.ActionTerminate != "terminate" {
		t.Error("ActionTerminate wrong")
	}
	if ai.ActionKeep != "keep" {
		t.Error("ActionKeep wrong")
	}
	if ai.ActionNotify != "notify" {
		t.Error("ActionNotify wrong")
	}
}
