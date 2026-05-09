"""Verify that the spektacular spec workflow completed successfully.

Tests are grouped by behaviour: one class per workflow step, plus a class
for overall workflow integrity.  Each step class checks that the step was
completed, that the agent called the spektacular CLI for it, and that it
produced meaningful, on-topic content in the spec file.
"""

import json
import re
from pathlib import Path

PROJECT_DIR = Path("/app")
SPEK_DIR = PROJECT_DIR / ".spektacular"
STATE_FILE = SPEK_DIR / "state.json"
AGENT_LOG_DIR = Path("/logs/agent")

# The canonical step order as defined by the state machine in
# internal/steps/spec/steps.go → Steps().
EXPECTED_STEP_ORDER = [
    "new",
    "overview",
    "requirements",
    "acceptance_criteria",
    "constraints",
    "technical_approach",
    "success_metrics",
    "non_goals",
    "verification",
    "finished",
]

# Minimum character count for each section's content (excluding comments).
MIN_SECTION_LENGTH = 100


# ---------------------------------------------------------------------------
# Helpers
# ---------------------------------------------------------------------------

def load_state() -> dict:
    assert STATE_FILE.exists(), f"State file not found at {STATE_FILE}"
    return json.loads(STATE_FILE.read_text())


def spec_file_path() -> Path:
    state = load_state()
    spec_name = (state.get("data") or {}).get("name")
    assert spec_name, "state.json has no data.name — cannot resolve spec file"
    return SPEK_DIR / "specs" / f"{spec_name}.md"


def load_spec() -> str:
    path = spec_file_path()
    assert path.exists(), f"Spec file not found at {path}"
    return path.read_text()


def parse_sections(spec_text: str) -> dict[str, str]:
    """Split the spec markdown into sections keyed by heading (lowercase).

    Returns a dict like {"overview": "section body …", "requirements": "…"}.
    Content between HTML comments (<!-- … -->) is stripped so we only measure
    real content the agent wrote.
    """
    cleaned = re.sub(r"<!--.*?-->", "", spec_text, flags=re.DOTALL)

    sections: dict[str, str] = {}
    current_heading = None
    current_lines: list[str] = []

    for line in cleaned.splitlines():
        heading_match = re.match(r"^##\s+(.+)", line)
        if heading_match:
            if current_heading is not None:
                sections[current_heading] = "\n".join(current_lines).strip()
            current_heading = heading_match.group(1).strip().lower()
            current_lines = []
        elif current_heading is not None:
            current_lines.append(line)

    if current_heading is not None:
        sections[current_heading] = "\n".join(current_lines).strip()

    return sections


def extract_tool_calls() -> list[dict]:
    """Parse the agent transcript and return all Bash tool calls in order.

    Each entry is {"command": "…"} for Bash tool_use blocks.
    """
    transcripts = sorted(AGENT_LOG_DIR.glob("*.txt"))
    assert transcripts, f"No agent transcripts found under {AGENT_LOG_DIR}"

    calls = []
    for transcript in transcripts:
        for line in transcript.read_text().splitlines():
            line = line.strip()
            if not line:
                continue
            try:
                obj = json.loads(line)
            except json.JSONDecodeError:
                continue
            if obj.get("type") == "item.completed":
                item = obj.get("item", {})
                if item.get("type") == "command_execution":
                    cmd = item.get("command", "")
                    if cmd:
                        calls.append({"command": cmd})
                    continue
            if obj.get("type") != "assistant":
                continue
            msg = obj.get("message", {})
            for block in msg.get("content", []):
                if block.get("type") == "tool_use" and block.get("name") == "Bash":
                    cmd = block.get("input", {}).get("command", "")
                    if cmd:
                        calls.append({"command": cmd})
    return calls


def find_spektacular_calls(tool_calls: list[dict]) -> list[str]:
    """Return an ordered list of spektacular spec commands from tool calls.

    Returns entries like "spec new" or "spec goto overview".
    """
    results = []
    for call in tool_calls:
        cmd = call["command"]
        if "spektacular spec new" in cmd:
            results.append("spec new")
        elif "spektacular spec goto" in cmd:
            normalized = cmd.replace('\\"', '"').replace("\\'", "'")
            m = re.search(r'["\']step["\']\s*:\s*["\']([a-z_]+)["\']', normalized)
            if m:
                results.append(f"spec goto {m.group(1)}")
                continue

            for step in EXPECTED_STEP_ORDER:
                if step == "new":
                    continue
                if re.search(rf"\b{re.escape(step)}\b", normalized):
                    results.append(f"spec goto {step}")
                    break
    return results


