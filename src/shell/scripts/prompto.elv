set-env PROMPTO_SHELL elvish
set-env PROMPTO_SHELL_VERSION $version
set-env POWERLINE_COMMAND prompto

# disable all known python virtual environment prompts
set-env VIRTUAL_ENV_DISABLE_PROMPT 1
set-env PYENV_VIRTUALENV_DISABLE_PROMPT 1

var _prompto_executable = (external ::PROMPTO::)
var _prompto_status = 0
var _prompto_no_status = 1
var _prompto_execution_time = -1
var _prompto_terminal_width = ($_prompto_executable get width)

# A flag to simulate a mutex.
var _prompto_primary_ready = $false

fn _omp-after-readline-hook {|_|
    set _prompto_execution_time = -1

    # Getting the terminal width can fail inside a prompt function, so we do this here.
    set _prompto_terminal_width = ($_prompto_executable get width)
}

fn _omp-after-command-hook {|m|
    # The command execution time should not be available in the first prompt.
    if (== $_prompto_no_status 0) {
        set _prompto_execution_time = (printf %.0f (* $m[duration] 1000))
    }

    set _prompto_no_status = 0

    var error = $m[error]
    if (is $error $nil) {
        set _prompto_status = 0
    } else {
        try {
            set _prompto_status = $error[reason][exit-status]
        } catch {
            # built-in commands don't have a status code.
            set _prompto_status = 1
        }
    }
}

fn _prompto_get_prompt {|type @arguments|
    $_prompto_executable render $type ^
        --shell=elvish ^
        --shell-version=$E:PROMPTO_SHELL_VERSION ^
        --status=$_prompto_status ^
        --no-status=$_prompto_no_status ^
        --execution-time=$_prompto_execution_time ^
        --terminal-width=$_prompto_terminal_width ^
        $@arguments
}

set edit:after-readline = [ $@edit:after-readline $_omp-after-readline-hook~ ]
set edit:after-command = [ $@edit:after-command $_omp-after-command-hook~ ]

set edit:prompt = {||
    # Workaround to avoid a race condition in cache access.
    while $true {
        if (not $_prompto_primary_ready) {
            break
        }
    }

    _prompto_get_prompt primary

    # Now it can start to render the right prompt.
    set _prompto_primary_ready = $true
}

set edit:rprompt = {||
    # Workaround to avoid a race condition in cache access.
    while $true {
        if $_prompto_primary_ready {
            break
        }
    }

    _prompto_get_prompt right

    # A "prompt rendering period" ends.
    set _prompto_primary_ready = $false
}
