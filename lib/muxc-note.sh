#!/usr/bin/env bash
# muxc note — add or edit session notes

cmd_note() {
    local name="${1:-}"
    [[ -n "$name" ]] || die "Usage: muxc note <name> [<text>]"
    shift

    session_exists "$name" || die "Session \"$name\" not found"

    local dir
    dir="$(session_dir "$name")"
    local notes_file="$dir/notes"

    if [[ $# -gt 0 ]]; then
        # Append text to notes
        echo "$*" >> "$notes_file"
        append_history "$name" "note" "appended"
        success "Notes updated for \"$name\""
    else
        # Open in editor
        local editor="${EDITOR:-vi}"
        "$editor" "$notes_file"
        append_history "$name" "note" "edited"
        success "Notes saved for \"$name\""
    fi
}
