# Python

## Segment Type

`python`

## What

Display the currently active [Python][python] version and [virtualenv]. Supports [conda], virtualenv and pyenv (if
python points to pyenv shim).

## Sample Configuration

```yaml
prompt:
  - segments: ["python"]

python:
  type: "python"
  style: "powerline"
  powerline_symbol: ""
  foreground: "#ffd43b"
  background: "#306998"
  template: "  {{ .Full }} "
```

## Options

- `home_enabled`
  - Type: `boolean`
  - Default: `false`
  - Description: display the segment in the HOME folder or not
- `fetch_virtual_env`
  - Type: `boolean`
  - Default: `true`
  - Description: fetch the name of the virtualenv or not
- `display_default`
  - Type: `boolean`
  - Default: `true`
  - Description: show the name of the virtualenv when it's default (`system`, `base`) or not
- `fetch_version`
  - Type: `boolean`
  - Default: `true`
  - Description: fetch the python version
- `cache_duration`
  - Type: `string`
  - Default: `none`
  - Description: how long to cache the version. Use values like `30s`, `5m`, or `1h`. Use `none` to disable caching
- `missing_command_text`
  - Type: `string`
  - Description: text to display when the command is missing
- `display_mode`
  - Type: `string`
  - Default: `environment`
  - Description: `always`: the segment is always displayed; `files`: the segment is only displayed when file
    `extensions` listed are present; `environment`: the segment is only displayed when in a virtual environment;
    `context`: displays the segment when the environment or files is active
- `version_url_template`
  - Type: `string`
  - Description: a template that builds the URL of the version information or release notes
- `extensions`
  - Type: `[]string`
  - Default: `*.py, *.ipynb, pyproject.toml, venv.bak`
  - Description: allows to override the default list of file extensions to validate
- `folders`
  - Type: `[]string`
  - Default: `.venv, venv, virtualenv, venv-win, pyenv-win`
  - Description: allows to override the list of folder names to validate
- `tooling`
  - Type: `[]string`
  - Default: `pyenv, python, python3, py`
  - Description: the tooling to use for fetching the version. Available options: `pyenv`, `python`, `python3`, `py`,
    `uv`
- `folder_name_fallback`
  - Type: `boolean`
  - Default: `true`
  - Description: instead of `default_venv_names` (case sensitive), use the parent folder name as the virtual
    environment's name or not
- `default_venv_names`
  - Type: `[]string`
  - Default: `.venv, venv`
  - Description: allows to override the list of environment's name replaced when `folder_name_fallback` is `true`

## Template

### Default Template

```template
 {{ if .Error }}{{ .Error }}{{ else }}{{ if .Venv }}{{ .Venv }} {{ end }}{{ .Full }}{{ end }}
```

### Properties

- `.Venv`
  - Type: `string`
  - Description: the virtual environment name (if present)
- `.Full`
  - Type: `string`
  - Description: the full version
- `.Major`
  - Type: `string`
  - Description: major number
- `.Minor`
  - Type: `string`
  - Description: minor number
- `.Patch`
  - Type: `string`
  - Description: patch number
- `.URL`
  - Type: `string`
  - Description: URL of the version info / release notes
- `.Error`
  - Type: `string`
  - Description: error encountered when fetching the version string

[python]: https://www.python.org/
[virtualenv]: https://virtualenv.pypa.io/
[conda]: https://conda.org/
