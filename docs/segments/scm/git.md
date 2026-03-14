# Git

## Segment Type

`git`

## What

Display git information when in a [Git][git] repository. Also works for subfolders. For maximum compatibility, make sure
your `git` executable is up-to-date (when branch or status information is incorrect for example).

## Sample Configuration

```yaml
prompt:
  - segments: ["git"]

git:
  type: "git"
  style: "powerline"
  powerline_symbol: ""
  foreground: "#193549"
  background: "#ffeb3b"
  background_templates: ["{{ if or (.Working.Changed) (.Staging.Changed) }}#FFEB3B{{ end }}", "{{ if and (gt .Ahead 0) (gt .Behind 0) }}#FFCC80{{ end }}", "{{ if gt .Ahead 0 }}#B388FF{{ end }}", "{{ if gt .Behind 0 }}#B388FB{{ end }}"]
  template: "{{ .UpstreamIcon }}{{ .HEAD }}{{if .BranchStatus }} {{ .BranchStatus }}{{ end }}{{ if .Working.Changed }}  {{ .Working.String }}{{ end }}{{ if and (.Working.Changed) (.Staging.Changed) }} |{{ end }}{{ if .Staging.Changed }}  {{ .Staging.String }}{{ end }}{{ if gt .StashCount 0 }}  {{ .StashCount }}{{ end }}"
  options:
    fetch_status: true
    fetch_upstream_icon: true
    untracked_modes:
      /Users/user/Projects/prompto/: "no"
    source: "cli"
    mapped_branches:
      feat/*: "🚀 "
      bug/*: "🐛 "
```

## Options

### Fetching information

As doing multiple git calls can slow down the prompt experience, we do not fetch information by default. You can set the
following options to `true` to enable fetching additional information (and populate the template).

- `fetch_status`
  - Type: `boolean`
  - Default: `false`
  - Description: fetch the local changes
- `fetch_push_status`
  - Type: `boolean`
  - Default: `false`
  - Description: fetch the push-remote ahead/behind information. Requires `fetch_status` to be enabled
- `ignore_status`
  - Type: `[]string`
  - Description: do not fetch status for these repo's. Uses the repo's root folder and same logic as the
    [exclude_folders][exclude_folders] property
- `fetch_upstream_icon`
  - Type: `boolean`
  - Default: `false`
  - Description: fetch upstream icon
- `fetch_bare_info`
  - Type: `boolean`
  - Default: `false`
  - Description: fetch bare repo info
- `fetch_user`
  - Type: [`User`](#user)
  - Default: `false`
  - Description: fetch the current configured user for the repository
- `untracked_modes`
  - Type: `map[string]string`
  - Description: map of repositories where to override the default [untracked files mode][untracked]. Supported values
    are `no`, `normal`, and `all`. For example, `"untracked_modes": { "/Users/me/repos/repo1": "no" }` overrides that
    repository while the default remains `normal` for everything else. Use `*` to override the mode for all repositories.
- `ignore_submodules`
  - Type: `map[string]string`
  - Description: map of repo's where to change the [--ignore-submodules][submodules] flag (`none`, `untracked`, `dirty`
    or `all`). For example `"ignore_submodules": { "/Users/me/repos/repo1": "all" }`. If you want to override for all
    repo's, use `*` to set the mode instead of the repo path
- `native_fallback`
  - Type: `boolean`
  - Default: `false`
  - Description: when set to `true` and `git.exe` is not available when inside a WSL2 shared Windows drive, we will
    fallback to the native `git` executable to fetch data. Not all information can be displayed in this case
