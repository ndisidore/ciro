//! Parser error types with rich diagnostics.

#![allow(unused_assignments)] // False positives from miette derive macro

use miette::{Diagnostic, NamedSource, SourceSpan};
use thiserror::Error;

/// Errors that can occur during pipeline parsing.
#[derive(Debug, Error, Diagnostic)]
pub enum ParseError {
    /// KDL syntax error - wraps the KDL parser's own diagnostic
    #[error(transparent)]
    #[diagnostic(transparent)]
    Syntax(#[from] kdl::KdlError),

    /// IO error reading file
    #[error("failed to read file: {message}")]
    #[diagnostic(code(ciro::parse::io))]
    Io { message: String },

    /// Missing required node
    #[error("missing required `{node}` node")]
    #[diagnostic(code(ciro::parse::missing_node))]
    MissingNode {
        node: String,
        #[source_code]
        src: NamedSource<String>,
        #[label("expected `{node}` here")]
        span: SourceSpan,
    },

    /// Missing required property
    #[error("missing required `{property}` in `{node}`")]
    #[diagnostic(code(ciro::parse::missing_property))]
    MissingProperty {
        node: String,
        property: String,
        #[source_code]
        src: NamedSource<String>,
        #[label("this {node} is missing `{property}`")]
        span: SourceSpan,
    },

    /// Invalid value type
    #[error("expected {expected} for `{property}`")]
    #[diagnostic(code(ciro::parse::invalid_type))]
    InvalidType {
        property: String,
        expected: String,
        #[source_code]
        src: NamedSource<String>,
        #[label("expected {expected}")]
        span: SourceSpan,
    },

    /// Unknown node
    #[error("unknown node `{node}`")]
    #[diagnostic(code(ciro::parse::unknown_node), help("valid nodes are: {valid}"))]
    UnknownNode {
        node: String,
        valid: String,
        #[source_code]
        src: NamedSource<String>,
        #[label("unknown node")]
        span: SourceSpan,
    },

    /// Duplicate step name
    #[error("duplicate step name `{name}`")]
    #[diagnostic(code(ciro::parse::duplicate_step))]
    DuplicateStep {
        name: String,
        #[source_code]
        src: NamedSource<String>,
        #[label("duplicate step")]
        span: SourceSpan,
    },
}
