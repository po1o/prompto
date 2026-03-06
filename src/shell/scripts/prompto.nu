# make sure we have the right prompt render correctly
if ($env.config? | is-not-empty) {
    $env.config = ($env.config | upsert render_right_prompt_on_last_line true)
}

$env.POWERLINE_COMMAND = 'prompto'
$env.PROMPT_INDICATOR = ""
$env.PROMPTO_SESSION_ID = "::SESSION_ID::"
$env.PROMPTO_SHELL = "nu"
$env.PROMPTO_SHELL_VERSION = (version | get version)

# disable all known python virtual environment prompts
$env.VIRTUAL_ENV_DISABLE_PROMPT = 1
$env.PYENV_VIRTUALENV_DISABLE_PROMPT = 1

let _prompto_executable: string = (echo ::PROMPTO::)
$env._prompto_daemon_mode = false
$env._prompto_current_prompt = ""
$env._prompto_current_rprompt = ""
$env._prompto_current_transient = ""
$env._prompto_current_secondary = ""

def enable_prompto_daemon [] {
    $env._prompto_daemon_mode = true
}

# PROMPTS

def --wrapped _prompto_get_prompt [
    type: string,
    ...args: string
] {
    mut execution_time = -1
    mut no_status = true
    # We have to do this because the initial value of `$env.CMD_DURATION_MS` is always `0823`, which is an official setting.
    # See https://github.com/nushell/nushell/discussions/6402#discussioncomment-3466687.
    if $env.CMD_DURATION_MS != '0823' {
        $execution_time = $env.CMD_DURATION_MS
        $no_status = false
    }

    (
        ^$_prompto_executable render $type
            --shell=nu
            $"--shell-version=($env.PROMPTO_SHELL_VERSION)"
            $"--status=($env.LAST_EXIT_CODE)"
            $"--no-status=($no_status)"
            $"--execution-time=($execution_time)"
            $"--terminal-width=((term size).columns)"
            $"--job-count=(job list | length)"
            ...$args
    )
}

def _prompto_daemon_render [clear: bool] {
    mut execution_time = -1
    mut no_status = true
    if $env.CMD_DURATION_MS != '0823' {
        $execution_time = $env.CMD_DURATION_MS
        $no_status = false
    }

    for line in (
        ^$_prompto_executable render
            --shell=nu
            $"--shell-version=($env.PROMPTO_SHELL_VERSION)"
            $"--status=($env.LAST_EXIT_CODE)"
            $"--no-status=($no_status)"
            $"--execution-time=($execution_time)"
            $"--terminal-width=((term size).columns)"
            $"--job-count=(job list | length)"
            $"--cleared=($clear)"
            | lines
    ) {
        if not ($line | str contains ":") {
            continue
        }

        let key = ($line | split row ":" | first)
        let value = ($line | str replace --regex '^[^:]*:' '')

        if $key == "primary" {
            $env._prompto_current_prompt = $value
            continue
        }

        if $key == "right" {
            $env._prompto_current_rprompt = $value
            continue
        }

        if $key == "transient" {
            $env._prompto_current_transient = $value
            continue
        }

        if $key == "secondary" {
            $env._prompto_current_secondary = $value
            continue
        }

        if $key == "status" {
            break
        }
    }
}

$env.PROMPT_MULTILINE_INDICATOR = (
    ^$_prompto_executable render secondary
        --shell=nu
        $"--shell-version=($env.PROMPTO_SHELL_VERSION)"
)

$env.PROMPT_COMMAND = {||
    # hack to set the cursor line to 1 when the user clears the screen
    # this obviously isn't bulletproof, but it's a start
    mut clear = false
    if $nu.history-enabled {
        $clear = (history | is-empty) or ((history | last 1 | get 0.command) == "clear")
    }

    if ($env.SET_POSHCONTEXT? | is-not-empty) {
        do --env $env.SET_POSHCONTEXT
    }

    if $env._prompto_daemon_mode {
        _prompto_daemon_render $clear
        $env._prompto_current_prompt
    } else {
        _prompto_get_prompt primary $"--cleared=($clear)"
    }
}

$env.PROMPT_COMMAND_RIGHT = {||
    if $env._prompto_daemon_mode {
        $env._prompto_current_rprompt
    } else {
        _prompto_get_prompt right
    }
}