# ---------------------------------------------------------------------------
# Workflow-level tests
# ---------------------------------------------------------------------------

class TestWorkflow:
    """Overall workflow integrity checks."""

    def test_project_initialized(self):
        """spektacular init was run — config.yaml exists."""
        assert (SPEK_DIR / "config.yaml").exists(), "config.yaml missing — init not run"

    def test_spec_file_exists(self):
        """The spec markdown file was created."""
        path = spec_file_path()
        assert path.exists(), f"Spec file not found at {path}"

    def test_workflow_reached_finished(self):
        """Workflow current_step is finished or done."""
        state = load_state()
        assert state["current_step"] in ("finished", "done"), (
            f"Workflow did not finish, stuck at: {state['current_step']}"
        )

    def test_all_steps_completed(self):
        """Every expected step appears in completed_steps."""
        state = load_state()
        completed = set(state.get("completed_steps", []))
        missing = [s for s in EXPECTED_STEP_ORDER if s not in completed]
        assert not missing, f"Steps not completed: {missing}"

    def test_steps_executed_in_order(self):
        """completed_steps matches the order defined by the state machine."""
        state = load_state()
        completed = state.get("completed_steps", [])
        assert completed == EXPECTED_STEP_ORDER, (
            f"Steps out of order.\n"
            f"  Expected: {EXPECTED_STEP_ORDER}\n"
            f"  Got:      {completed}"
        )

    def test_no_placeholder_text(self):
        """Spec contains no TODO/TBD/FIXME placeholder markers."""
        content = load_spec()
        placeholder_markers = ["TODO", "TBD", "FIXME", "placeholder", "[insert"]
        found = [m for m in placeholder_markers if m in content]
        assert not found, f"Found placeholder markers in spec: {found}"


# ---------------------------------------------------------------------------
# Per-step behaviour tests
# ---------------------------------------------------------------------------

class TestNewStep:
    """The agent called spektacular spec new to create the spec."""

    def test_step_completed(self):
        state = load_state()
        assert "new" in state.get("completed_steps", [])

    def test_tool_called(self):
        calls = find_spektacular_calls(extract_tool_calls())
        assert "spec new" in calls, (
            f"Agent did not call 'spektacular spec new'. Calls: {calls}"
        )


class TestOverviewStep:
    """The agent completed the overview step with meaningful content."""

    def test_step_completed(self):
        state = load_state()
        assert "overview" in state.get("completed_steps", [])

    def test_tool_called(self):
        """Overview is reached automatically after spec new — verify new was called."""
        calls = find_spektacular_calls(extract_tool_calls())
        assert "spec new" in calls, (
            f"Agent did not call 'spektacular spec new' (which transitions to overview). Calls: {calls}"
        )

    def test_section_has_content(self):
        sections = parse_sections(load_spec())
        assert "overview" in sections, "Overview section not found in spec"
        assert len(sections["overview"]) >= MIN_SECTION_LENGTH, (
            f"Overview too short ({len(sections['overview'])} chars)"
        )

    def test_content_is_relevant(self):
        sections = parse_sections(load_spec())
        content = sections.get("overview", "").lower()
        assert any(term in content for term in ["jwt", "token", "auth"]), (
            "Overview does not mention JWT, token, or auth"
        )


class TestRequirementsStep:
    """The agent completed the requirements step with meaningful content."""

    def test_step_completed(self):
        state = load_state()
        assert "requirements" in state.get("completed_steps", [])

    def test_tool_called(self):
        calls = find_spektacular_calls(extract_tool_calls())
        assert "spec goto requirements" in calls, (
            f"Agent did not call 'spektacular spec goto' for requirements. Calls: {calls}"
        )

    def test_section_has_content(self):
        sections = parse_sections(load_spec())
        assert "requirements" in sections, "Requirements section not found in spec"
        assert len(sections["requirements"]) >= MIN_SECTION_LENGTH, (
            f"Requirements too short ({len(sections['requirements'])} chars)"
        )

    def test_has_multiple_requirements(self):
        sections = parse_sections(load_spec())
        content = sections.get("requirements", "")
        items = re.findall(r"^[-*]\s|\[.\]", content, re.MULTILINE)
        assert len(items) >= 3, (
            f"Expected at least 3 requirement items, found {len(items)}"
        )


