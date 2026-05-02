# gist bash completion
#
# Install:   source <(gist completion bash)
#
# Completes `gist` itself. For commands that aren't gist's own (anything that
# would fall through to git), this delegates to git's bash completion if it
# was sourced earlier — so e.g. `gist commit -m<TAB>` works the same as
# `git commit -m<TAB>`. Source git's completion before this script.

_gist() {
    local cur cword
    cur="${COMP_WORDS[COMP_CWORD]}"
    cword=$COMP_CWORD

    local cmds="status update branch switch remote fetch config legend version help"

    if (( cword == 1 )); then
        COMPREPLY=( $(compgen -W "$cmds" -- "$cur") )
        return
    fi

    local sub="${COMP_WORDS[1]}"
    case "$sub" in
        status|update|legend|config|version|help)
            return
            ;;
    esac

    if [[ -n "$_GIST_GIT_COMPL_FUNC" ]]; then
        local saved=("${COMP_WORDS[@]}")
        COMP_WORDS[0]=git
        "$_GIST_GIT_COMPL_FUNC"
        COMP_WORDS=("${saved[@]}")
    fi
}

_GIST_GIT_COMPL_FUNC=""
if _gist_existing=$(complete -p git 2>/dev/null); then
    _GIST_GIT_COMPL_FUNC=$(printf '%s' "$_gist_existing" | sed -n 's/.*-F \([^ ]\+\).*/\1/p')
fi
unset _gist_existing

complete -F _gist gist
