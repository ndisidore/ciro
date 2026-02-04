//! KDL to Pipeline parser implementation.

// Diagnostic errors intentionally carry source context for rich error messages.
// This makes the error type large, but is a deliberate design choice for good UX.
#![allow(clippy::result_large_err)]

use std::collections::HashSet;
use std::path::Path;

use ciro_core::{Cache, Mount, Pipeline, Step};
use kdl::{KdlDocument, KdlNode, KdlValue};
use miette::NamedSource;

use crate::error::ParseError;

/// Parse a KDL file into a Pipeline.
///
/// # Errors
///
/// Returns a `ParseError` if the file cannot be parsed or contains invalid content.
pub fn parse_pipeline(path: &Path) -> Result<Pipeline, ParseError> {
    let content = std::fs::read_to_string(path).map_err(|e| ParseError::Io {
        message: format!("{}: {e}", path.display()),
    })?;

    parse_pipeline_str(&content, path.display().to_string())
}

/// Parse a KDL string into a Pipeline.
///
/// # Errors
///
/// Returns a `ParseError` if the content cannot be parsed or contains invalid content.
pub fn parse_pipeline_str(content: &str, filename: String) -> Result<Pipeline, ParseError> {
    let doc: KdlDocument = content.parse()?;

    let src = NamedSource::new(filename, content.to_string());
    parse_document(&doc, src)
}

fn parse_document(doc: &KdlDocument, src: NamedSource<String>) -> Result<Pipeline, ParseError> {
    let pipeline_node = doc
        .nodes()
        .iter()
        .find(|n| n.name().value() == "pipeline")
        .ok_or_else(|| ParseError::MissingNode {
            node: "pipeline".to_string(),
            src: src.clone(),
            span: (0, 1).into(),
        })?;

    let name = get_first_string_arg(pipeline_node, "pipeline", &src)?;

    let children = pipeline_node
        .children()
        .ok_or_else(|| ParseError::MissingNode {
            node: "step".to_string(),
            src: src.clone(),
            span: node_span(pipeline_node),
        })?;

    let mut steps = Vec::new();
    let mut step_names = HashSet::new();

    for node in children.nodes() {
        match node.name().value() {
            "step" => {
                let step = parse_step(node, &src)?;
                if !step_names.insert(step.name.clone()) {
                    return Err(ParseError::DuplicateStep {
                        name: step.name,
                        src,
                        span: node_span(node),
                    });
                }
                steps.push(step);
            }
            other => {
                return Err(ParseError::UnknownNode {
                    node: other.to_string(),
                    valid: "step".to_string(),
                    src,
                    span: node_span(node),
                });
            }
        }
    }

    if steps.is_empty() {
        return Err(ParseError::MissingNode {
            node: "step".to_string(),
            src,
            span: node_span(pipeline_node),
        });
    }

    Ok(Pipeline { name, steps })
}

fn parse_step(node: &KdlNode, src: &NamedSource<String>) -> Result<Step, ParseError> {
    let name = get_first_string_arg(node, "step", src)?;

    let children = node.children().ok_or_else(|| ParseError::MissingProperty {
        node: "step".to_string(),
        property: "image".to_string(),
        src: src.clone(),
        span: node_span(node),
    })?;

    let mut step = Step::new(name, String::new());

    for child in children.nodes() {
        match child.name().value() {
            "image" => {
                step.image = get_first_string_arg(child, "image", src)?;
            }
            "run" => {
                let cmd = get_first_string_arg(child, "run", src)?;
                step.run.push(cmd);
            }
            "workdir" => {
                step.workdir = Some(get_first_string_arg(child, "workdir", src)?);
            }
            "depends-on" => {
                let dep = get_first_string_arg(child, "depends-on", src)?;
                step.depends_on.push(dep);
            }
            "mount" => {
                let mount = parse_mount(child, src)?;
                step.mounts.push(mount);
            }
            "cache" => {
                let cache = parse_cache(child, src)?;
                step.caches.push(cache);
            }
            other => {
                return Err(ParseError::UnknownNode {
                    node: other.to_string(),
                    valid: "image, run, workdir, depends-on, mount, cache".to_string(),
                    src: src.clone(),
                    span: node_span(child),
                });
            }
        }
    }

    if step.image.is_empty() {
        return Err(ParseError::MissingProperty {
            node: "step".to_string(),
            property: "image".to_string(),
            src: src.clone(),
            span: node_span(node),
        });
    }

    Ok(step)
}

fn parse_mount(node: &KdlNode, src: &NamedSource<String>) -> Result<Mount, ParseError> {
    let entries = node.entries();
    if entries.len() < 2 {
        return Err(ParseError::InvalidType {
            property: "mount".to_string(),
            expected: "two string arguments (source, target)".to_string(),
            src: src.clone(),
            span: node_span(node),
        });
    }

    let source = entries
        .first()
        .and_then(|e| e.value().as_string())
        .ok_or_else(|| ParseError::InvalidType {
            property: "mount source".to_string(),
            expected: "string".to_string(),
            src: src.clone(),
            span: node_span(node),
        })?
        .to_string();

    let target = entries
        .get(1)
        .and_then(|e| e.value().as_string())
        .ok_or_else(|| ParseError::InvalidType {
            property: "mount target".to_string(),
            expected: "string".to_string(),
            src: src.clone(),
            span: node_span(node),
        })?
        .to_string();

    Ok(Mount { source, target })
}

