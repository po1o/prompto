export POSH_SHELL='zsh'
export POSH_SHELL_VERSION=$ZSH_VERSION
export POWERLINE_COMMAND='oh-my-posh'
export CONDA_PROMPT_MODIFIER=false
export ZLE_RPROMPT_INDENT=0
export OSTYPE=$OSTYPE

# disable all known python virtual environment prompts
export VIRTUAL_ENV_DISABLE_PROMPT=1
export PYENV_VIRTUALENV_DISABLE_PROMPT=1

_omp_executable=::OMP::
_omp_config=::CONFIG::
_omp_tooltip_command=''

# switches to enable/disable features
_omp_cursor_positioning=0
_omp_ftcs_marks=0

# set secondary prompt
_omp_secondary_prompt=''

function _omp_set_cursor_position() {
  # not supported in Midnight Commander
  # see https://github.com/JanDeDobbeleer/oh-my-posh/issues/3415
  if [[ $_omp_cursor_positioning == 0 ]] || [[ -v MC_SID ]]; then
    return
  fi

  local oldstty=$(stty -g)
  stty raw -echo min 0

  local pos
  echo -en '\033[6n' >/dev/tty
  read -r -d R pos
  pos=${pos:2} # strip off the esc-[
  local parts=(${(s:;:)pos})

  stty $oldstty

  export POSH_CURSOR_LINE=${parts[1]}
  export POSH_CURSOR_COLUMN=${parts[2]}
}

# template function for context loading
function set_poshcontext() {
  return
}

function _omp_preexec() {
  if [[ $_omp_ftcs_marks == 1 ]]; then
    printf '\033]133;C\007'
  fi

  _omp_start_time=$($_omp_executable get millis)
}

