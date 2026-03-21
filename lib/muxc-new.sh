#!/usr/bin/env bash
# muxc new — create a new named session and attach

cmd_new() {
    local name=""
    local target_cwd="$PWD"
    local tags=()
    local claude_passthrough=()
    local parsing_claude_args=false

    # Parse arguments
    while [[ $# -gt 0 ]]; do
        if $parsing_claude_args; then
            claude_passthrough+=("$1")
            shift
            continue
        fi
        case "$1" in
            --)
                parsing_claude_args=true
                shift
                ;;
            --cwd)
                [[ -n "${2:-}" ]] || die "--cwd requires a directory argument"
                target_cwd="$2"
                shift 2
                ;;
            --tag)
                [[ -n "${2:-}" ]] || die "--tag requires a value"
                tags+=("$2")
                shift 2
                ;;
            -*)
                die "Unknown option: $1. Use -- to pass flags to claude."
                ;;
            *)
                if [[ -z "$name" ]]; then
                    name="$1"
                else
                    die "Unexpected argument: $1"
                fi
                shift
                ;;
        esac
    done

    # Validate
    [[ -n "$name" ]] || die "Usage: muxc new <name> [--cwd <dir>] [--tag <t>...] [-- <claude-args>...]"
    validate_name "$name"

    if session_exists "$name"; then
        die "Session \"$name\" already exists. Use 'muxc attach $name' or choose a different name."
    fi

    # Resolve and validate cwd
    target_cwd="$(readlink -f "$target_cwd")"
    [[ -d "$target_cwd" ]] || die "Directory does not exist: $target_cwd"

    # Generate session UUID
    local uuid
    uuid=$(uuidgen) || die "Failed to generate UUID (is uuidgen installed?)"

    # Encode claude args
    local encoded_args=""
    if [[ ${#claude_passthrough[@]} -gt 0 ]]; then
        encoded_args=$(encode_claude_args "${claude_passthrough[*]}")
    fi

    # Create session directory and files
    local dir
    dir="$(session_dir "$name")"
    mkdir -p "$dir"

    # Set meta variables
    session_id="$uuid"
    claude_pid=""
    cwd="$target_cwd"
    status="active"
    created_at="$(iso_now)"
    accessed_at="$created_at"
    claude_args="$encoded_args"

    write_meta "$name"

    # Write tags
    if [[ ${#tags[@]} -gt 0 ]]; then
        printf '%s\n' "${tags[@]}" > "$dir/tags"
    else
        touch "$dir/tags"
    fi

    # Initialize notes and history
    touch "$dir/notes"
    append_history "$name" "created" "cwd=$target_cwd"

    success "Created session \"$name\" (cwd: $target_cwd)"

    # Navigate and launch
    nav "Navigating to $target_cwd"
    cd "$target_cwd" || die "Failed to cd to $target_cwd"

    local claude_bin
    claude_bin=$(get_claude_bin)

    # Build claude command
    local claude_cmd=("$claude_bin" "--session-id" "$uuid" "--name" "$name")
    if [[ ${#claude_passthrough[@]} -gt 0 ]]; then
        claude_cmd+=("${claude_passthrough[@]}")
    fi

    append_history "$name" "attached" "pid=$$"

    launch "Launching Claude Code..."
    exec "${claude_cmd[@]}"
}
