"""Specification creation functionality."""

from pathlib import Path
from importlib.resources import files


def get_template_path() -> Path:
    """Get the path to the spec template file."""
    template_path = Path(__file__).parent / "defaults" / "spec-template.md"
    return template_path


def create_spec(
    project_path: Path,
    name: str,
    title: str | None = None,
    description: str | None = None,
) -> Path:
    """Create a new specification from template.

    Args:
        name: The spec filename (without extension)
        title: Feature title (defaults to name)
        description: Feature description

    Returns:
        Path to the created spec file
    """
    # Default values
    if title is None:
        title = name.replace("-", " ").replace("_", " ").title()
    if description is None:
        description = f"Add description for {title} here."

    # Read template

    template_content = (
        files("spektacular")
        .joinpath("defaults/spec-template.md")
        .read_text(encoding="utf-8")
    )

    # Replace placeholders
    replacements = {
        "{title}": title,
        "{description}": description,
        "{requirement_1}": "Add first requirement",
        "{requirement_2}": "Add second requirement",
        "{requirement_3}": "Add third requirement",
        "{constraint_1}": "Add first constraint",
        "{constraint_2}": "Add second constraint",
        "{criteria_1}": "Add first acceptance criterion",
        "{criteria_2}": "Add second acceptance criterion",
        "{criteria_3}": "Add third acceptance criterion",
        "{technical_notes}": "Add technical approach details",
        "{success_metrics}": "Add success metrics",
        "{non_goals}": "Add non-goals",
    }

    spec_content = template_content
    for placeholder, replacement in replacements.items():
        spec_content = spec_content.replace(placeholder, replacement)

    # Write spec file
    spec_filename = f"{name}.md" if not name.endswith(".md") else name
    spec_path = project_path / ".spektacular" / "specs" / spec_filename

    if spec_path.exists():
        raise FileExistsError(f"Spec file already exists: {spec_path}")

    spec_path.write_text(spec_content)
    return spec_path
