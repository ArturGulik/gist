
# --- git overlay (added by `gist completion bash --alias=git`) ---
#
# Registers a completion for the `git` command that augments git's own
# completion with gist's overlay subcommands. Source git's bash completion
# BEFORE this script, otherwise only gist's commands will be offered for
# `git <TAB>`.

_gist_overlay_git() {
    local cur cword
    cur="${COMP_WORDS[COMP_CWORD]}"
    cword=$COMP_CWORD

    if (( cword == 1 )); then
        if [[ -n "$_GIST_GIT_COMPL_FUNC" ]]; then
            "$_GIST_GIT_COMPL_FUNC"
        fi
        local gist_cmds="status update branch switch remote fetch config legend version"
        local g existing seen
        for g in $(compgen -W "$gist_cmds" -- "$cur"); do
            seen=0
            for existing in "${COMPREPLY[@]}"; do
                # git's completer suffixes entries with a trailing space.
                [[ "${existing% }" == "$g" ]] && { seen=1; break; }
            done
            (( seen )) || COMPREPLY+=( "$g" )
        done
        return
    fi

    local sub="${COMP_WORDS[1]}"
    case "$sub" in
        status|update|legend|config|version)
            return
            ;;
    esac

    if [[ -n "$_GIST_GIT_COMPL_FUNC" ]]; then
        "$_GIST_GIT_COMPL_FUNC"
    fi
}

complete -F _gist_overlay_git git
