"""Project initialization functionality."""

from importlib.resources import files
from pathlib import Path
from .config import save_default_config


def create_gitignore(gitignore_path: Path) -> None:
    """Create .gitignore for .spektacular directory."""
    gitignore_content = (
        files("spektacular").joinpath("defaults/.gitignore").read_text(encoding="utf-8")
    )

    with open(gitignore_path, "w") as f:
        f.write(gitignore_content.strip())


def create_conventions_md(conventions_path: Path) -> None:
    """Create a default conventions.md file."""
    conventions_content = (
        files("spektacular")
        .joinpath("defaults/conventions.md")
        .read_text(encoding="utf-8")
    )
    with open(conventions_path, "w") as f:
        f.write(conventions_content.strip())


def init_project(project_path: Path, force: bool = False) -> None:
    """Initialize a new Spektacular project structure.

    Args:
        project_path: Path to the project root
        force: Whether to overwrite existing .spektacular directory

    Raises:
        FileExistsError: If .spektacular already exists and force=False
        OSError: If directory creation fails
    """
    spektacular_dir = project_path / ".spektacular"

    # Check if .spektacular already exists
    if spektacular_dir.exists() and not force:
        raise FileExistsError(
            f".spektacular directory already exists at {spektacular_dir}. "
            "Use --force to overwrite."
        )

    # Create directory structure
    directories = [
        spektacular_dir,
        spektacular_dir / "plans",
        spektacular_dir / "specs",
        spektacular_dir / "knowledge",
        spektacular_dir / "knowledge" / "learnings",
        spektacular_dir / "knowledge" / "architecture",
        spektacular_dir / "knowledge" / "gotchas",
    ]

    for directory in directories:
        directory.mkdir(parents=True, exist_ok=True)

    # Create config files using Pydantic models
    save_default_config(spektacular_dir)
    create_gitignore(spektacular_dir / ".gitignore")
    create_conventions_md(spektacular_dir / "knowledge" / "conventions.md")

    # Create empty README files for knowledge directories
    readme_dirs = [
        spektacular_dir / "knowledge" / "learnings",
        spektacular_dir / "knowledge" / "architecture",
        spektacular_dir / "knowledge" / "gotchas",
    ]

    for directory in readme_dirs:
        readme_path = directory / "README.md"
        readme_path.write_text(
            f"# {directory.name.title()}\n\nThis directory contains {directory.name} documentation.\n"
        )
