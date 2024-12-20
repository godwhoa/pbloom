package pbloom

import (
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestNewFilterFromEntriesAndSize tests the NewFilterFromEntriesAndSize constructor.
func TestNewFilterFromEntriesAndSize(t *testing.T) {
	tests := []struct {
		name        string
		entries     int
		size        int
		expectError bool
		expectedK   uint8
		expectedLen int
	}{
		{
			name:        "ValidInput",
			entries:     1000,
			size:        128,
			expectError: false,
			expectedK:   1,
			expectedLen: 128,
		},
		{
			name:        "ZeroEntries",
			entries:     0,
			size:        128,
			expectError: true,
		},
		{
			name:        "NegativeEntries",
			entries:     -10,
			size:        128,
			expectError: true,
		},
		{
			name:        "ZeroSize",
			entries:     1000,
			size:        0,
			expectError: true,
		},
		{
			name:        "NegativeSize",
			entries:     1000,
			size:        -128,
			expectError: true,
		},
		{
			name:        "HigherK",
			entries:     500,
			size:        128,
			expectError: false,
			expectedK:   2,
			expectedLen: 128,
		},
	}

	for _, tt := range tests {
		tt := tt // capture range variable
		t.Run(tt.name, func(t *testing.T) {
			filter, err := NewFilterFromEntriesAndSize(tt.entries, tt.size)
			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, filter)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, filter)
				assert.Equal(t, tt.expectedLen, len(filter.bits))
				assert.Equal(t, tt.expectedK, filter.k)
			}
		})
	}
}

// TestNewFilterFromEntriesAndFP tests the NewFilterFromEntriesAndFP constructor.
func TestNewFilterFromEntriesAndFP(t *testing.T) {
	tests := []struct {
		name         string
		entries      int
		fpRate       float64
		expectError  bool
		expectedK    uint8
		expectedSize int
	}{
		{
			// n = 1000, p = 0.01 -> 9586 bits/7 hashes
			// m = ceil(3118/8)*8 = 9592 bits = 1199B
			// k = round((m / n) * ln(2)) = round(6.6486677559) = 7
			// https://hur.st/bloomfilter/?n=1000&p=&m=9592&k=
			name:         "ValidInput",
			entries:      1000,
			fpRate:       0.01,
			expectError:  false,
			expectedK:    7,
			expectedSize: 1199,
		},
		{
			name:        "ZeroEntries",
			entries:     0,
			fpRate:      0.01,
			expectError: true,
		},
		{
			name:        "NegativeEntries",
			entries:     -100,
			fpRate:      0.01,
			expectError: true,
		},
		{
			name:        "FPRateZero",
			entries:     1000,
			fpRate:      0.0,
			expectError: true,
		},
		{
			name:        "FPRateOne",
			entries:     1000,
			fpRate:      1.0,
			expectError: true,
		},
		{
			name:        "FPRateAboveOne",
			entries:     1000,
			fpRate:      1.5,
			expectError: true,
		},
		{
			name:        "FPRateNegative",
			entries:     1000,
			fpRate:      -0.1,
			expectError: true,
		},
		{
			// n = 500, p = 0.05 -> 3118 bits/4 hashes
			// m = ceil(3118/8)*8 = 3120 bits = 390 bytes
			// k = (m / n) * ln(2) = round(4.3) = 4
			// https://hur.st/bloomfilter/?n=500&p=&m=3120&k=
			name:         "AnotherValidInput",
			entries:      500,
			fpRate:       0.05,
			expectError:  false,
			expectedK:    4,
			expectedSize: 390,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			filter, err := NewFilterFromEntriesAndFP(tt.entries, tt.fpRate)
			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, filter)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, filter)
				assert.Equal(t, tt.expectedSize, len(filter.bits), "Size should match")
				assert.Equal(t, tt.expectedK, filter.k, "K should match")
			}
		})
	}
}

