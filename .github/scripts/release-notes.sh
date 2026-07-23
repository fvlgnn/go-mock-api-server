#!/bin/sh
set -eu

tag="${1:?usage: release-notes.sh <vMAJOR.MINOR.PATCH> <output-file>}"
output="${2:?usage: release-notes.sh <vMAJOR.MINOR.PATCH> <output-file>}"
changelog="${CHANGELOG_FILE:-CHANGELOG.md}"

if ! printf '%s\n' "$tag" | grep -Eq '^v[0-9]+\.[0-9]+\.[0-9]+$'; then
    printf 'invalid release tag: %s\n' "$tag" >&2
    exit 1
fi

version="${tag#v}"

awk -v expected="## [$version]" '
    /^## \[/ {
        if (found) {
            exit
        }
        if (index($0, expected) == 1) {
            found = 1
            next
        }
    }
    found && /^\[[^]]+\]:/ { exit }
    found { print }
    END {
        if (!found) {
            exit 2
        }
    }
' "$changelog" > "$output" || {
    printf 'CHANGELOG section [%s] not found in %s\n' "$version" "$changelog" >&2
    exit 1
}

if ! grep -q '[^[:space:]]' "$output"; then
    printf 'CHANGELOG section [%s] is empty\n' "$version" >&2
    exit 1
fi
