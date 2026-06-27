.PHONY: test test-go test-py schema-validate benchmark

test: test-go test-py schema-validate

test-go:
	go test ./...

test-py:
	cd verifiers/python-symbols && python -m pytest

schema-validate:
	go test ./protocol/...

# benchmark runs the quality-gate corpus (catch rate >= 80%, FP rate < 10%).
# This is a required CI gate. Run separately because it spawns a Python subprocess per case.
# Gated behind the `benchmark` build tag so it stays out of the everyday `go test ./...`.
benchmark:
	go test -tags benchmark ./benchmark/... -v -timeout 600s
