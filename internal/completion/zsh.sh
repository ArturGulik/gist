#compdef gist
# gist zsh completion
#
# Install:   source <(gist completion zsh)
#
# Completes `gist` itself. For commands that fall through to git (anything
# not on gist's overlay list), delegates to zsh's _git completer.

_gist() {
    local -a subcmds
    subcmds=(
        'status:show repo dashboard'
        'update:refresh PR/MR cache'
        'branch:branch info'
        'switch:switch branches'
        'remote:show remotes'
        'fetch:fetch then refresh PR cache'
        'config:print default config'
        'legend:print symbol legend'
        'version:print version'
        'help:print help'
    )

    if (( CURRENT == 2 )); then
        _describe 'gist command' subcmds
        return
    fi

    case "${words[2]}" in
        status|update|legend|config|version|help)
            return
            ;;
    esac

    (( $+functions[_git] )) || autoload -U _git 2>/dev/null
    if (( $+functions[_git] )); then
        local -a saved_words
        saved_words=("${words[@]}")
        words[1]=git
        _git
        words=("${saved_words[@]}")
    fi
}

compdef _gist gist
