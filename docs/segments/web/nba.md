# NBA

## Segment Type

`nba`

## What

The NBA segment allows you to display the scheduling and score information for your favorite NBA team!

## Sample Configuration

In order to use the NBA segment, you need to provide a valid team [tri-code][tri-code] that you'd like to get data for
inside of the configuration. For example, if you'd like to get information for the Los Angeles Lakers, you'd need to use
the "LAL" tri-code.

This example uses "LAL" to get information for the Los Angeles Lakers. It also sets the foreground and background colors
to match the theming for the team. If you are interested in getting information about specific foreground and background
colors you could use for other teams, you can explore some of the color schemes [NBA team color schemes][color-schemes].

It is recommended that you set the HTTP timeout to a higher value than the normal default in case it takes some time to
gather the scoreboard information. In this case we have the http_timeout set to 1500.

```yaml
prompt:
  - segments: ["nba"]

nba:
  background: "#e9ac2f"
  foreground: "#8748dc"
  leading_diamond: ""
  style: "diamond"
  trailing_diamond: ""
  type: "nba"
  options:
    team: "LAL"
    http_timeout: 1500
```

## Options

- `team`
  - Type: `string`
  - Description: tri-code for the NBA team you want to get data for
- `days_offset`
  - Type: `int`
  - Default: `8`
  - Description: how many days in advance you wish to see that information for
- `http_timeout`
  - Type: `int`
  - Default: `20`
  - Description: How long do you want to wait before you want to see your prompt more than your sugar? I figure a half
    second is a good default

## Template

### Default Template

```template
 \udb82\udc06 {{ .HomeTeam}}{{ if .HasStats }} ({{.HomeTeamWins}}-{{.HomeTeamLosses}}){{ end }}{{ if .Started }}:{{.HomeScore}}{{ end }} vs {{ .AwayTeam}}{{ if .HasStats }} ({{.AwayTeamWins}}-{{.AwayTeamLosses}}){{ end }}{{ if .Started }}:{{.AwayScore}}{{ end }} | {{ if not .Started }}{{.GameDate}} | {{ end }}{{.Time}}
```

### Properties

- `.HomeTeam`
  - Type: `string`
  - Description: home team for the upcoming game
- `.AwayTeam`
  - Type: `string`
  - Description: away team for the upcoming game
- `.Time`
  - Type: `string`
  - Description: time (EST) that the upcoming game will start
- `.GameDate`
  - Type: `string`
  - Description: date the game will happen
- `.StartTimeUTC`
  - Type: `string`
  - Description: time (UTC) the game will start
- `.GameStatus`
  - Type: `integer`
  - Description: integer, 1 = scheduled, 2 = in progress, 3 = finished
- `.HomeScore`
  - Type: `int`
  - Description: score of the home team
- `.AwayScore`
  - Type: `int`
  - Description: score of the away team
- `.HomeTeamWins`
  - Type: `int`
  - Description: number of wins the home team currently has for the season
- `.HomeTeamLosses`
  - Type: `int`
  - Description: number of losses the home team currently has for the season
- `.AwayTeamWins`
  - Type: `int`
  - Description: number of wins the away team currently has for the season
- `.AwayTeamLosses`
  - Type: `int`
  - Description: number of losses the away team currently has for the season
- `.Started`
  - Type: `boolean`
  - Description: if the game was started or not
- `.HasStats`
  - Type: `boolean`
  - Description: if the game has game stats or not

[color-schemes]: https://teamcolorcodes.com/nba-team-color-codes/
[tri-code]: https://liaison.reuters.com/tools/sports-team-codes
