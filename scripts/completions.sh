#!/bin/sh
# Emit shell completion scripts into completions/ (used by goreleaser + make docs).
set -e
mkdir -p completions
go run . completion bash > completions/finctl.bash
go run . completion zsh  > completions/_finctl
go run . completion fish > completions/finctl.fish
