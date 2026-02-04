//! Pipeline data model definitions.

/// A CI pipeline containing one or more steps.
#[derive(Debug, Clone, PartialEq, Eq)]
pub struct Pipeline {
    /// Name of the pipeline
    pub name: String,
    /// Steps in the pipeline
    pub steps: Vec<Step>,
}

/// A single step in a pipeline.
#[derive(Debug, Clone, PartialEq, Eq)]
pub struct Step {
    /// Name of the step
    pub name: String,
    /// Container image to use
    pub image: String,
    /// Commands to run
    pub run: Vec<String>,
    /// Working directory
    pub workdir: Option<String>,
    /// Step dependencies
    pub depends_on: Vec<String>,
    /// Mounts
    pub mounts: Vec<Mount>,
    /// Cache mounts
    pub caches: Vec<Cache>,
}

impl Step {
    /// Create a new step with the given name and image.
    #[must_use]
    pub fn new(name: impl Into<String>, image: impl Into<String>) -> Self {
        Self {
            name: name.into(),
            image: image.into(),
            run: Vec::new(),
            workdir: None,
            depends_on: Vec::new(),
            mounts: Vec::new(),
            caches: Vec::new(),
        }
    }
}

/// A bind mount definition.
#[derive(Debug, Clone, PartialEq, Eq)]
pub struct Mount {
    /// Source path (host or named)
    pub source: String,
    /// Target path in container
    pub target: String,
}

/// A cache mount definition.
#[derive(Debug, Clone, PartialEq, Eq)]
pub struct Cache {
    /// Cache identifier
    pub id: String,
    /// Target path in container
    pub target: String,
}
