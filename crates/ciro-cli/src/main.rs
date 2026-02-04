use clap::{Parser, Subcommand};
use miette::Result;
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
        file: std::path::PathBuf,
    },
    /// Run a KDL pipeline
    Run {
        /// Path to the KDL pipeline file
        file: std::path::PathBuf,

        /// BuildKit daemon address
        #[arg(long, default_value = "tcp://127.0.0.1:1234")]
        addr: String,

        /// Dry run - print LLB without executing
        #[arg(long)]
        dry_run: bool,
    },
}

fn main() -> Result<()> {
    tracing_subscriber::fmt()
        .with_env_filter(EnvFilter::from_default_env())
        .init();

    let cli = Cli::parse();

    match cli.command {
        Some(Commands::Validate { file }) => {
            println!("Validating: {}", file.display());
            // TODO: Implement validation in Phase 2
            Ok(())
        }
        Some(Commands::Run { file, addr, dry_run }) => {
            println!("Running: {} (addr: {}, dry_run: {})", file.display(), addr, dry_run);
            // TODO: Implement execution in Phase 4
            Ok(())
        }
        None => {
            println!("Use --help to see available commands");
            Ok(())
        }
    }
}
