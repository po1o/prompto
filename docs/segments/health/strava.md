# Strava

## Segment Type

`strava`

## What

[Strava][strava] is a popular activity tracker for bike, run, and other training. The Strava segment shows your last
activity and can change color when it is time to get away from your computer and get active.

## Accessing your Strava data

To allow `prompto` to access your Strava data, grant access to read your public activities. This gives you an
access and a refresh token. Paste the tokens into your Strava segment configuration.

Click the following link to connect with Strava:

[Connect your Strava account][strava-connect]

This link still uses the legacy hosted redirect helper at `https://prompto.dev/api/auth`. If you run your own fork or
OAuth helper, replace that redirect URI with one you control before using it.

## Sample Configuration

This configuration sets the background green if you have an activity the last two days, orange if you have one last 5
days, and red otherwise. The `foreground_templates` example below could be set to just a single color, if that color is
visible against any of your backgrounds.

```yaml
prompt:
  - segments: ["strava"]

strava:
  type: "strava"
  style: "powerline"
  powerline_symbol: ""
  foreground: "#ffffff"
  background: "#000000"
  background_templates: ["{{ if gt .Hours 100 }}#dc3545{{ end }}", "{{ if and (lt .Hours 100) (gt .Hours 50) }}#ffc107{{ end }}", "{{ if lt .Hours 50 }}#28a745{{ end }}"]
  foreground_templates: ["{{ if gt .Hours 100 }}#FFFFFF{{ end }}", "{{ if and (lt .Hours 100) (gt .Hours 50) }}#343a40{{ end }}", "{{ if lt .Hours 50 }}#FFFFFF{{ end }}"]
  template: "  {{.Name}} {{.Ago}} {{.Icon}} "
  options:
    access_token: "11111111111111111"
    refresh_token: "1111111111111111"
    http_timeout: 1500
```

## Options

- `access_token`
  - Type: [`template`][templates]
  - Description: token from Strava login, see login link in section above.
- `refresh_token`
  - Type: [`template`][templates]
  - Description: token from Strava login, see login link in section above.
- `expires_in`
  - Type: `int`
  - Default: `0`
  - Description: the default timeout of the token from the Strava login
- `http_timeout`
  - Type: `int`
  - Default: `20`
  - Description: in milliseconds - how long do you want to wait before you want to see your prompt more than your strava
    data?
- `ride_icon`
  - Type: `string`
  - Default: `\uf206`
- `run_icon`
  - Type: `string`
  - Default: `\ue213`
- `skiing_icon`
  - Type: `string`
  - Default: `\ue213`
- `workout_icon`
  - Type: `string`
  - Default: `\ue213`
- `unknown_activity_icon`
  - Type: `string`
  - Default: `\ue213`

## Template

### Default Template

```template
 {{ if .Error }}{{ .Error }}{{ else }}{{ .Ago }}{{ end }}
```

### Properties

- `.ID`
  - Type: `time`
  - Description: The id of the entry
- `.DateString`
  - Type: `time`
  - Description: The timestamp of the entry
- `.Type`
  - Type: `string`
  - Description: Activity types as used in strava
- `.UtcOffset`
  - Type: `int`
  - Description: The UTC offset
- `.Hours`
  - Type: `int`
  - Description: Number of hours since last activity
- `.Name`
  - Type: `string`
  - Description: The name of the activity
- `.Duration`
  - Type: `float64`
  - Description: Total duration in seconds
- `.Distance`
  - Type: `float64`
  - Description: Total distance in meters
- `.DeviceWatts`
  - Type: `bool`
  - Description: Device has watts
- `.AverageWatts`
  - Type: `float64`
  - Description: Average watts
- `.WeightedAverageWatts`
  - Type: `float64`
  - Description: Weighted average watts
- `.AverageHeartRate`
  - Type: `float64`
  - Description: Average heart rate
- `.MaxHeartRate`
  - Type: `float64`
  - Description: Max heart rate
- `.KudosCount`
  - Type: `int`
  - Description: Kudos count
- `.Icon`
  - Type: `string`
  - Description: Activity based icon

Now, go out and have a fun ride or run!

[templates]: ../../configuration/templates.md
[strava]: http://www.strava.com/
[strava-connect]: https://www.strava.com/oauth/authorize?client_id=76033&response_type=code&redirect_uri=https://prompto.dev/api/auth&approval_prompt=force&scope=read,activity:read&state=strava
