FROM ubuntu:24.04

RUN apt-get update && apt-get install -y --no-install-recommends \
    ca-certificates curl jq git \
    && rm -rf /var/lib/apt/lists/*

RUN curl -LsSf https://astral.sh/uv/0.9.5/install.sh | sh

ENV PATH="/root/.local/bin:$PATH"
RUN uv python install 3.12 && uv tool install pytest==8.4.1

COPY spektacular /usr/local/bin/spektacular
RUN chmod +x /usr/local/bin/spektacular

WORKDIR /app

# --- Colocated project + repo footprint at /app -----------------------------
#
# The auth-service project lives at /app. It is both a project and (through
# the "auth" member entry pointing at ".") a colocated repository. Both the
# project config and the repo config live under the same .spektacular/
# directory, which is what happens naturally when a repo is a member of a
# project that shares its working tree. init preserves an existing
# config.yaml/repo.yaml, so pre-seeding here is safe.
RUN mkdir -p /app/.spektacular/specs /app/.spektacular/plans/20260101000000-jwt-auth
COPY config.yaml /app/.spektacular/config.yaml
COPY repo.yaml /app/.spektacular/repo.yaml
COPY spec.md /app/.spektacular/specs/20260101000000-jwt-auth.md
COPY plan.md /app/.spektacular/plans/20260101000000-jwt-auth/plan.md
COPY test-plan.md /app/.spektacular/plans/20260101000000-jwt-auth/test-plan.md
COPY state.json /app/.spektacular/state.json

# --- External member repo footprint at /opt/docs-repo -----------------------
#
# The "docs" member repository has its own working tree at /opt/docs-repo
# with its own .spektacular/repo.yaml. It does not need to be a git repo —
# repo.Set.Resolve only checks that .spektacular/repo.yaml exists and parses.
# Any repo-routed changelog write (`changelog file write --repo docs`) will
# land under /opt/docs-repo/.spektacular/changelog/auth-service/<name>.md.
RUN mkdir -p /opt/docs-repo/.spektacular
COPY docs-repo.yaml /opt/docs-repo/.spektacular/repo.yaml

# The pre-seeded spec, plan, and test-plan files each carry in-progress
# frontmatter directly (see environment/*.md). The finished step's
# metadata.Close(StatusCompleted) call updates them to completed. No
# runtime frontmatter injection is needed.
