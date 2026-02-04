use std::path::PathBuf;

use clap::{Parser, Subcommand};
use miette::{IntoDiagnostic, Result};
use tracing_subscriber::EnvFilter;

#[derive(Parser)]
#[command(name = "ciro")]
#[command(author, version, about = "A minimal CI platform powered by BuildKit")]
struct Cli {
    #[command(subcommand)]
    command: Option<Commands>,
}

#[derive(Subcommand)]
enum Commands {
    /// Validate a KDL pipeline file
    Validate {
        /// Path to the KDL pipeline file
        file: PathBuf,
    },
    /// Run a KDL pipeline
    Run {
        /// Path to the KDL pipeline file
        file: PathBuf,

        /// `BuildKit` daemon address
        #[arg(long, default_value = "tcp://127.0.0.1:1234")]
        addr: String,

        /// Dry run - print LLB without executing
        #[arg(long)]
        dry_run: bool,
    },
}

fn main() -> Result<()> {
    miette::set_hook(Box::new(|_| {
        Box::new(
            miette::MietteHandlerOpts::new()
                .terminal_links(true)
                .context_lines(3)
                .build(),
        )
    }))
    .ok();

    tracing_subscriber::fmt()
        .with_env_filter(EnvFilter::from_default_env())
        .init();

    let cli = Cli::parse();

    match cli.command {
        Some(Commands::Validate { file }) => {
            let path = file.canonicalize().into_diagnostic()?;
            let pipeline = ciro_parser::parse_pipeline(&path)?;
            println!("Pipeline '{}' is valid", pipeline.name);
            println!("  Steps: {}", pipeline.steps.len());
            for step in &pipeline.steps {
                println!("    - {} (image: {})", step.name, step.image);
            }
            Ok(())
        }
        Some(Commands::Run {
            file,
            addr,
            dry_run,
        }) => {
            let path = file.canonicalize().into_diagnostic()?;
            let pipeline = ciro_parser::parse_pipeline(&path)?;
            println!(
                "Running pipeline '{}' (addr: {}, dry_run: {})",
                pipeline.name, addr, dry_run
            );
            // TODO: Implement execution in Phase 4
            Ok(())
        }
        None => {
            println!("Use --help to see available commands");
            Ok(())
        }
    }
}
