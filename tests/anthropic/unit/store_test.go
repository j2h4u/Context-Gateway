package unit

// Store Tests - Dual TTL Cache Behavior
//
// Tests the memory store with dual TTL (Time-To-Live) support:
// - Original content: short TTL (5 min default) - for expand_context retrieval
// - Compressed content: long TTL (24h default) - for cache hits
// This allows efficient reuse of compressions while expiring originals quickly.

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/compresr/context-gateway/internal/store"
)

// TestStore_SetAndGet verifies basic set/get operations.
func TestStore_SetAndGet(t *testing.T) {
	st := store.NewMemoryStore(5 * time.Minute)
	key := "test_key"
	value := "test_value"

	st.Set(key, value)
	retrieved, exists := st.Get(key)
	require.True(t, exists)
	assert.Equal(t, value, retrieved)
}

// TestStore_CompressedSetAndGet verifies compressed content operations.
func TestStore_CompressedSetAndGet(t *testing.T) {
	st := store.NewMemoryStore(5 * time.Minute)
	key := "shadow_abc123"
	compressed := "compressed summary"

	st.SetCompressed(key, compressed)
	retrieved, exists := st.GetCompressed(key)
	require.True(t, exists)
	assert.Equal(t, compressed, retrieved)
}

// TestStore_DualTTL verifies separate TTLs for original vs compressed.
func TestStore_DualTTL(t *testing.T) {
	originalTTL := 10 * time.Millisecond
	compressedTTL := 100 * time.Millisecond

	st := store.NewMemoryStoreWithDualTTL(originalTTL, compressedTTL)
	key := "shadow_test"

	// Set both original and compressed
	st.Set(key, "original content")
	st.SetCompressed(key, "compressed content")

	// Both should exist initially
	_, origExists := st.Get(key)
	_, compExists := st.GetCompressed(key)
	assert.True(t, origExists, "original should exist")
	assert.True(t, compExists, "compressed should exist")

	// Wait for original TTL to expire
	time.Sleep(15 * time.Millisecond)

	// Original should be gone, compressed should remain
	_, origExists = st.Get(key)
	_, compExists = st.GetCompressed(key)
	assert.False(t, origExists, "original should expire with short TTL")
	assert.True(t, compExists, "compressed should remain with long TTL")
}

// TestStore_NonexistentKey verifies missing key returns false.
func TestStore_NonexistentKey(t *testing.T) {
	st := store.NewMemoryStore(5 * time.Minute)

	_, exists := st.Get("nonexistent")
	assert.False(t, exists)

	_, exists = st.GetCompressed("nonexistent")
	assert.False(t, exists)
}

// TestStore_Overwrite verifies values can be overwritten.
func TestStore_Overwrite(t *testing.T) {
	st := store.NewMemoryStore(5 * time.Minute)
	key := "shadow_key"

	st.Set(key, "value1")
	st.Set(key, "value2")

	retrieved, exists := st.Get(key)
	require.True(t, exists)
	assert.Equal(t, "value2", retrieved)
}