// TestNewFilterFromBits tests the NewFilterFromBits constructor.
func TestNewFilterFromBits(t *testing.T) {
	tests := []struct {
		name        string
		bits        []byte
		k           uint8
		expectError bool
	}{
		{
			name:        "ValidInput",
			bits:        []byte{0xFF, 0x00, 0xAA},
			k:           3,
			expectError: false,
		},
		{
			name:        "EmptyBits",
			bits:        []byte{},
			k:           3,
			expectError: true,
		},
		{
			name:        "ZeroK",
			bits:        []byte{0xFF, 0x00},
			k:           0,
			expectError: true,
		},
		{
			name:        "LargeK",
			bits:        []byte{0xFF, 0xFF, 0xFF},
			k:           100,
			expectError: false,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			filter, err := NewFilterFromBits(tt.bits, tt.k)
			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, filter)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, filter)
				assert.Equal(t, tt.k, filter.k)
				assert.Equal(t, tt.bits, filter.bits)
			}
		})
	}
}

// TestFromSerialized tests the FromSerialized constructor.
func TestFromSerialized(t *testing.T) {
	// Prepare a filter to serialize
	originalFilter := &Filter{
		bits: []byte{0x0F, 0xF0, 0xAA},
		k:    5,
	}

	data, err := originalFilter.Serialize()
	assert.NoError(t, err)
	assert.NotNil(t, data)

	tests := []struct {
		name           string
		data           []byte
		expectError    bool
		expectedFilter *Filter
	}{
		{
			name:           "ValidSerializedData",
			data:           data,
			expectError:    false,
			expectedFilter: originalFilter,
		},
		{
			name:        "EmptyData",
			data:        []byte{},
			expectError: true,
		},
		{
			name:        "InvalidData",
			data:        []byte{0xFF, 0xFF},
			expectError: true,
		},
		{
			name:        "RandomData",
			data:        []byte("random bytes"),
			expectError: true,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			filter, err := FromSerialized(tt.data)
			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, filter)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, filter)
				assert.Equal(t, tt.expectedFilter.k, filter.k)
				assert.Equal(t, tt.expectedFilter.bits, filter.bits)
			}
		})
	}
}

// TestPutExists tests the Put and Exists methods of the Filter.
func TestPutExists(t *testing.T) {
	// Initialize a filter with enough size to avoid false positives for the test
	// For simplicity, use 1KB = 8192 bits, which is ample for small test keys
	filter, err := NewFilterFromEntriesAndSize(100, 128) // 1024 bits
	assert.NoError(t, err)
	assert.NotNil(t, filter)

	// Define keys to insert and check
	insertedKeys := []string{"apple", "banana", "cherry", "date", "elderberry"}

	// Insert keys
	for _, key := range insertedKeys {
		filter.Put([]byte(key))
	}

	// Table-driven test cases for Exists
	tests := []struct {
		name     string
		key      string
		expected bool
	}{
		// Inserted keys should exist
		{"Exists_apple", "apple", true},
		{"Exists_banana", "banana", true},
		{"Exists_cherry", "cherry", true},
		{"Exists_date", "date", true},
		{"Exists_elderberry", "elderberry", true},
		// Non-inserted keys should not exist (with possible false positives)
		{"NotExists_fig", "fig", false},
		{"NotExists_grape", "grape", false},
		{"NotExists_honeydew", "honeydew", false},
		{"NotExists_kiwi", "kiwi", false},
		{"NotExists_lemon", "lemon", false},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			exists := filter.Exists([]byte(tt.key))
			if tt.expected {
				assert.True(t, exists, "Expected key %s to exist", tt.key)
			} else {
				if exists {
					// Due to the nature of Bloom filters, false positives are possible.
					// For this test with sufficient size, we assume no false positives.
					t.Errorf("Expected key %s to not exist, but it exists (false positive)", tt.key)
				} else {
					assert.False(t, exists, "Expected key %s to not exist", tt.key)
				}
			}
		})
	}
}

