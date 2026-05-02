
# --- git overlay (added by `gist completion zsh --alias=git`) ---
#
# Augments zsh's `_git` completion with gist's overlay subcommands. zsh's
# standard _git is loaded via fpath; this script wraps it.

_gist_overlay_git() {
    if (( $+functions[_git] )) || autoload -U _git 2>/dev/null; then
        _git "$@"
    fi
    if (( CURRENT == 2 )); then
        local -a gist_cmds
        gist_cmds=(
            'status:gist dashboard'
            'update:refresh PR/MR cache'
            'legend:print symbol legend'
            'config:print default config'
        )
        _describe 'gist overlay' gist_cmds
    fi
}

compdef _gist_overlay_git git
