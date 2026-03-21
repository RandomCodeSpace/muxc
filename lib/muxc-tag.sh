#!/usr/bin/env bash
# muxc tag — manage session tags

cmd_tag() {
    local name="${1:-}"
    local action="${2:-}"
    local tag="${3:-}"

    [[ -n "$name" && -n "$action" && -n "$tag" ]] || die "Usage: muxc tag <name> add|rm <tag>"
    session_exists "$name" || die "Session \"$name\" not found"

    local dir
    dir="$(session_dir "$name")"
    local tags_file="$dir/tags"
    touch "$tags_file"

    case "$action" in
        add)
            # Check for duplicate
            if grep -qxF "$tag" "$tags_file" 2>/dev/null; then
                warn "Tag \"$tag\" already exists on session \"$name\""
                return 0
            fi
            echo "$tag" >> "$tags_file"
            append_history "$name" "tag-add" "tag=$tag"
            success "Added tag \"$tag\" to \"$name\""
            ;;
        rm|remove)
            if ! grep -qxF "$tag" "$tags_file" 2>/dev/null; then
                warn "Tag \"$tag\" not found on session \"$name\""
                return 0
            fi
            grep -vxF "$tag" "$tags_file" > "$tags_file.tmp" || true
            mv -f "$tags_file.tmp" "$tags_file"
            append_history "$name" "tag-rm" "tag=$tag"
            success "Removed tag \"$tag\" from \"$name\""
            ;;
        *)
            die "Unknown action: $action. Use 'add' or 'rm'."
            ;;
    esac
}