class TestAcceptanceCriteriaStep:
    """The agent completed the acceptance criteria step with meaningful content."""

    def test_step_completed(self):
        state = load_state()
        assert "acceptance_criteria" in state.get("completed_steps", [])

    def test_tool_called(self):
        calls = find_spektacular_calls(extract_tool_calls())
        assert "spec goto acceptance_criteria" in calls, (
            f"Agent did not call 'spektacular spec goto' for acceptance_criteria. Calls: {calls}"
        )

    def test_section_has_content(self):
        sections = parse_sections(load_spec())
        assert "acceptance criteria" in sections, "Acceptance Criteria section not found"
        assert len(sections["acceptance criteria"]) >= MIN_SECTION_LENGTH, (
            f"Acceptance Criteria too short ({len(sections['acceptance criteria'])} chars)"
        )

    def test_has_verifiable_criteria(self):
        sections = parse_sections(load_spec())
        content = sections.get("acceptance criteria", "")
        items = re.findall(r"^[-*]\s|\[.\]", content, re.MULTILINE)
        assert len(items) >= 3, (
            f"Expected at least 3 acceptance criteria, found {len(items)}"
        )


class TestConstraintsStep:
    """The agent completed the constraints step with meaningful content."""

    def test_step_completed(self):
        state = load_state()
        assert "constraints" in state.get("completed_steps", [])

    def test_tool_called(self):
        calls = find_spektacular_calls(extract_tool_calls())
        assert "spec goto constraints" in calls, (
            f"Agent did not call 'spektacular spec goto' for constraints. Calls: {calls}"
        )

    def test_section_has_content(self):
        sections = parse_sections(load_spec())
        assert "constraints" in sections, "Constraints section not found"
        assert len(sections["constraints"]) >= MIN_SECTION_LENGTH, (
            f"Constraints too short ({len(sections['constraints'])} chars)"
        )


class TestTechnicalApproachStep:
    """The agent completed the technical approach step with meaningful content."""

    def test_step_completed(self):
        state = load_state()
        assert "technical_approach" in state.get("completed_steps", [])

    def test_tool_called(self):
        calls = find_spektacular_calls(extract_tool_calls())
        assert "spec goto technical_approach" in calls, (
            f"Agent did not call 'spektacular spec goto' for technical_approach. Calls: {calls}"
        )

    def test_section_has_content(self):
        sections = parse_sections(load_spec())
        assert "technical approach" in sections, "Technical Approach section not found"
        assert len(sections["technical approach"]) >= MIN_SECTION_LENGTH, (
            f"Technical Approach too short ({len(sections['technical approach'])} chars)"
        )

    def test_content_is_technical(self):
        sections = parse_sections(load_spec())
        content = sections.get("technical approach", "").lower()
        assert any(term in content for term in [
            "api", "endpoint", "database", "service", "key", "token", "jwt",
            "architecture", "migration", "schema",
        ]), "Technical Approach lacks technical content"


class TestSuccessMetricsStep:
    """The agent completed the success metrics step with meaningful content."""

    def test_step_completed(self):
        state = load_state()
        assert "success_metrics" in state.get("completed_steps", [])

    def test_tool_called(self):
        calls = find_spektacular_calls(extract_tool_calls())
        assert "spec goto success_metrics" in calls, (
            f"Agent did not call 'spektacular spec goto' for success_metrics. Calls: {calls}"
        )

    def test_section_has_content(self):
        sections = parse_sections(load_spec())
        assert "success metrics" in sections, "Success Metrics section not found"
        assert len(sections["success metrics"]) >= MIN_SECTION_LENGTH, (
            f"Success Metrics too short ({len(sections['success metrics'])} chars)"
        )


class TestNonGoalsStep:
    """The agent completed the non-goals step with meaningful content."""

    def test_step_completed(self):
        state = load_state()
        assert "non_goals" in state.get("completed_steps", [])

    def test_tool_called(self):
        calls = find_spektacular_calls(extract_tool_calls())
        assert "spec goto non_goals" in calls, (
            f"Agent did not call 'spektacular spec goto' for non_goals. Calls: {calls}"
        )

    def test_section_has_content(self):
        sections = parse_sections(load_spec())
        assert "non-goals" in sections, "Non-Goals section not found"
        assert len(sections["non-goals"]) >= MIN_SECTION_LENGTH, (
            f"Non-Goals too short ({len(sections['non-goals'])} chars)"
        )


class TestVerificationStep:
    """The agent completed the verification step."""

    def test_step_completed(self):
        state = load_state()
        assert "verification" in state.get("completed_steps", [])

    def test_tool_called(self):
        calls = find_spektacular_calls(extract_tool_calls())
        assert "spec goto verification" in calls, (
            f"Agent did not call 'spektacular spec goto' for verification. Calls: {calls}"
        )
