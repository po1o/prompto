# Withings

## Segment Type

`withings`

## What

The [Withings][withings] health ecosystem of connected devices & apps is designed to improve daily wellbeing and
long-term health.

## Accessing your Withings data

To allow `prompto` to access your Withings data, grant access to read your public activities. This gives you
an access and a refresh token. Paste the tokens into your Withings segment configuration.

Click the following link to connect with Withings:

[Connect your Withings
account](https://account.withings.com/oauth2_user/authorize2?client_id=93675962e88ddfe53f83c0c900558f72174e0ac70ccfb57e48053530c7e6e494&response_type=code&redirect_uri=https://prompto.dev/api/auth&scope=user.activity,user.metrics&state=withings)

This link still uses the legacy hosted redirect helper at `https://prompto.dev/api/auth`. If you run your own fork or
OAuth helper, replace that redirect URI with one you control before using it.

## Sample Configuration

```yaml
prompt:
  - segments: ["withings"]

withings:
  type: "withings"
  style: "powerline"
  powerline_symbol: ""
  foreground: "#ffffff"
  background: "#000000"
  template: "{{ if gt .Weight 0.0 }} {{ round .Weight 2 }}kg {{ end }}"
  options:
    access_token: "11111111111111111"
    refresh_token: "1111111111111111"
    http_timeout: 1500
```

## Options

- `access_token`
  - Type: [`template`][templates]
  - Description: token from Withings login, see login link in section above.
- `refresh_token`
  - Type: [`template`][templates]
  - Description: token from Withings login, see login link in section above.
- `expires_in`
  - Type: `int`
  - Default: `0`
  - Description: the default timeout of the token from the Withings login
- `http_timeout`
  - Type: `int`
  - Default: `20`
  - Description: how long do you want to wait before you want to see your prompt more than your Withings data?

## Template

### Default Template

```template
{{ if gt .Weight 0.0 }} {{ round .Weight 2 }}kg {{ end }}
```

### Properties

- `.Weight`
  - Type: `float`
  - Description: your last measured weight
- `.SleepHours`
  - Type: `string`
  - Description: your last measured sleep SleepHours
- `.Steps`
  - Type: `int`
  - Description: your last measured steps

Now, go out and be active!

[templates]: ../../configuration/templates.md
[withings]: https://www.withings.com/