function _omp_precmd() {
  if [[ -z $_omp_secondary_prompt ]]; then
    _omp_secondary_prompt=$($_omp_executable print secondary --shell=zsh)
  fi

  _omp_status=$?
  _omp_pipestatus=(${pipestatus[@]})
  _omp_job_count=${#jobstates}
  _omp_stack_count=${#dirstack[@]}
  _omp_execution_time=-1
  _omp_no_status=true
  _omp_tooltip_command=''

  if [ $_omp_start_time ]; then
    local omp_now=$($_omp_executable get millis)
    _omp_execution_time=$(($omp_now - $_omp_start_time))
    _omp_no_status=false
  fi

  if [[ ${_omp_pipestatus[-1]} != "$_omp_status" ]]; then
    _omp_pipestatus=("$_omp_status")
  fi

  set_poshcontext
  _omp_apply_cursor_shape
  _omp_set_cursor_position

  # We do this to avoid unexpected expansions in a prompt string.
  unsetopt PROMPT_SUBST
  unsetopt PROMPT_BANG

  # Ensure that escape sequences work in a prompt string.
  setopt PROMPT_PERCENT

  PS2=$_omp_secondary_prompt
  eval "$(_omp_get_prompt primary --eval)"

  unset _omp_start_time
}

# add hook functions
autoload -Uz add-zsh-hook
add-zsh-hook precmd _omp_precmd
add-zsh-hook preexec _omp_preexec

# Prevent incorrect behaviors when the initialization is executed twice in current session.
function _omp_cleanup() {
  local omp_widgets=(
    self-insert
    zle-line-init
  )
  local widget
  for widget in "${omp_widgets[@]}"; do
    if [[ ${widgets[._omp_original::$widget]} ]]; then
      # Restore the original widget.
      zle -A ._omp_original::$widget $widget
    elif [[ ${widgets[$widget]} = user:_omp_* ]]; then
      # Delete the OMP-defined widget.
      zle -D $widget
    fi
  done
}
_omp_cleanup
unset -f _omp_cleanup

function _omp_get_prompt() {
  local type=$1
  local args=("${@[2,-1]}")
  local vim_mode_arg=""
  if [[ $_omp_vim_mode == 1 ]]; then
    vim_mode_arg="--vim-mode=$(_omp_get_vim_mode)"
  fi
  $_omp_executable print $type \
    --save-cache \
    --shell=zsh \
    --shell-version=$ZSH_VERSION \
    --status=$_omp_status \
    --pipestatus="${_omp_pipestatus[*]}" \
    --no-status=$_omp_no_status \
    --execution-time=$_omp_execution_time \
    --job-count=$_omp_job_count \
    --stack-count=$_omp_stack_count \
    --terminal-width="${COLUMNS-0}" \
    $vim_mode_arg \
    ${args[@]}
}

function _omp_render_tooltip() {
  if [[ $KEYS != ' ' ]]; then
    return
  fi

  setopt local_options no_shwordsplit

  # Get the first word of command line as tip.
  local tooltip_command=${${(MS)BUFFER##[[:graph:]]*}%%[[:space:]]*}

  # Ignore an empty/repeated tooltip command.
  if [[ -z $tooltip_command ]] || [[ $tooltip_command = "$_omp_tooltip_command" ]]; then
    return
  fi

  _omp_tooltip_command="$tooltip_command"
  local tooltip=$(_omp_get_prompt tooltip --command="$tooltip_command")
  if [[ -z $tooltip ]]; then
    return
  fi

  RPROMPT=$tooltip
  zle .reset-prompt
}

function _omp_zle-line-init() {
  [[ $CONTEXT == start ]] || return 0

  # Start regular line editor.
  (( $+zle_bracketed_paste )) && print -r -n - $zle_bracketed_paste[1]
  zle .recursive-edit
  local -i ret=$?
  (( $+zle_bracketed_paste )) && print -r -n - $zle_bracketed_paste[2]

  if [[ $_omp_daemon_mode == 1 ]]; then
    # Always use daemon-provided transient prompt in daemon mode.
    # If it's empty (e.g. first render), fall back to an empty prompt instead of CLI rendering.
    PS1=${_omp_transient_prompt-}
  else
    # We need this workaround because when the `filler` is set,
    # there will be a redundant blank line below the transient prompt if the input is empty.
    local terminal_width_option
    if [[ -z $BUFFER ]]; then
      terminal_width_option="--terminal-width=$((${COLUMNS-0} - 1))"
    fi
    eval "$(_omp_get_prompt transient --eval $terminal_width_option)"
  fi
  zle .reset-prompt

  if ((ret)); then
    # TODO (fix): this is not equal to sending a SIGINT, since the status code ($?) is set to 1 instead of 130.
    zle .send-break
  fi

  # Exit the shell if we receive EOT.
  if [[ $KEYS == $'\4' ]]; then
    exit
  fi

  zle .accept-line
  return $ret
}

# Helper function for calling a widget before the specified OMP function.
function _omp_call_widget() {
  # The name of the OMP function.
  local omp_func=$1
  # The remainder are the widget to call and potential arguments.
  shift

  zle "$@" && shift 2 && $omp_func "$@"
}

# Create a widget with the specified OMP function.
# An existing widget will be preserved and decorated with the function.
function _omp_create_widget() {
  # The name of the widget to create/decorate.
  local widget=$1
  # The name of the OMP function.
  local omp_func=$2

  case ${widgets[$widget]:-''} in
  # Already decorated: do nothing.
  user:_omp_decorated_*) ;;

  # Non-existent: just create it.
  '')
    zle -N $widget $omp_func
    ;;

  # User-defined or builtin: backup and decorate it.
  *)
    # Back up the original widget. The leading dot in widget name is to work around bugs when used with zsh-syntax-highlighting in Zsh v5.8 or lower.
    zle -A $widget ._omp_original::$widget
    eval "_omp_decorated_${(q)widget}() { _omp_call_widget ${(q)omp_func} ._omp_original::${(q)widget} -- \"\$@\" }"
    zle -N $widget _omp_decorated_$widget
    ;;
  esac
}

# Daemon mode variables
_omp_daemon_mode=0
_omp_daemon_fd=
_omp_transient_prompt=

# Vim mode variables
_omp_vim_mode=0
_omp_vim_mode_repaint=0
_omp_cursor_shape=0
_omp_cursor_blink=0

# Check if terminal handles cursor natively (Ghostty, Kitty)
function _omp_should_change_cursor() {
  [[ -n "$GHOSTTY_RESOURCES_DIR" ]] && return 1
  [[ -n "$KITTY_WINDOW_ID" ]] && return 1
  return 0
}

# Get current vim mode for segment template
function _omp_get_vim_mode() {
  case $KEYMAP in
    vicmd) echo "normal" ;;
    viins|main) echo "insert" ;;
    visual) echo "visual" ;;
    viopp) echo "operator" ;;
    *) echo "insert" ;;
  esac
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
    case $KEYMAP in
      vicmd)
        print -n "\e[${block_code} q"  # Block for normal mode
        ;;
      viins|main|*)
        print -n "\e[${beam_code} q"  # Beam for insert mode
        ;;
    esac
  fi
}

# Vim mode keymap change handler
function _omp_zle-keymap-select() {
  _omp_apply_cursor_shape

  # In daemon mode, trigger async repaint with new vim mode
  if [[ $_omp_daemon_mode == 1 ]]; then
    _omp_daemon_render --repaint
  fi

  # Trigger prompt repaint
  zle .reset-prompt
}

