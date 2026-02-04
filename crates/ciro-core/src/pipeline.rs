//! Pipeline data model definitions.

/// A CI pipeline containing one or more steps.
#[derive(Debug, Clone)]
pub struct Pipeline {
    /// Name of the pipeline
    pub name: String,
    /// Steps in the pipeline
    pub steps: Vec<Step>,
}

/// A single step in a pipeline.
#[derive(Debug, Clone)]
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
}

/// A mount definition.
#[derive(Debug, Clone)]
pub struct Mount {
    /// Source path (host or named)
    pub source: String,
    /// Target path in container
    pub target: String,
}
