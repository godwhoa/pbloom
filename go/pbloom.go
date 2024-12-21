package pbloom

import (
	"bytes"
	"errors"
	"math"

	"github.com/spaolacci/murmur3"
	"github.com/vmihailenco/msgpack/v5"
)

type Filter struct {
	bits   []byte
	k      uint8
	hasher murmur3.Hash128
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
		bits:   make([]byte, size),
		k:      uint8(k),
		hasher: murmur3.New128(),
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
		bits:   make([]byte, size),
		k:      uint8(k),
		hasher: murmur3.New128(),
	}, nil
}

// NewFilterFromBits initializes a Bloom filter from an existing bit slice and specified number of hash functions.
// This can be useful when deserializing or reconstructing a Bloom filter from stored data.
func NewFilterFromBits(bits []byte, k uint8) (*Filter, error) {
	if len(bits) == 0 {
		return nil, errors.New("bits slice cannot be empty")
	}
	if k == 0 {
		return nil, errors.New("number of hash functions must be positive")
	}

	return &Filter{
		bits:   bits,
		k:      k,
		hasher: murmur3.New128(),
	}, nil
}

// FromSerialized deserializes a Bloom filter from a byte slice using MessagePack.
func FromSerialized(data []byte) (*Filter, error) {
	dec := msgpack.NewDecoder(bytes.NewReader(data))
	bits, err := dec.DecodeBytes()
	if err != nil {
		return nil, err
	}
	k, err := dec.DecodeUint8()
	if err != nil {
		return nil, err
	}
	return &Filter{
		bits:   bits,
		k:      k,
		hasher: murmur3.New128(),
	}, nil
}

// Put inserts a key into the Bloom filter by setting the appropriate bits.
func (f *Filter) Put(key []byte) {
	M := uint64(len(f.bits) * 8)
	f.hasher.Reset()
	f.hasher.Write(key)
	h1, h2 := f.hasher.Sum128()
	for i := uint64(0); i < uint64(f.k); i++ {
		hash := (h1 + i*h2) % M
		f.bits[hash/8] |= 1 << (hash % 8)
	}
}

// Exists checks whether a key is possibly in the Bloom filter.
// Returns true if the key might be in the set, or false if it is definitely not present.
func (f *Filter) Exists(key []byte) bool {
	M := uint64(len(f.bits) * 8)
	f.hasher.Reset()
	f.hasher.Write(key)
	h1, h2 := f.hasher.Sum128()
	for i := uint64(0); i < uint64(f.k); i++ {
		hash := (h1 + i*h2) % M
		if f.bits[hash/8]&(1<<(hash%8)) == 0 {
			return false
		}
	}
	return true
}

// Serialize serializes the Bloom filter into a byte slice using MessagePack.
func (f *Filter) Serialize() ([]byte, error) {
	encoded := bytes.Buffer{}
	enc := msgpack.NewEncoder(&encoded)
	if err := enc.EncodeBytes(f.bits); err != nil {
		return nil, err
	}
	if err := enc.EncodeUint8(f.k); err != nil {
		return nil, err
	}
	return encoded.Bytes(), nil
}
