use pgrx::prelude::*;
use pbloom::Filter;

::pgrx::pg_module_magic!();

#[pg_extern]
fn pbloom_contains(filter_column: &[u8], key: &[u8]) -> bool {
    Filter::from_serialized(filter_column)
        .and_then(|filter| filter.contains(key))
        .unwrap_or(false)
}

#[pg_extern]
fn pbloom_add(filter_column: &[u8], key: &[u8]) -> Vec<u8> {
    Filter::from_serialized(filter_column)
        .and_then(|mut filter| {
            let _ = filter.add(key);
            filter.serialize()
        })
        .unwrap_or_default()
}

#[pg_extern]
fn pbloom_create(entries: i32, fp: f64) -> Vec<u8> {
    Filter::new_from_entries_and_fp(entries as usize, fp)
        .unwrap()
        .serialize()
        .unwrap()
}

#[cfg(any(test, feature = "pg_test"))]
#[pg_schema]
mod tests {
    use pgrx::prelude::*;

    #[pg_test]
    fn test_pbloom_check() {
        let mut filter = pbloom::Filter::new_from_entries_and_fp(1000, 0.01).unwrap();
        let _ = filter.add(b"hello");
        let filter_column = filter.serialize().unwrap();
        assert_eq!(crate::pbloom_contains(filter_column.as_slice(), b"hello"), true);
    }

}

/// This module is required by `cargo pgrx test` invocations.
/// It must be visible at the root of your extension crate.
#[cfg(test)]
pub mod pg_test {
    pub fn setup(_options: Vec<&str>) {
        // perform one-off initialization when the pg_test framework starts
    }

    #[must_use]
    pub fn postgresql_conf_options() -> Vec<&'static str> {
        // return any postgresql.conf settings that are required for your tests
        vec![]
    }
}
