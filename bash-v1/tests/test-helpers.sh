#!/usr/bin/env bash
# test-helpers.sh — simple test framework for muxc

TEST_COUNT=0
PASS_COUNT=0
FAIL_COUNT=0

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
NC='\033[0m'

assert_eq() {
    local expected="$1"
    local actual="$2"
    local msg="${3:-assertion}"
    TEST_COUNT=$((TEST_COUNT + 1))
    if [[ "$expected" == "$actual" ]]; then
        PASS_COUNT=$((PASS_COUNT + 1))
        echo -e "  ${GREEN}✓${NC} $msg"
    else
        FAIL_COUNT=$((FAIL_COUNT + 1))
        echo -e "  ${RED}✗${NC} $msg"
        echo -e "    expected: '$expected'"
        echo -e "    actual:   '$actual'"
    fi
}

assert_contains() {
    local haystack="$1"
    local needle="$2"
    local msg="${3:-assertion}"
    TEST_COUNT=$((TEST_COUNT + 1))
    if [[ "$haystack" == *"$needle"* ]]; then
        PASS_COUNT=$((PASS_COUNT + 1))
        echo -e "  ${GREEN}✓${NC} $msg"
    else
        FAIL_COUNT=$((FAIL_COUNT + 1))
        echo -e "  ${RED}✗${NC} $msg"
        echo -e "    string:   '$haystack'"
        echo -e "    expected to contain: '$needle'"
    fi
}

assert_file_exists() {
    local path="$1"
    local msg="${2:-file exists: $path}"
    TEST_COUNT=$((TEST_COUNT + 1))
    if [[ -e "$path" ]]; then
        PASS_COUNT=$((PASS_COUNT + 1))
        echo -e "  ${GREEN}✓${NC} $msg"
    else
        FAIL_COUNT=$((FAIL_COUNT + 1))
        echo -e "  ${RED}✗${NC} $msg"
        echo -e "    path does not exist: $path"
    fi
}

assert_file_not_exists() {
    local path="$1"
    local msg="${2:-file does not exist: $path}"
    TEST_COUNT=$((TEST_COUNT + 1))
    if [[ ! -e "$path" ]]; then
        PASS_COUNT=$((PASS_COUNT + 1))
        echo -e "  ${GREEN}✓${NC} $msg"
    else
        FAIL_COUNT=$((FAIL_COUNT + 1))
        echo -e "  ${RED}✗${NC} $msg"
        echo -e "    path unexpectedly exists: $path"
    fi
}

assert_exit_code() {
    local expected="$1"
    local actual="$2"
    local msg="${3:-exit code}"
    assert_eq "$expected" "$actual" "$msg"
}

test_summary() {
    echo ""
    echo "────────────────────────────────────"
    echo "Tests: $TEST_COUNT | Passed: $PASS_COUNT | Failed: $FAIL_COUNT"
    if [[ $FAIL_COUNT -gt 0 ]]; then
        echo -e "${RED}FAILED${NC}"
        return 1
    else
        echo -e "${GREEN}ALL PASSED${NC}"
        return 0
    fi
}

# Setup a clean test environment
setup_test_env() {
    export MUXC_TEST_DIR
    MUXC_TEST_DIR=$(mktemp -d /tmp/muxc-test-XXXXXX)
    export MUXC_HOME="$MUXC_TEST_DIR/.muxc"

    # Create a mock claude binary
    export MOCK_CLAUDE="$MUXC_TEST_DIR/mock-claude"
    cat > "$MOCK_CLAUDE" <<'MOCK'
#!/usr/bin/env bash
# Mock claude — logs invocation and exits
echo "MOCK_CLAUDE_CALLED" >> "${MUXC_TEST_DIR}/claude-calls.log"
echo "ARGS: $*" >> "${MUXC_TEST_DIR}/claude-calls.log"
# Don't actually exec — just exit
exit 0
MOCK
    chmod +x "$MOCK_CLAUDE"

    # Override get_claude_bin to return mock
    mkdir -p "$MUXC_HOME"
    echo "claude_bin=$MOCK_CLAUDE" > "$MUXC_HOME/config"
}

# Cleanup test environment
cleanup_test_env() {
    if [[ -n "${MUXC_TEST_DIR:-}" && -d "$MUXC_TEST_DIR" ]]; then
        rm -rf "$MUXC_TEST_DIR"
    fi
}
