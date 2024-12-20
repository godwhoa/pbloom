package pbloom

import (
	"errors"
	"math"

	"github.com/go-faster/city"
	"github.com/twmb/murmur3"
	"github.com/vmihailenco/msgpack/v5"
)

type Filter struct {
	Bits []byte
	K    uint64
}

// NewFilterFromEntriesAndSize initializes a Bloom filter with a specified number of entries and storage size in bytes.
// It calculates the optimal number of hash functions (K) based on the provided parameters.
func NewFilterFromEntriesAndSize(entries int, size int) (*Filter, error) {
	if entries <= 0 {
		return nil, errors.New("number of entries must be positive")
	}
	if size <= 0 {
		return nil, errors.New("size must be positive")
	}

	m := float64(size * 8) // Total bits
	k := math.Ceil((m / float64(entries)) * math.Log(2))

	return &Filter{
		Bits: make([]byte, size),
		K:    uint64(k),
	}, nil
}

// NewFilterFromEntriesAndFP initializes a Bloom filter with a specified number of entries and desired false positive rate.
// It calculates the necessary size (in bytes) and the optimal number of hash functions (K).
func NewFilterFromEntriesAndFP(entries int, fpRate float64) (*Filter, error) {
	if entries <= 0 {
		return nil, errors.New("number of entries must be positive")
	}
	if fpRate <= 0 || fpRate >= 1 {
		return nil, errors.New("false positive rate must be between 0 and 1")
	}

	// Calculate m: number of bits
	m := -float64(entries) * math.Log(fpRate) / (math.Pow(math.Log(2), 2))
	// Round m up to the nearest multiple of 8
	m = math.Ceil(m/8.0) * 8.0
	size := int(m / 8)

	// Calculate k: number of hash functions
	k := math.Round((m / float64(entries)) * math.Log(2))

	return &Filter{
		Bits: make([]byte, size),
		K:    uint64(k),
	}, nil
}

// NewFilterFromBits initializes a Bloom filter from an existing bit slice and specified number of hash functions.
// This can be useful when deserializing or reconstructing a Bloom filter from stored data.
func NewFilterFromBits(bits []byte, k uint64) (*Filter, error) {
	if len(bits) == 0 {
		return nil, errors.New("bits slice cannot be empty")
	}
	if k == 0 {
		return nil, errors.New("number of hash functions must be positive")
	}

	return &Filter{
		Bits: bits,
		K:    k,
	}, nil
}

// FromSerialized deserializes a Bloom filter from a byte slice using MessagePack.
func FromSerialized(data []byte) (*Filter, error) {
	f := &Filter{}
	if err := msgpack.Unmarshal(data, f); err != nil {
		return nil, err
	}
	return f, nil
}

// Put inserts a key into the Bloom filter by setting the appropriate bits.
func (f *Filter) Put(key string) {
	M := uint64(len(f.Bits) * 8)
	h1 := murmur3.StringSum64(key)
	h2 := city.Hash64([]byte(key))
	for i := uint64(0); i < f.K; i++ {
		hash := (h1 + i*h2) % M
		f.Bits[hash/8] |= 1 << (hash % 8)
	}
}

// Exists checks whether a key is possibly in the Bloom filter.
// Returns true if the key might be in the set, or false if it is definitely not present.
func (f *Filter) Exists(key string) bool {
	M := uint64(len(f.Bits) * 8)
	h1 := murmur3.StringSum64(key)
	h2 := city.Hash64([]byte(key))
	for i := uint64(0); i < f.K; i++ {
		hash := (h1 + i*h2) % M
		if f.Bits[hash/8]&(1<<(hash%8)) == 0 {
			return false
		}
	}
	return true
}

// Serialize serializes the Bloom filter into a byte slice using MessagePack.
func (f *Filter) Serialize() ([]byte, error) {
	return msgpack.Marshal(f)
}
