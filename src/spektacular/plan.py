"""Plan mode orchestration -- transforms specs into plans."""

from importlib.resources import files
from pathlib import Path

import click

from .config import SpektacularConfig
from .runner import build_prompt, detect_questions, run_claude


def load_knowledge(project_path: Path) -> dict[str, str]:
    """Load all knowledge files from .spektacular/knowledge/."""
    knowledge_dir = project_path / ".spektacular" / "knowledge"
    contents = {}
    if knowledge_dir.exists():
        for md_file in sorted(knowledge_dir.rglob("*.md")):
            relative = md_file.relative_to(knowledge_dir)
            contents[str(relative)] = md_file.read_text(encoding="utf-8")
    return contents


def load_agent_prompt() -> str:
    """Load the planner agent prompt from package defaults."""
    return (
        files("spektacular")
        .joinpath("defaults/agents/planner.md")
        .read_text(encoding="utf-8")
    )


def prompt_user_for_answer(questions: list) -> str:
    """Prompt the user in the terminal to answer questions."""
    answers = []
    for q in questions:
        click.echo(f"\n{'='*60}")
        click.echo(f"  {q.header}: {q.question}")
        click.echo(f"{'='*60}")
        if q.options:
            for i, opt in enumerate(q.options, 1):
                click.echo(f"  {i}. {opt['label']} -- {opt.get('description', '')}")
            click.echo()
            choice = click.prompt("  Select option (number) or type custom answer", default="1")
            try:
                idx = int(choice) - 1
                if 0 <= idx < len(q.options):
                    answers.append(q.options[idx]["label"])
                else:
                    answers.append(choice)
            except ValueError:
                answers.append(choice)
        else:
            answer = click.prompt("  Your answer")
            answers.append(answer)
    return "; ".join(answers)


def write_plan_output(plan_dir: Path, result_text: str) -> None:
    """Write the agent's output to plan directory."""
    plan_dir.mkdir(parents=True, exist_ok=True)
    (plan_dir / "plan.md").write_text(result_text, encoding="utf-8")


def run_plan(
    spec_path: Path,
    project_path: Path,
    config: SpektacularConfig,
) -> Path:
    """Execute the plan workflow for a specification."""
    spec_content = spec_path.read_text(encoding="utf-8")
    agent_prompt = load_agent_prompt()
    knowledge = load_knowledge(project_path)
    prompt = build_prompt(spec_content, agent_prompt, knowledge)

    spec_name = spec_path.stem
    plan_dir = project_path / ".spektacular" / "plans" / spec_name

    session_id = None
    final_result = None

    click.echo(f"Starting plan generation for: {spec_path.name}")
    click.echo(f"Output directory: {plan_dir}\n")

    current_prompt = prompt
    while True:
        questions_found = []
        for event in run_claude(current_prompt, config, session_id, project_path):
            if event.session_id:
                session_id = event.session_id
            if text := event.text_content:
                click.echo(text)
                detected = detect_questions(text)
                if detected:
                    questions_found.extend(detected)
            if event.is_result:
                if event.is_error:
                    raise RuntimeError(f"Agent error: {event.result_text}")
                final_result = event.result_text

        if questions_found:
            answer = prompt_user_for_answer(questions_found)
            click.echo("\nResuming agent with answer...")
            current_prompt = answer
            continue
        break

    if final_result:
        write_plan_output(plan_dir, final_result)
        click.echo(f"\nPlan written to: {plan_dir}/plan.md")
    else:
        raise RuntimeError("Agent completed without producing a result")
    return plan_dir
