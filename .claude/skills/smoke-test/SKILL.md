---
name: smoke-test
description: Run syntax check, shellcheck, and basic muxc commands for quick validation
disable-model-invocation: true
---

# Smoke Test

Run all validation checks for the muxc script.

## Steps

1. Run bash syntax check: `bash -n muxc`
2. Run ShellCheck linter: `shellcheck muxc`
3. Run basic commands: `./muxc version`, `./muxc ls`
4. Report results — pass or fail with details
