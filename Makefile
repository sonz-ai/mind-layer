PYTHON ?= $(if $(wildcard .venv/bin/python),.venv/bin/python,python3)

.PHONY: test verify run build benchmark docs demo demo-scenario
test:
	go test ./...
verify:
	$(PYTHON) scripts/verify.py
run:
	go run ./cmd/mind-layer
build:
	go build -trimpath -o bin/mind-layer ./cmd/mind-layer
	go build -trimpath -o bin/character-demo ./cmd/character-demo
benchmark:
	go run ./cmd/benchmark
docs:
	$(PYTHON) scripts/build_docs.py
demo:
	go run ./cmd/character-demo
demo-scenario:
	$(PYTHON) scripts/run_character_scenario.py
