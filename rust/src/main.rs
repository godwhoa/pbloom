use std::{error::Error, io::{Cursor, Read}};

use murmur3::murmur3_x64_128;
use hex_literal::hex;
use sha2::{Sha256, Sha512, Digest};

struct Filter {
    bits: Vec<u8>,
    hash_count: u8,
}

fn hash(item: &[u8]) -> (u64, u64) {
    murmur3_x64_128(&mut Cursor::new(item), 0)
        .map(|hash| {
            let high = (hash >> 64) as u64;
            let low = hash as u64;
            (low, high)
        })
        .unwrap()
}


impl Filter {
    pub fn new(size: usize, hash_count: u8) -> Self {
        Filter {
            bits: vec![0; size],
            hash_count,
        }
    }

    pub fn new_from_entries_and_fp(entries: usize, fp_rate: f64) -> Result<Self, &'static str> {
        if entries == 0 {
            return Err("number of entries must be positive");
        }
        if fp_rate <= 0.0 || fp_rate >= 1.0 {
            return Err("false positive rate must be between 0 and 1");
        }

        // Calculate m: number of bits
        let m = -((entries as f64) * fp_rate.ln()) / (2.0f64.ln().powi(2));
        // Round m up to the nearest multiple of 8
        let m = (m / 8.0).ceil() * 8.0;
        let size = (m / 8.0) as usize;

        // Calculate k: number of hash functions
        let k = ((m / entries as f64) * 2.0f64.ln()).round() as u8;

        Ok(Filter {
            bits: vec![0; size],
            hash_count: k,
        })
    }

    pub fn from_serialized(serialized: &[u8]) -> Result<Self, rmp::decode::ValueReadError> {
        let mut reader = Cursor::new(serialized);

        let bits_len = rmp::decode::read_bin_len(&mut reader)?;
        let mut bits: Vec<u8> = vec![0; bits_len as usize];
        let _ = reader.read_exact(&mut bits);
        
        let hash_count = rmp::decode::read_u8(&mut reader)?;
        
        Ok(Filter { bits, hash_count })
    }

    pub fn add(&mut self, item: &[u8]) {
        let m: usize = self.bits.len() * 8;
        let (h1, h2) = hash(item);
        for i in 0..self.hash_count as u64 {
            let index = h1.wrapping_add(i.wrapping_mul(h2)) as usize % m;
            self.bits[index / 8] |= 1 << (index % 8);
        }
    }

    pub fn contains(&self, item: &[u8]) -> bool {
        let m: usize = self.bits.len() * 8;
        let (h1, h2) = hash(item);
        for i in 0..self.hash_count as u64 {
            let index = h1.wrapping_add(i.wrapping_mul(h2)) as usize % m;
            if self.bits[index / 8] & (1 << (index % 8)) == 0 {
                return false;
            }
        }
        true
    }

    pub fn serialize(&self) -> Result<Vec<u8>, rmp::encode::ValueWriteError> {
        let mut buf = Vec::new();
        rmp::encode::write_bin(&mut buf, &self.bits)?;
        rmp::encode::write_u8(&mut buf, self.hash_count)?;
        Ok(buf)
    }
}

// tests

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_new_from_entries_and_fp() {
        let cases = vec![
            ("entries=1000, fp=0.01", 1000, 0.01, 1199, 7),
            ("entries=500, fp=0.001", 500, 0.05, 390, 4),
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
        filter.add(b"hello");
        filter.add(b"world");
        filter.add(b"foo");
        filter.add(b"bar");

        assert_eq!(filter.contains(b"hello"), true);
        assert_eq!(filter.contains(b"world"), true);
        assert_eq!(filter.contains(b"foo"), true);
        assert_eq!(filter.contains(b"bar"), true);
        assert_eq!(filter.contains(b"baz"), false);
        assert_eq!(filter.contains(b"qux"), false);
    }

    #[test]
    fn test_serialize() {
        let mut filter = Filter::new(1000, 7);
        filter.add(b"hello");
        filter.add(b"world");
        filter.add(b"foo");
        filter.add(b"bar");

        let serialized = filter.serialize().unwrap();

        let defilter = Filter::from_serialized(&serialized).unwrap();

        assert_eq!(defilter.contains(b"hello"), true);
        assert_eq!(defilter.contains(b"world"), true);
        assert_eq!(defilter.contains(b"foo"), true);
        assert_eq!(defilter.contains(b"bar"), true);
        assert_eq!(defilter.contains(b"baz"), false);
        assert_eq!(defilter.contains(b"qux"), false);
    }

    #[test]
    fn test_portability() {
        let mut filter = Filter::new(1199, 7);
        let entries = 1000;

        for i in 0..entries {
            let key = i.to_string();
            filter.add(key.as_bytes());
        }

        let serialized = filter.serialize().unwrap();


        let mut hasher = Sha256::new();
        hasher.update(serialized.as_slice());
        let hash = hasher.finalize();
        assert_eq!(hash[..], hex!("b38258a2d43384e9d346f0a18f5f430fe3098fec322c97b6569d0aa1f7de610d")[..]);
    }

    #[test]
    fn test_hash_portability() {
        let s = String::from("hello");
        let b = s.as_bytes();
        assert_eq!(b[..], hex!("68656c6c6f")[..]);

        let (h1, h2) = hash(b);
        assert_eq!(h1, 0xcbd8a7b341bd9b02);
        assert_eq!(h2, 0x5b1e906a48ae1d19);
    }
}
