BINARY := spektacular
VERSION := 0.3.0

HARBOR_AUTH := ANTHROPIC_AUTH_TOKEN=$$(python3 -c "import json; print(json.load(open('$$HOME/.claude/.credentials.json'))['claudeAiOauth']['accessToken'])")
HARBOR_MODEL := claude-sonnet-4-6

.PHONY: build test lint clean install install-local cross harbor-test plan-harbor-test harbor-test-spec harbor-test-spec-claude harbor-test-spec-codex _harbor-test-spec

build:
	go build -ldflags "-X github.com/jumppad-labs/spektacular/cmd.version=$(VERSION)" -o ./bin/$(BINARY) .

test:
	go test ./...

lint:
	go vet ./...

clean:
	rm -f ./bin 

install-local: build
	sudo cp ./bin/$(BINARY) /usr/local/bin/$(BINARY)

dagger_build:
	dagger -v call --progress=plain -m dagger all \
		--output=./all_archive \
		--src=. \
		--notorize-cert=${QUILL_SIGN_P12} \
		--notorize-cert-password=QUILL_SIGN_PASSWORD \
		--notorize-key=${QUILL_NOTARY_KEY} \
		--notorize-id=${QUILL_NOTARY_KEY_ID} \
		--notorize-issuer=${QUILL_NOTARY_ISSUER}

#--github-token=GITHUB_TOKEN \

cross:
	CGO_ENABLED=0 GOOS=darwin  GOARCH=arm64 go build -o ./bin/$(BINARY)-darwin-arm64  .
	CGO_ENABLED=0 GOOS=darwin  GOARCH=amd64 go build -o ./bin/$(BINARY)-darwin-amd64  .
	CGO_ENABLED=0 GOOS=linux   GOARCH=amd64 go build -o ./bin/$(BINARY)-linux-amd64   .
	CGO_ENABLED=0 GOOS=linux   GOARCH=arm64 go build -o ./bin/$(BINARY)-linux-arm64   .
	CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -o ./bin/$(BINARY)-windows-amd64.exe .

harbor-test-spec: harbor-test-spec-claude

harbor-test-spec-claude:
	$(MAKE) _harbor-test-spec AGENT=claude HARBOR_AGENT=claude-code SPEK_NEW='/spek:new user-auth'

harbor-test-spec-codex:
	$(MAKE) _harbor-test-spec AGENT=codex HARBOR_AGENT=codex SPEK_NEW='$$$$spek-new user-auth'

# Renders tests/harbor/spec-workflow into tests/harbor/.build/spec-workflow-$(AGENT)
# with agent-specific placeholders substituted in instruction.md, then runs harbor.
# Callers must set AGENT, HARBOR_AGENT, SPEK_NEW.
_harbor-test-spec:
	@test -n "$(AGENT)" || (echo "AGENT is required" && exit 1)
	@mkdir -p tests/harbor/.build/spec-workflow-$(AGENT)/environment \
		tests/harbor/.build/spec-workflow-$(AGENT)/solution \
		tests/harbor/.build/spec-workflow-$(AGENT)/tests
	GOOS=linux GOARCH=amd64 go build -o tests/harbor/.build/spec-workflow-$(AGENT)/environment/spektacular .
	cp tests/harbor/spec-workflow/task.toml tests/harbor/.build/spec-workflow-$(AGENT)/task.toml
	cp tests/harbor/spec-workflow/environment/Dockerfile tests/harbor/.build/spec-workflow-$(AGENT)/environment/Dockerfile
	cp tests/harbor/spec-workflow/solution/solve.sh tests/harbor/.build/spec-workflow-$(AGENT)/solution/solve.sh
	cp tests/harbor/spec-workflow/tests/test.sh tests/harbor/.build/spec-workflow-$(AGENT)/tests/test.sh
	cp tests/harbor/spec-workflow/tests/test_spec_workflow.py tests/harbor/.build/spec-workflow-$(AGENT)/tests/test_spec_workflow.py
	sed -e 's|{{agent}}|$(AGENT)|g' -e 's|{{spek_new}}|$(SPEK_NEW)|g' \
		tests/harbor/spec-workflow/instruction.md \
		> tests/harbor/.build/spec-workflow-$(AGENT)/instruction.md
	$(HARBOR_AUTH) harbor run -p tests/harbor/.build/spec-workflow-$(AGENT) -a $(HARBOR_AGENT) -m $(HARBOR_MODEL) -o tests/harbor/jobs
	@echo ""
	@echo "=== Test Results ==="
	@cat $$(ls -td tests/harbor/jobs/*/spec-workflow-$(AGENT)__*/verifier/test-stdout.txt 2>/dev/null | head -1)

harbor-test-plan:
	GOOS=linux GOARCH=amd64 go build -o tests/harbor/plan-workflow/environment/spektacular .
	$(HARBOR_AUTH) harbor run -p tests/harbor/plan-workflow -a claude-code -m $(HARBOR_MODEL) -o tests/harbor/jobs
	@echo ""
	@echo "=== Test Results ==="
	@cat $$(ls -td tests/harbor/jobs/*/plan-workflow__*/verifier/test-stdout.txt 2>/dev/null | head -1)
