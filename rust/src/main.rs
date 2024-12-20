use std::io::Cursor;

use murmur3::murmur3_x64_128;

struct Filter {
    bits: Vec<u8>,
    hash_count: u8,
}

fn hash(item: &[u8]) -> (u64, u64) {
    murmur3_x64_128(&mut Cursor::new(item), 0)
        .map(|hash| {
            let high = (hash >> 64) as u64;
            let low = hash as u64;
            (high, low)
        })
        .unwrap()
}

impl Filter {
    fn new(size: usize, hash_count: u8) -> Self {
        Filter {
            bits: vec![0; size],
            hash_count,
        }
    }

    fn from_serialized(serialized: &[u8]) -> Result<Self, rmp_serde::decode::Error> {
        let (bits, hash_count): (Vec<u8>, u8) = rmp_serde::from_slice(serialized)?;
        Ok(Filter { bits, hash_count })
    }

    fn add(&mut self, item: &[u8]) {
        let m: usize = self.bits.len() * 8;
        let (h1, h2) = hash(item);
        for i in 0..self.hash_count as u64 {
            let index = h1.wrapping_add(i.wrapping_mul(h2)) as usize % m;
            self.bits[index / 8] |= 1 << (index % 8);
        }
    }

    fn contains(&self, item: &[u8]) -> bool {
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

    fn serialize(&self) -> Result<Vec<u8>, rmp_serde::encode::Error> {
        rmp_serde::to_vec(&(self.bits.as_slice(), self.hash_count))
    }
}

fn main() {
    let mut filter = Filter::new(1000, 7);
    filter.add(b"hello");
    filter.add(b"world");
    filter.add(b"foo");
    filter.add(b"bar");

    let serialized = filter.serialize().unwrap();

    let defilter = Filter::from_serialized(&serialized).unwrap();

    println!("{}", defilter.contains(b"hello")); // true
    println!("{}", defilter.contains(b"world")); // true
    println!("{}", defilter.contains(b"foo")); // true
    println!("{}", defilter.contains(b"bar")); // true
    println!("{}", defilter.contains(b"baz")); // false
    println!("{}", defilter.contains(b"qux")); // false
}
