Import-Module prompto-git
Import-Module PSFzf -ArgumentList 'Ctrl+t', 'Ctrl+r'
Import-Module z
Import-Module Terminal-Icons

Set-PSReadlineKeyHandler -Key Tab -Function MenuComplete

$env:PROMPTO_GIT_ENABLED=$true
prompto init pwsh | Invoke-Expression