fn parse_cache(node: &KdlNode, src: &NamedSource<String>) -> Result<Cache, ParseError> {
    let entries = node.entries();
    if entries.len() < 2 {
        return Err(ParseError::InvalidType {
            property: "cache".to_string(),
            expected: "two string arguments (id, target)".to_string(),
            src: src.clone(),
            span: node_span(node),
        });
    }

    let id = entries
        .first()
        .and_then(|e| e.value().as_string())
        .ok_or_else(|| ParseError::InvalidType {
            property: "cache id".to_string(),
            expected: "string".to_string(),
            src: src.clone(),
            span: node_span(node),
        })?
        .to_string();

    let target = entries
        .get(1)
        .and_then(|e| e.value().as_string())
        .ok_or_else(|| ParseError::InvalidType {
            property: "cache target".to_string(),
            expected: "string".to_string(),
            src: src.clone(),
            span: node_span(node),
        })?
        .to_string();

    Ok(Cache { id, target })
}

fn get_first_string_arg(
    node: &KdlNode,
    node_name: &str,
    src: &NamedSource<String>,
) -> Result<String, ParseError> {
    node.entries()
        .first()
        .and_then(|e| {
            if let KdlValue::String(s) = e.value() {
                Some(s.clone())
            } else {
                None
            }
        })
        .ok_or_else(|| ParseError::InvalidType {
            property: node_name.to_string(),
            expected: "string argument".to_string(),
            src: src.clone(),
            span: node_span(node),
        })
}

fn node_span(node: &KdlNode) -> miette::SourceSpan {
    let span = node.span();
    (span.offset(), span.len()).into()
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_parse_simple_pipeline() {
        let content = r#"
pipeline "test" {
    step "hello" {
        image "alpine:latest"
        run "echo hello"
    }
}
"#;
        let pipeline = parse_pipeline_str(content, "test.kdl".to_string()).unwrap();
        assert_eq!(pipeline.name, "test");
        assert_eq!(pipeline.steps.len(), 1);
        assert_eq!(pipeline.steps[0].name, "hello");
        assert_eq!(pipeline.steps[0].image, "alpine:latest");
        assert_eq!(pipeline.steps[0].run, vec!["echo hello"]);
    }

    #[test]
    fn test_parse_step_with_dependencies() {
        let content = r#"
pipeline "test" {
    step "first" {
        image "alpine"
        run "echo first"
    }
    step "second" {
        image "alpine"
        depends-on "first"
        run "echo second"
    }
}
"#;
        let pipeline = parse_pipeline_str(content, "test.kdl".to_string()).unwrap();
        assert_eq!(pipeline.steps.len(), 2);
        assert_eq!(pipeline.steps[1].depends_on, vec!["first"]);
    }

    #[test]
    fn test_parse_step_with_mount() {
        let content = r#"
pipeline "test" {
    step "build" {
        image "rust:latest"
        mount "." "/src"
        workdir "/src"
        run "cargo build"
    }
}
"#;
        let pipeline = parse_pipeline_str(content, "test.kdl".to_string()).unwrap();
        assert_eq!(pipeline.steps[0].mounts.len(), 1);
        assert_eq!(pipeline.steps[0].mounts[0].source, ".");
        assert_eq!(pipeline.steps[0].mounts[0].target, "/src");
        assert_eq!(pipeline.steps[0].workdir, Some("/src".to_string()));
    }

    #[test]
    fn test_parse_step_with_cache() {
        let content = r#"
pipeline "test" {
    step "build" {
        image "rust:latest"
        cache "cargo-cache" "/usr/local/cargo/registry"
        run "cargo build"
    }
}
"#;
        let pipeline = parse_pipeline_str(content, "test.kdl".to_string()).unwrap();
        assert_eq!(pipeline.steps[0].caches.len(), 1);
        assert_eq!(pipeline.steps[0].caches[0].id, "cargo-cache");
        assert_eq!(
            pipeline.steps[0].caches[0].target,
            "/usr/local/cargo/registry"
        );
    }

    #[test]
    fn test_missing_pipeline() {
        let content = r#"
step "test" {
    image "alpine"
}
"#;
        let result = parse_pipeline_str(content, "test.kdl".to_string());
        assert!(result.is_err());
    }

    #[test]
    fn test_missing_image() {
        let content = r#"
pipeline "test" {
    step "hello" {
        run "echo hello"
    }
}
"#;
        let result = parse_pipeline_str(content, "test.kdl".to_string());
        assert!(result.is_err());
    }

    #[test]
    fn test_duplicate_step_name() {
        let content = r#"
pipeline "test" {
    step "hello" {
        image "alpine"
    }
    step "hello" {
        image "alpine"
    }
}
"#;
        let result = parse_pipeline_str(content, "test.kdl".to_string());
        assert!(result.is_err());
    }
}
