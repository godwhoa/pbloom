use std::io::{Cursor, Read};

use murmur3::murmur3_x64_128;
use rmp::{decode, encode};

/// A Bloom filter implementation.
pub struct Filter {
    bits: Vec<u8>,
    hash_count: u8,
}

/// Errors that can occur when creating a `Filter` from serialized data.
#[derive(Debug)]
pub enum FilterError {
    DecodeError(decode::ValueReadError),
    EncodeError(encode::ValueWriteError),
    IOError(std::io::Error),
}

impl From<decode::ValueReadError> for FilterError {
    fn from(err: decode::ValueReadError) -> Self {
        FilterError::DecodeError(err)
    }
}

impl From<std::io::Error> for FilterError {
    fn from(err: std::io::Error) -> Self {
        FilterError::IOError(err)
    }
}

impl From<encode::ValueWriteError> for FilterError {
    fn from(err: encode::ValueWriteError) -> Self {
        FilterError::EncodeError(err)
    }
}

impl Filter {
    /// Creates a new `Filter` with the specified size in bytes and number of hash functions.
    pub fn new(size: usize, hash_count: u8) -> Self {
        Self {
            bits: vec![0; size],
            hash_count,
        }
    }

    /// Creates a new `Filter` based on the number of entries and desired false positive rate.
    pub fn new_from_entries_and_fp(entries: usize, fp_rate: f64) -> Result<Self, &'static str> {
        if entries == 0 {
            return Err("Number of entries must be positive");
        }
        if !(0.0..1.0).contains(&fp_rate) {
            return Err("False positive rate must be between 0 and 1");
        }

        // Calculate m: number of bits
        let m = -(entries as f64 * fp_rate.ln()) / (2.0_f64.ln().powi(2));
        // Round m up to the nearest multiple of 8
        let m = (m / 8.0).ceil() * 8.0;
        let size = m as usize / 8;

        // Calculate k: number of hash functions
        let k = ((m / entries as f64) * 2.0_f64.ln()).round() as u8;

        Ok(Self {
            bits: vec![0; size],
            hash_count: k,
        })
    }

    /// Deserializes a `Filter` from a byte slice.
    pub fn from_serialized(serialized: &[u8]) -> Result<Self, FilterError> {
        let mut reader = Cursor::new(serialized);

        let bits_len = decode::read_bin_len(&mut reader)?;
        let mut bits = vec![0u8; bits_len as usize];
        reader.read_exact(&mut bits)?;

        let hash_count = decode::read_u8(&mut reader)?;

        Ok(Self { bits, hash_count })
    }

    /// Computes two 64-bit hashes for the given item using Murmur3.
    fn hash(item: &[u8]) -> Result<(u64, u64), std::io::Error> {
        let hash = murmur3_x64_128(&mut Cursor::new(item), 0)?;
        Ok(((hash & 0xFFFF_FFFF_FFFF_FFFF) as u64, (hash >> 64) as u64))
    }

    /// Adds an item to the filter.
    pub fn add(&mut self, item: &[u8]) -> Result<(), FilterError> {
        let m = (self.bits.len() * 8) as u64;
        let (h1, h2) = Self::hash(item)?;

        for i in 0..self.hash_count as u64 {
            let index = (h1.wrapping_add(i.wrapping_mul(h2)) % m as u64) as usize;
            self.bits[index / 8] |= 1 << (index % 8);
        }
        Ok(())
    }

    /// Checks if an item is present in the filter.
    pub fn contains(&self, item: &[u8]) -> Result<bool, FilterError> {
        let m = (self.bits.len() * 8) as u64;
        let (h1, h2) = Self::hash(item)?;

        Ok((0..self.hash_count as u64).all(|i| {
            let index = (h1.wrapping_add(i.wrapping_mul(h2)) % m) as usize;
            self.bits[index / 8] & (1 << (index % 8)) != 0
        }))
    }

    /// Serializes the filter into a byte vector.
    pub fn serialize(&self) -> Result<Vec<u8>, FilterError> {
        let mut buf = Vec::with_capacity(self.bits.len() + 1);
        encode::write_bin(&mut buf, &self.bits)?;
        encode::write_u8(&mut buf, self.hash_count)?;
        Ok(buf)
    }
}

#[cfg(test)]
mod tests {

    use hex_literal::hex;
    use sha2::{Digest, Sha256};

    use super::*;

    #[test]
    fn test_new_from_entries_and_fp() {
        let cases = vec![
            ("entries=1000, fp=0.01", 1000, 0.01, 1199, 7),
            ("entries=500, fp=0.05", 500, 0.05, 390, 4),
        ];

        for (title, entries, fp_rate, expected_bits_len, expected_hash_count) in cases {
            let filter = Filter::new_from_entries_and_fp(entries, fp_rate).unwrap();
            assert_eq!(filter.bits.len(), expected_bits_len, "{}", title);
            assert_eq!(filter.hash_count, expected_hash_count, "{}", title);
        }
    }

    #[test]
    fn test_filter() {
        let mut filter = Filter::new(1000, 7);
        filter.add(b"hello").unwrap();
        filter.add(b"world").unwrap();
        filter.add(b"foo").unwrap();
        filter.add(b"bar").unwrap();

        assert_eq!(filter.contains(b"hello").unwrap(), true);
        assert_eq!(filter.contains(b"world").unwrap(), true);
        assert_eq!(filter.contains(b"foo").unwrap(), true);
        assert_eq!(filter.contains(b"bar").unwrap(), true);
        assert_eq!(filter.contains(b"baz").unwrap(), false);
        assert_eq!(filter.contains(b"qux").unwrap(), false);
    }

    #[test]
    fn test_serialize() {
        let mut filter = Filter::new(1000, 7);
        filter.add(b"hello").unwrap();
        filter.add(b"world").unwrap();
        filter.add(b"foo").unwrap();
        filter.add(b"bar").unwrap();

        let serialized = filter.serialize().unwrap();
        let defilter = Filter::from_serialized(&serialized).unwrap();

        assert_eq!(defilter.contains(b"hello").unwrap(), true);
        assert_eq!(defilter.contains(b"world").unwrap(), true);
        assert_eq!(defilter.contains(b"foo").unwrap(), true);
        assert_eq!(defilter.contains(b"bar").unwrap(), true);
        assert_eq!(defilter.contains(b"baz").unwrap(), false);
        assert_eq!(defilter.contains(b"qux").unwrap(), false);
    }

    #[test]
    fn test_portability() {
        let mut filter = Filter::new(1199, 7);
        let entries = 1000;

        for i in 0..entries {
            let key = i.to_string();
            filter.add(key.as_bytes()).unwrap();
        }

        let serialized = filter.serialize().unwrap();

        let mut hasher = Sha256::new();
        hasher.update(&serialized);
        let hash = hasher.finalize();

        assert_eq!(
            hash[..],
            hex!("b38258a2d43384e9d346f0a18f5f430fe3098fec322c97b6569d0aa1f7de610d")[..]
        );
    }

    #[test]
    fn test_hash_portability() {
        let s = "hello";
        let b = s.as_bytes();
        assert_eq!(b, hex!("68656c6c6f"));

        // Since hash returns a Result, we need to handle it
        let (h1, h2) = Filter::hash(b).unwrap();
        assert_eq!(h1, 0xcbd8a7b341bd9b02);
        assert_eq!(h2, 0x5b1e906a48ae1d19);
    }
}