- `status_formats`
  - Type: `map[string]string`
  - Description: a key, value map allowing to override how individual status items are displayed. For example,
    `"status_formats": { "Added": "Added: %d" }` will display the added count as `Added: 1` instead of `+1`. See the
    [Status](#status) section for available overrides.
- `source`
  - Type: `string`
  - Default: `cli`
  - Description: `cli`: fetch the information using the git CLI; `pwsh`: fetch the information from the
    [posh-git][poshgit] PowerShell Module
- `mapped_branches`
  - Type: `object`
  - Description: custom glyph/text for specific branches. You can use `*` at the end as a wildcard character for
    matching
- `branch_template`
  - Type: `string`
  - Description: a [template][templates] to format that branch name. You can use `{{ .Branch }}` as reference to the
    original branch name
- `disable_with_jj`
  - Type: `boolean`
  - Default: `false`
  - Description: disable the git segment in case of a [Jujutsu] collocated repository

### Icons

#### Branch

- `branch_icon`
  - Type: `string`
  - Default: `\uE0A0`
  - Description: the icon to use in front of the git branch name
- `branch_identical_icon`
  - Type: `string`
  - Default: `\u2261`
  - Description: the icon to display when remote and local are identical
- `branch_ahead_icon`
  - Type: `string`
  - Default: `\u2191`
  - Description: the icon to display when the local branch is ahead of its remote
- `branch_behind_icon`
  - Type: `string`
  - Default: `\u2193`
  - Description: the icon to display when the local branch is behind its remote
- `branch_gone_icon`
  - Type: `string`
  - Default: `\u2262`
  - Description: the icon to display when there's no remote branch

#### HEAD

- `commit_icon`
  - Type: `string`
  - Default: `\uF417`
  - Description: icon/text to display before the commit context (detached HEAD)
- `tag_icon`
  - Type: `string`
  - Default: `\uF412`
  - Description: icon/text to display before the tag context
- `rebase_icon`
  - Type: `string`
  - Default: `\uE728`
  - Description: icon/text to display before the context when in a rebase
- `cherry_pick_icon`
  - Type: `string`
  - Default: `\uE29B`
  - Description: icon/text to display before the context when doing a cherry-pick
- `revert_icon`
  - Type: `string`
  - Default: `\uF0E2`
  - Description: icon/text to display before the context when doing a revert
- `merge_icon`
  - Type: `string`
  - Default: `\uE727`
  - Description: icon/text to display before the merge context
- `no_commits_icon`
  - Type: `string`
  - Default: `\uF594`
  - Description: icon/text to display when there are no commits in the repo

#### Upstream

- `github_icon`
  - Type: `string`
  - Default: `\uF408`
  - Description: icon/text to display when the upstream is GitHub
- `gitlab_icon`
  - Type: `string`
  - Default: `\uF296`
  - Description: icon/text to display when the upstream is GitLab
- `bitbucket_icon`
  - Type: `string`
  - Default: `\uF171`
  - Description: icon/text to display when the upstream is Bitbucket
- `azure_devops_icon`
  - Type: `string`
  - Default: `\uEBE8`
  - Description: icon/text to display when the upstream is Azure DevOps
- `codecommit_icon`
  - Type: `string`
  - Default: `\uF270`
  - Description: icon/text to display when the upstream is AWS CodeCommit
- `codeberg_icon`
  - Type: `string`
  - Default: `\uF330`
  - Description: icon/text to display when the upstream is Codeberg
- `git_icon`
  - Type: `string`
  - Default: `\uE5FB`
  - Description: icon/text to display when the upstream is not known/mapped
- `upstream_icons`
  - Type: `map[string]string`
  - Description: a key, value map representing the remote URL (or a part of that URL) and icon to use in case the
    upstream URL contains the key. These get precedence over the standard icons

## Template

### Default Template

```template
{{ .HEAD }}{{if .BranchStatus }} {{ .BranchStatus }}{{ end }}{{ if .Working.Changed }} \uF044 {{ .Working.String }}{{ end }}{{ if and (.Staging.Changed) (.Working.Changed) }} |{{ end }}{{ if .Staging.Changed }} \uF046 {{ .Staging.String }}{{ end }}
```

### Properties

- `.RepoName`
  - Type: `string`
  - Description: the repo folder name
- `.Working`
  - Type: `Status`
  - Description: changes in the worktree (see below)
- `.Staging`
  - Type: `Status`
  - Description: staged changes in the work tree (see below)
- `.HEAD`
  - Type: `string`
  - Description: the current HEAD context (branch/rebase/merge/...)
- `.Ref`
  - Type: `string`
  - Description: the current HEAD reference (branch/tag/...)
- `.Behind`
  - Type: `int`
  - Description: commits behind of upstream
- `.Ahead`
  - Type: `int`
  - Description: commits ahead of upstream
- `.PushBehind`
  - Type: `int`
  - Description: commits behind of push remote
- `.PushAhead`
  - Type: `int`
  - Description: commits ahead of push remote
- `.BranchStatus`
  - Type: `string`
  - Description: the current branch context (ahead/behind string representation)
- `.Upstream`
  - Type: `string`
  - Description: the upstream name (remote)
- `.UpstreamGone`
  - Type: `boolean`
  - Description: whether the upstream is gone (no remote)
- `.UpstreamIcon`
  - Type: `string`
  - Description: the upstream icon (based on the icons above)
- `.UpstreamURL`
  - Type: `string`
  - Description: the upstream URL for use in [hyperlinks][hyperlinks] in templates: `{{ url .UpstreamIcon .UpstreamURL
    }}`
- `.RawUpstreamURL`
  - Type: `string`
  - Description: the raw upstream URL (not cleaned up for display)
- `.Hash`
  - Type: `string`
  - Description: the full commit hash
- `.ShortHash`
  - Type: `string`
  - Description: the short commit hash (7 characters)
- `.StashCount`
  - Type: `int`
  - Description: the stash count
- `.WorktreeCount`
  - Type: `int`
  - Description: the worktree count
- `.IsWorkTree`
  - Type: `boolean`
  - Description: if in a worktree repo or not
- `.IsBare`
  - Type: `boolean`
  - Description: if in a bare repo or not, only set when `fetch_bare_info` is set to `true`
- `.Dir`
  - Type: `string`
  - Description: the repository's root directory
- `.RelativeDir`
  - Type: `string`
  - Description: the current directory relative to the root directory
- `.Kraken`
  - Type: `string`
  - Description: a link to the current HEAD in [GitKraken][kraken-ref] for use in [hyperlinks][hyperlinks] in templates
    `{{ url .HEAD .Kraken }}`
- `.Commit`
  - Type: `Commit`
  - Description: HEAD commit information (see below)
- `.Detached`
  - Type: `boolean`
  - Description: true when the head is detached
- `.Merge`
  - Type: `boolean`
  - Description: true when in a merge
- `.Rebase`
  - Type: `Rebase`
  - Description: contains the relevant information when in a rebase
- `.CherryPick`
  - Type: `boolean`
  - Description: true when in a cherry pick
- `.Revert`
  - Type: `boolean`
  - Description: true when in a revert
- `.User`
  - Type: `User`
  - Description: the current configured user (requires `fetch_user` to be enabled)
- `.Remotes`
  - Type: `map[string]string`
  - Description: a map of remote names to their URLs
- `.LatestTag`
  - Type: `string`
  - Description: the latest tag name

#### Status

- `.Unmerged`
  - Type: `int`
  - Description: number of unmerged changes
- `.Deleted`
  - Type: `int`
  - Description: number of deleted changes
- `.Added`
  - Type: `int`
  - Description: number of added changes
- `.Modified`
  - Type: `int`
  - Description: number of modified changes
- `.Untracked`
  - Type: `int`
  - Description: number of untracked changes
- `.Changed`
  - Type: `boolean`
  - Description: if the status contains changes or not
- `.String`
  - Type: `string`
  - Description: a string representation of the changes above

Local changes use the following syntax:

- `x`
  - Description: Unmerged
- `-`
  - Description: Deleted
- `+`
  - Description: Added
- `~`
  - Description: Modified
- `?`
  - Description: Untracked

#### Commit

- `.Author`
  - Type: `User`
  - Description: the author of the commit (see below)
- `.Committer`
  - Type: `User`
  - Description: the committer of the commit (see below)
- `.Subject`
  - Type: `string`
  - Description: the commit subject
- `.Timestamp`
  - Type: `time.Time`
  - Description: the commit timestamp
- `.Sha`
  - Type: `string`
  - Description: the commit SHA1
- `.Refs`
  - Type: `Refs`
  - Description: the commit references

##### User

- `.Name`
  - Type: `string`
  - Description: the user's name
- `.Email`
  - Type: `string`
  - Description: the user's email

##### Refs

- `.Heads`
  - Type: `[]string`
  - Description: branches
- `.Tags`
  - Type: `[]string`
  - Description: commit's tags
- `.Remotes`
  - Type: `[]string`
  - Description: remote references

As these are arrays of strings, you can join them using the `join` function:

```template
{{ join ", " .Commit.Refs.Tags }}
```

#### Rebase

- `.Current`
  - Type: `int`
  - Description: the current rebase step
- `.Total`
  - Type: `int`
  - Description: the total number of rebase steps
- `.HEAD`
  - Type: `string`
  - Description: the current HEAD
- `.Onto`
  - Type: `string`
  - Description: the branch we're rebasing onto

## posh-git

If you want to display the default [posh-git][poshgit] output, **do not** use this segment but add the following snippet
after initializing `prompto` in your `$PROFILE`:

```powershell
function Set-PoshGitStatus {
    $global:GitStatus = Get-GitStatus
    $env:PROMPTO_GIT_STRING = Write-GitStatus -Status $global:GitStatus
}
New-Alias -Name 'Set-PoshContext' -Value 'Set-PoshGitStatus' -Scope Global -Force
```

You can then use the `PROMPTO_GIT_STRING` environment variable in a [text segment][text]:

```yaml
type: "text"
template: "{{ if .Env.PROMPTO_GIT_STRING }} {{ .Env.PROMPTO_GIT_STRING }} {{ end }}"
```

[git]: https://git-scm.com/
[poshgit]: https://github.com/dahlbyk/posh-git
[templates]: ../../configuration/templates.md
[hyperlinks]: ../../configuration/templates.md#custom-helper-functions
[untracked]: https://git-scm.com/docs/git-status#Documentation/git-status.txt---untracked-filesltmodegt
[submodules]: https://git-scm.com/docs/git-status#Documentation/git-status.txt---ignore-submodulesltwhengt
[kraken-ref]: https://www.gitkraken.com/invite/nQmDPR9D
[text]: ../system/text.md
[exclude_folders]: ../../configuration/segments.md#folder-filters
[Jujutsu]: https://www.jj-vcs.dev/