function _omp_daemon_precmd() {
  _omp_status=$?
  _omp_pipestatus=(${pipestatus[@]})
  _omp_job_count=${#jobstates}
  _omp_stack_count=${#dirstack[@]}
  _omp_execution_time=-1
  _omp_no_status=true
  _omp_tooltip_command=''

  if [ $_omp_start_time ]; then
    local omp_now=$($_omp_executable get millis)
    _omp_execution_time=$(($omp_now - $_omp_start_time))
    _omp_no_status=false
  fi

  if [[ ${_omp_pipestatus[-1]} != "$_omp_status" ]]; then
    _omp_pipestatus=("$_omp_status")
  fi

  set_poshcontext
  _omp_apply_cursor_shape
  _omp_set_cursor_position

  unsetopt PROMPT_SUBST
  unsetopt PROMPT_BANG
  setopt PROMPT_PERCENT

  PS2=$_omp_secondary_prompt

  _omp_daemon_render
  unset _omp_start_time
}

# Async daemon render - used by both precmd and vim mode changes
# Pass --repaint for vim mode toggles (soft cancel, reuse computations)
function _omp_daemon_render() {
  local repaint_flag=$1

  # Clean up any existing fd handler from previous render
  if [[ -n $_omp_daemon_fd ]]; then
    zle -F $_omp_daemon_fd
    exec {_omp_daemon_fd}<&-
    _omp_daemon_fd=
  fi

  local vim_mode_arg=""
  if [[ $_omp_vim_mode == 1 ]]; then
    vim_mode_arg="--vim-mode=$(_omp_get_vim_mode)"
  fi

  local fd
  exec {fd}< <($_omp_executable render \
    --config=$_omp_config \
    --shell=zsh \
    --shell-version=$ZSH_VERSION \
    --pwd="$PWD" \
    --pid=$$ \
    --status=$_omp_status \
    --pipestatus="${_omp_pipestatus[*]}" \
    --no-status=$_omp_no_status \
    --execution-time=$_omp_execution_time \
    --job-count=$_omp_job_count \
    --stack-count=$_omp_stack_count \
    --terminal-width="${COLUMNS-0}" \
    $vim_mode_arg \
    $repaint_flag \
    2>/dev/null)

  # Read first batch synchronously (partial results after daemon timeout)
  local line batch_complete=0
  while [[ $batch_complete -eq 0 ]] && IFS= read -r line <&$fd; do
    _omp_daemon_parse_line "$line"
    if [[ $line == status:* ]]; then
      batch_complete=1
      if [[ $line == "status:complete" ]]; then
        # All done, close fd
        exec {fd}<&-
        return
      fi
    fi
  done

  # More updates may come - register fd handler for async streaming
  _omp_daemon_fd=$fd
  zle -F $fd _omp_daemon_handler
}

function _omp_daemon_parse_line() {
  local line=$1
  local type=${line%%:*}
  local text=${line#*:}

  case $type in
    primary)
      PS1=$text
      ;;
    right)
      RPROMPT=$text
      ;;
    secondary)
      PS2=$text
      ;;
    transient)
      _omp_transient_prompt=$text
      ;;
  esac
}

function _omp_daemon_handler() {
  local fd=$1
  local line batch_complete=0

  # Read until we see a status line (daemon always ends a batch with status:*).
  while [[ $batch_complete -eq 0 ]] && IFS= read -r line <&$fd; do
    _omp_daemon_parse_line "$line"
    if [[ $line == status:* ]]; then
      batch_complete=1
    fi
  done

  # If we read at least one status line, repaint
  if [[ $batch_complete -eq 1 ]]; then
    zle .reset-prompt
  fi

  # If the stream ended or we saw the final status, clean up the fd.
  if [[ $batch_complete -eq 0 ]] || [[ $line == "status:complete" ]]; then
    zle -F $fd
    exec {fd}<&-
    _omp_daemon_fd=
  fi
}

function enable_poshdaemon() {
  # Start daemon if not running
  $_omp_executable daemon start --config=$_omp_config --silent >/dev/null 2>&1 &!

  # Replace precmd with daemon version
  _omp_daemon_mode=1
  add-zsh-hook -d precmd _omp_precmd
  add-zsh-hook precmd _omp_daemon_precmd
}

function enable_poshtooltips() {
  local widget=${$(bindkey ' '):2}

  if [[ -z $widget ]]; then
    widget=self-insert
  fi

  _omp_create_widget $widget _omp_render_tooltip
}

# Set up vim mode keybindings (oh-my-zsh style)
function _omp_setup_vim_keybindings() {
  # Reduce mode switching delay (default is 40 = 400ms)
  KEYTIMEOUT=15

  # Insert mode: restore common emacs-style keybindings
  bindkey -M viins '^P' up-line-or-history
  bindkey -M viins '^N' down-line-or-history
  bindkey -M viins '^W' backward-kill-word
  bindkey -M viins '^H' backward-delete-char
  bindkey -M viins '^?' backward-delete-char
  bindkey -M viins '^A' beginning-of-line
  bindkey -M viins '^E' end-of-line
  bindkey -M viins '^R' history-incremental-search-backward
  bindkey -M viins '^S' history-incremental-search-forward

  # Normal mode: edit command line in $EDITOR
  autoload -Uz edit-command-line
  zle -N edit-command-line
  bindkey -M vicmd 'vv' edit-command-line
}

# legacy functions
function enable_poshtransientprompt() {}

_omp_custom_dir=${OMP_CUSTOM:-}
if [[ -z $_omp_custom_dir ]]; then
  # Avoid double-sourcing if oh-my-zsh is managing ZSH_CUSTOM.
  if [[ -n $ZSH ]] && [[ -f $ZSH/oh-my-zsh.sh ]]; then
    _omp_custom_dir=
  else
    _omp_custom_dir=$ZSH_CUSTOM
  fi
fi

if [[ -n $_omp_custom_dir ]] && [[ -d $_omp_custom_dir ]]; then
  for script in $_omp_custom_dir/*.zsh(N); do
    source $script
  done
fi
