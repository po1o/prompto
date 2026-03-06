$POWERLINE_COMMAND = "prompto"
$PROMPTO_SHELL = "xonsh"
$PROMPTO_SHELL_VERSION = $XONSH_VERSION

# disable all known python virtual environment prompts
$VIRTUAL_ENV_DISABLE_PROMPT = 1
$PYENV_VIRTUALENV_DISABLE_PROMPT = 1

_prompto_executable = ::PROMPTO::
_prompto_history_length = 0

def _prompto_get_context():
    global _prompto_history_length
    status = 0
    duration = -1

    if __xonsh__.history:
        last_cmd = __xonsh__.history[-1]
        if last_cmd:
            status = last_cmd.rtn

        history_length = len(__xonsh__.history)
        if history_length != _prompto_history_length:
            _prompto_history_length = history_length
            duration = round((last_cmd.ts[1] - last_cmd.ts[0]) * 1000)

    return status, duration

def _prompto_get_prompt(type: str, *args: str):
    status, duration = _prompto_get_context()
    return $(
        @(_prompto_executable) render @(type) \
            --shell=xonsh \
            --shell-version=$XONSH_VERSION \
            --status=@(status) \
            --execution-time=@(duration) \
            @(args)
    )

def _prompto_get_primary():
    return _prompto_get_prompt('primary')

def _prompto_get_right():
    return _prompto_get_prompt('right')

$PROMPT = _prompto_get_primary
# When the primary prompt has multiple lines, the right prompt is always displayed on the first line, which is inconsistent with other supported shells.
# The behavior is controlled by Xonsh, and there is no way to change it.
$RIGHT_PROMPT = _prompto_get_right