// TestSerializeAndDeserialize tests the Serialize and FromSerialized methods.
func TestSerializeAndDeserialize(t *testing.T) {
	// Initialize a filter
	filter, err := NewFilterFromEntriesAndFP(1000, 0.01)
	assert.NoError(t, err)
	assert.NotNil(t, filter)

	// Insert some keys
	keys := []string{"alpha", "beta", "gamma", "delta", "epsilon"}
	for _, key := range keys {
		filter.Put([]byte(key))
	}

	// Serialize the filter
	serializedData, err := filter.Serialize()
	assert.NoError(t, err)
	assert.NotNil(t, serializedData)

	// Deserialize the data
	deserializedFilter, err := FromSerialized(serializedData)
	assert.NoError(t, err)
	assert.NotNil(t, deserializedFilter)

	// Compare the original and deserialized filters
	assert.Equal(t, filter.k, deserializedFilter.k, "K values should match")
	assert.Equal(t, filter.bits, deserializedFilter.bits, "Bits slices should match")

	// Verify that the keys still exist in the deserialized filter
	for _, key := range keys {
		assert.True(t, deserializedFilter.Exists([]byte(key)), "Key %s should exist in deserialized filter", key)
	}

	// Verify that non-inserted keys do not exist
	nonKeys := []string{"zeta", "eta", "theta"}
	for _, key := range nonKeys {
		exists := deserializedFilter.Exists([]byte(key))
		if exists {
			t.Errorf("Key %s should not exist in deserialized filter (false positive)", key)
		} else {
			assert.False(t, exists, "Key %s should not exist in deserialized filter", key)
		}
	}
}

// TestFilterBasicOperations tests basic operations like Put and Exists in a simple scenario.
func TestFilterBasicOperations(t *testing.T) {
	filter, err := NewFilterFromEntriesAndSize(10, 16) // 128 bits
	assert.NoError(t, err)
	assert.NotNil(t, filter)

	keys := []string{"one", "two", "three", "four", "five"}

	// Initially, none of the keys should exist
	for _, key := range keys {
		assert.False(t, filter.Exists([]byte(key)), "Key %s should not exist initially", key)
	}

	// Insert "one" and "two"
	filter.Put([]byte("one"))
	filter.Put([]byte("two"))

	// Check existence
	assert.True(t, filter.Exists([]byte("one")), "'one' should exist after insertion")
	assert.True(t, filter.Exists([]byte("two")), "'two' should exist after insertion")
	assert.False(t, filter.Exists([]byte("three")), "'three' should not exist")
	assert.False(t, filter.Exists([]byte("four")), "'four' should not exist")
	assert.False(t, filter.Exists([]byte("five")), "'five' should not exist")

	// Insert "three"
	filter.Put([]byte("three"))
	assert.True(t, filter.Exists([]byte("three")), "'three' should exist after insertion")

	// Check non-inserted keys
	assert.False(t, filter.Exists([]byte("four")), "'four' should not exist")
	assert.False(t, filter.Exists([]byte("five")), "'five' should not exist")

	// Edge case: Insert empty string
	filter.Put([]byte(""))
	assert.True(t, filter.Exists([]byte("")), "Empty string should exist after insertion")
}

// TestMultipleInsertions tests inserting the same key multiple times.
func TestMultipleInsertions(t *testing.T) {
	filter, err := NewFilterFromEntriesAndSize(100, 128) // 1024 bits
	assert.NoError(t, err)
	assert.NotNil(t, filter)

	key := "duplicate"

	// Insert the key multiple times
	for i := 0; i < 10; i++ {
		filter.Put([]byte(key))
	}

	// Check that the key exists
	assert.True(t, filter.Exists([]byte(key)), "Key %s should exist after multiple insertions", key)
}

// TestFalsePositiveRate tests the false positive rate of the Bloom filter.
// Note: This is a probabilistic test and may occasionally fail.
func TestFalsePositiveRate(t *testing.T) {
	entries := 1000
	fpRate := 0.01
	filter, err := NewFilterFromEntriesAndFP(entries, fpRate)
	assert.NoError(t, err)
	assert.NotNil(t, filter)

	// Insert "entries" number of keys
	for i := 0; i < entries; i++ {
		filter.Put(generateKey(i))
	}

	// Test a separate set of keys
	testSize := 10000
	falsePositives := 0
	for i := entries; i < entries+testSize; i++ {
		if filter.Exists(generateKey(i)) {
			falsePositives++
		}
	}

	actualFPRate := float64(falsePositives) / float64(testSize)
	// Allow a margin of error, e.g., 1.5 times the expected FP rate
	assert.LessOrEqual(t, actualFPRate, fpRate*1.5, "False positive rate should be within acceptable bounds")
}

// generateKey generates a unique string key based on an integer.
func generateKey(i int) []byte {
	return []byte("key_" + strconv.Itoa(i))
}
