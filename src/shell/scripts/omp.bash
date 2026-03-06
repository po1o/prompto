export POSH_SHELL='bash'
export POSH_SHELL_VERSION=$BASH_VERSION
export POWERLINE_COMMAND='oh-my-posh'
export CONDA_PROMPT_MODIFIER=false
export OSTYPE=$OSTYPE

# disable all known python virtual environment prompts
export VIRTUAL_ENV_DISABLE_PROMPT=1
export PYENV_VIRTUALENV_DISABLE_PROMPT=1

# global variables
_omp_start_time=''
_omp_stack_count=0
_omp_execution_time=-1
_omp_no_status=true
_omp_status=0
_omp_pipestatus=0
_omp_executable=::OMP::

# switches to enable/disable features
_omp_cursor_positioning=0
_omp_ftcs_marks=0

# start timer on command start
PS0='${_omp_start_time:0:$((_omp_start_time="$(_omp_start_timer)",0))}$(_omp_ftcs_command_start)'

# set secondary prompt
_omp_secondary_prompt=(
    "$_omp_executable" render secondary \
        --shell=bash \
        --shell-version="$BASH_VERSION"
)

function _omp_set_cursor_position() {
    # not supported in Midnight Commander
    # see https://github.com/JanDeDobbeleer/oh-my-posh/issues/3415
    if [[ $_omp_cursor_positioning == 0 ]] || [[ -v MC_SID ]]; then
        return
    fi

    local oldstty=$(stty -g)
    stty raw -echo min 0

    local COL
    local ROW
    IFS=';' read -rsdR -p $'
[6n' ROW COL

    stty "$oldstty"

    export POSH_CURSOR_LINE=${ROW#*[}
    export POSH_CURSOR_COLUMN=${COL}
}

function _omp_start_timer() {
    "$_omp_executable" get millis
}

function _omp_ftcs_command_start() {
    if [[ $_omp_ftcs_marks == 1 ]]; then
        printf '\e]133;C\a'
    fi
}

# template function for context loading
function set_poshcontext() {
    return
}

function _omp_get_primary() {
    # Avoid unexpected expansions when we're generating the prompt below.
    shopt -u promptvars
    trap 'shopt -s promptvars' RETURN

    local vim_mode_arg=""
    if [[ $_omp_vim_mode == 1 ]]; then
        vim_mode_arg="--vim-mode=$(_omp_get_vim_mode)"
    fi

    local prompt
    if shopt -oq posix; then
        # Disable in POSIX mode.
        prompt='[NOTICE: Oh My Posh prompt is not supported in POSIX mode]\n\u@\h:\w\$ '
    else
        prompt=(
            "$_omp_executable" render primary \
                --shell=bash \
                --shell-version="$BASH_VERSION" \
                --status="$_omp_status" \
                --pipestatus="${_omp_pipestatus[*]}" \
                --no-status="$_omp_no_status" \
                --execution-time="$_omp_execution_time" \
                --stack-count="$_omp_stack_count" \
                --terminal-width="${COLUMNS-0}" \
                $vim_mode_arg |
                tr -d '\0'
        )
    fi
    echo "${prompt@P}"
}

function _omp_get_secondary() {
    # Avoid unexpected expansions when we're generating the prompt below.
    shopt -u promptvars
    trap 'shopt -s promptvars' RETURN

    if shopt -oq posix; then
        # Disable in POSIX mode.
        echo '> '
    else
        echo "${_omp_secondary_prompt@P}"
    fi
}

function _omp_hook() {
    _omp_status=$? _omp_pipestatus=("${PIPESTATUS[@]}")

    if [[ -v BP_PIPESTATUS && ${#BP_PIPESTATUS[@]} -ge ${#_omp_pipestatus[@]} ]]; then
        _omp_pipestatus=("${BP_PIPESTATUS[@]}")
    fi

    _omp_stack_count=$((${#DIRSTACK[@]} - 1))

    _omp_execution_time=-1
    if [[ $_omp_start_time ]]; then
        local omp_now=$("$_omp_executable" get millis)
        _omp_execution_time=$((omp_now - _omp_start_time))
        _omp_no_status=false
    fi
    _omp_start_time=''

    if [[ ${_omp_pipestatus[-1]} != "$_omp_status" ]]; then
        _omp_pipestatus=("$_omp_status")
    fi

    set_poshcontext
    _omp_apply_cursor_shape
    _omp_set_cursor_position

    PS1='$(_omp_get_primary)'
    PS2='$(_omp_get_secondary)'

    # Ensure that command substitution works in a prompt string.
    shopt -s promptvars

    return $_omp_status
}

function _omp_install_hook() {
    local cmd
    local prompt_command

    for cmd in "${PROMPT_COMMAND[@]}"; do
        # skip initializing when we're already initialized
        if [[ $cmd = _omp_hook ]]; then
            return
        fi

        # check if the command starts with source, if so, do not add it again
        # this is done to avoid issues with sourcing the same file multiple times
        if [[ $cmd = source* ]]; then
            continue
        fi

        prompt_command+=("$cmd")
    done

    PROMPT_COMMAND=("${prompt_command[@]}" _omp_hook)
}

_omp_install_hook

# Daemon mode variables
_omp_daemon_mode=0
_omp_config=::CONFIG::
_omp_transient_prompt=''

# Vim mode variables
_omp_vim_mode=0
_omp_vim_mode_repaint=0
_omp_cursor_shape=0
_omp_cursor_blink=0

# Get current vim mode for segment template
function _omp_get_vim_mode() {
    if [[ -n "$BLE_VERSION" ]]; then
        case "$_ble_decode_keymap" in
            vi_nmap) echo "normal" ;;
            vi_imap) echo "insert" ;;
            vi_xmap|vi_smap) echo "visual" ;;
            vi_omap) echo "operator" ;;
            vi_cmap) echo "command" ;;
            *) echo "insert" ;;
        esac
    else
        echo "insert"
    fi
}

# Check if terminal handles cursor natively (Ghostty, Kitty)
function _omp_should_change_cursor() {
    [[ -n "$GHOSTTY_RESOURCES_DIR" ]] && return 1
    [[ -n "$KITTY_WINDOW_ID" ]] && return 1
    return 0
}

function _omp_apply_cursor_shape() {
    # Change cursor shape if enabled and terminal doesn't handle it natively
    if [[ "$_omp_cursor_shape" == "1" ]] && _omp_should_change_cursor; then
        local block_code=2
        local beam_code=6
        if [[ "$_omp_cursor_blink" == "1" ]]; then
            block_code=1
            beam_code=5
        fi
        case "$(_omp_get_vim_mode)" in
            normal|visual|operator|command)
                printf '\e[%s q' "$block_code"  # Block for normal/visual/operator/command mode
                ;;
            insert|*)
                printf '\e[%s q' "$beam_code"  # Beam for insert mode
                ;;
        esac
    fi
}

