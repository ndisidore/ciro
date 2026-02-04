//! KDL parser for Ciro CI pipelines.

mod error;
mod parser;

pub use error::ParseError;
pub use parser::parse_pipeline;
