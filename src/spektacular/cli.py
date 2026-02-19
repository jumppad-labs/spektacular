"""Main CLI interface for Spektacular."""

import click
from pathlib import Path

from . import __version__
from .init import init_project


@click.group()
@click.version_option(version=__version__)
@click.pass_context
def cli(ctx):
    """Spektacular: Agent-agnostic tool for spec-driven development."""
    ctx.ensure_object(dict)


@cli.command()
@click.option(
    "--force", 
    is_flag=True, 
    help="Overwrite existing .spektacular directory if it exists"
)
def init(force):
    """Initialize a new Spektacular project structure."""
    try:
        project_path = Path.cwd()
        init_project(project_path, force=force)
        click.echo(f"✅ Initialized Spektacular project in {project_path}")
    except Exception as e:
        click.echo(f"❌ Error initializing project: {e}", err=True)
        raise click.Abort()


@cli.command()
@click.argument("spec_file", type=click.Path(exists=True, path_type=Path))
def run(spec_file):
    """Run Spektacular on a specification file."""
    click.echo(f"🚀 Processing spec: {spec_file}")
    # TODO: Implement spec processing
    click.echo("⚠️  Spec processing not yet implemented")


def main():
    """Entry point for the CLI."""
    cli()


if __name__ == "__main__":
    main()
