"""Main CLI interface for Spektacular."""

import click
from pathlib import Path

from . import __version__
from .config import SpektacularConfig
from .init import init_project
from .spec import create_spec
from .tui import run_plan_tui


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
    help="Overwrite existing .spektacular directory if it exists",
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


@cli.command()
@click.argument("name")
@click.option("--title", help="Feature title")
@click.option("--description", help="Feature description")
def new(name, title, description):
    """Create a new specification from template."""
    try:
        project_path = Path.cwd()
        spec_path = create_spec(project_path, name, title, description)
        click.echo(f"✅ Created spec: {spec_path}")
    except Exception as e:
        click.echo(f"❌ Error creating spec: {e}", err=True)
        raise click.Abort()


@cli.command()
@click.argument("spec_file", type=click.Path(exists=True, path_type=Path))
def plan(spec_file):
    """Generate an implementation plan from a specification."""
    try:
        project_path = Path.cwd()
        config_path = project_path / ".spektacular" / "config.yaml"
        if config_path.exists():
            config = SpektacularConfig.from_yaml_file(config_path)
        else:
            config = SpektacularConfig()
        plan_dir = run_plan_tui(spec_file, project_path, config)
        if plan_dir:
            click.echo(f"Plan generated: {plan_dir}")
    except Exception as e:
        click.echo(f"Error generating plan: {e}", err=True)
        raise click.Abort()


def main():
    """Entry point for the CLI."""
    cli()


if __name__ == "__main__":
    main()