# Vim mode change handler for ble.sh
function _omp_ble_keymap_change() {
    _omp_apply_cursor_shape

    # In daemon mode, trigger async repaint with new vim mode
    if [[ $_omp_daemon_mode == 1 ]]; then
        _omp_daemon_render --repaint
    fi
}

function _omp_daemon_parse_line() {
    local line="$1"
    local type="${line%%:*}"
    local text="${line#*:}"

    case "$type" in
        primary)
            PS1="$text"
            ;;
        right)
            bleopt prompt_rps1="$text"
            ;;
        secondary)
            PS2="$text"
            ;;
        transient)
            _omp_transient_prompt="$text"
            ;;
    esac
}

function _omp_daemon_job() {
    local line
    local batch_complete=0
    while true; do
        batch_complete=0
        while IFS= read -r line; do
            _omp_daemon_parse_line "$line"
            if [[ $line == status:* ]]; then
                batch_complete=1
                ble/textarea#render
                if [[ $line == "status:complete" ]]; then
                    return
                fi
                break
            fi
        done
        if [[ $batch_complete -eq 0 ]]; then
            return
        fi
    done
}

# Async daemon render - used by both prompt hook and vim mode changes
# Pass --repaint for vim mode toggles (soft cancel, reuse computations)
function _omp_daemon_render() {
    local repaint_flag="$1"

    local vim_mode_arg=""
    if [[ $_omp_vim_mode == 1 ]]; then
        vim_mode_arg="--vim-mode=$(_omp_get_vim_mode)"
    fi

    # Run the render command in the background using ble.sh job system
    ble/util/job.start \
        "$_omp_executable" render \
            --config=$_omp_config \
            --shell=bash \
            --shell-version=$BASH_VERSION \
            --pwd=$PWD \
            --pid=$$ \
            --status=$_omp_status \
            --pipestatus=${_omp_pipestatus[*]} \
            --no-status=$_omp_no_status \
            --execution-time=$_omp_execution_time \
            --stack-count=$_omp_stack_count \
            --terminal-width=${COLUMNS-0} \
            --escape=false \
            $vim_mode_arg \
            $repaint_flag"
        _omp_daemon_job
}

function _omp_daemon_hook() {
    _omp_status=$? _omp_pipestatus=("${PIPESTATUS[@]}")

    if [[ -v BP_PIPESTATUS && ${#BP_PIPESTATUS[@]} -ge ${#_omp_pipestatus[@]} ]]; then
        _omp_pipestatus=("${BP_PIPESTATUS[@]}")
    fi

    _omp_stack_count=$((${#DIRSTACK[@]} - 1))

    _omp_execution_time=-1
    if [[ $_omp_start_time ]]; then
        local omp_now=$("$_omp_executable" get millis)
        _omp_execution_time=$((omp_now - _omp_start_time))
        _omp_no_status=false
    fi
    _omp_start_time=''

    if [[ ${_omp_pipestatus[-1]} != "$_omp_status" ]]; then
        _omp_pipestatus=("$_omp_status")
    fi

    set_poshcontext
    _omp_apply_cursor_shape
    _omp_set_cursor_position

    _omp_daemon_render
}

function enable_poshdaemon() {
    # Check for ble.sh
    if [[ -z "$BLE_VERSION" ]]; then
        return
    fi

    # Start daemon
    "$_omp_executable" daemon start --config="$_omp_config" --silent >/dev/null 2>&1 &

    _omp_daemon_mode=1

    # Remove standard hook and add daemon hook using blehook if possible, or PROMPT_COMMAND
    blehook PROMPT_COMMAND-=_omp_hook
    blehook PROMPT_COMMAND+=_omp_daemon_hook

    # Register vim mode keymap change hook
    blehook keymap_vi_load+=_omp_register_vim_hooks

    # Transient prompt configuration
    if [[ -n "$_omp_transient_prompt" ]]; then
        bleopt prompt_ps1_transient=always
        bleopt prompt_ps1_final='$_omp_transient_prompt'
    fi
}

function _omp_register_vim_hooks() {
    # Hook into ble.sh keymap changes
    ble/function#try ble/keymap:vi/invoke-hook keymap_enter _omp_ble_keymap_change
}

function enable_posh_vim_mode() {
    _omp_vim_mode=1

    # Register vim mode hooks if ble.sh is available
    if [[ -n "$BLE_VERSION" ]]; then
        blehook keymap_vi_load+=_omp_register_vim_hooks
    fi
}
