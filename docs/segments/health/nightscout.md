# Nightscout

## Segment Type

`nightscout`

## What

[Nightscout][nightscout] (CGM in the Cloud) is an open source project that exposes CGM data over HTTP. It is commonly
used for secure remote viewing of blood sugar data, including in `prompto` segments on the command line.

## Sample Configuration

This example is using mg/dl by default because the Nightscout API sends the sugar glucose value (.Sgv) in mg/dl format.
Below is also a template for displaying the glucose value in mmol/L. When using different color ranges you should
multiply your high and low range glucose values by 18 and use these values in the templates. You'll also want to think
about your background and foreground colors. Don't use white text on a yellow background, for example.

The `foreground_templates` example below could be set to just a single color, if that color is visible against any of
your backgrounds.

```yaml
prompt:
  - segments: ["nightscout"]

nightscout:
  type: "nightscout"
  style: "diamond"
  foreground: "#ffffff"
  background: "#ff0000"
  background_templates: ["{{ if gt .Sgv 150 }}#FFFF00{{ end }}", "{{ if lt .Sgv 60 }}#FF0000{{ end }}", "#00FF00"]
  foreground_templates: ["{{ if gt .Sgv 150 }}#000000{{ end }}", "{{ if lt .Sgv 60 }}#000000{{ end }}", "#000000"]
  leading_diamond: ""
  trailing_diamond: ""
  template: " {{ .Sgv }}{{ .TrendIcon }}"
  options:
    url: "https://YOURNIGHTSCOUTAPP.herokuapp.com/api/v1/entries.json?count=1&token=APITOKENFROMYOURADMIN"
    http_timeout: 1500
```

Or display in mmol/l (instead of the default mg/dl) with the following template:

```yaml
prompt:
  - segments: ["nightscout"]

nightscout:
  template: " {{ if eq (mod .Sgv 18) 0 }}{{divf .Sgv 18}}.0{{ else }} {{ round (divf .Sgv 18) 1 }}{{ end }}{{ .TrendIcon }}"
  type: "nightscout"
```

## Options

- `url`
  - Type: [`template`][templates]
  - Description: Your Nightscout URL, including the full path to entries.json AND count=1 AND token. Example above.
    You'll know this works if you can curl it yourself and get a single value
- `headers`
  - Type: `map[string]string`
  - Description: A key, value map of Headers to send with the request
- `http_timeout`
  - Type: `int`
  - Default: `20`
  - Description: in milliseconds - how long do you want to wait before you want to see your prompt more than your sugar?
    I figure a half second is a good default

### Info

You can change the icons for trend, put the trend elsewhere, add text, however you like! Make sure your NerdFont has the
glyph you want or [search for one][nf-search].

- `DoubleUpIcon`
  - Description: defaults to `↑↑`
- `SingleUpIcon`
  - Description: defaults to `↑`
- `FortyFiveUpIcon`
  - Description: defaults to `↗`
- `FlatIcon`
  - Description: defaults to `→`
- `FortyFiveDownIcon`
  - Description: defaults to `↘`
- `SingleDownIcon`
  - Description: defaults to `↓`
- `DoubleDownIcon`
  - Description: defaults to `↓↓`

## Template

### Default Template

```template
 {{ .Sgv }}
```

### Properties

- `.ID`
  - Type: `string`
  - Description: The internal ID of the object
- `.Sgv`
  - Type: `int`
  - Description: Your Serum Glucose Value (your sugar)
- `.Date`
  - Type: `int`
  - Description: The unix timestamp of the entry
- `.DateString`
  - Type: `time`
  - Description: The timestamp of the entry
- `.Trend`
  - Type: `int`
  - Description: The trend of the entry
- `.Device`
  - Type: `string`
  - Description: The device linked to the entry
- `.Type`
  - Type: `string`
  - Description: The type of the entry
- `.UtcOffset`
  - Type: `int`
  - Description: The UTC offset
- `.SysTime`
  - Type: `time`
  - Description: The time on the system
- `.Mills`
  - Type: `int`
  - Description: The amount of mills
- `.TrendIcon`
  - Type: `string`
  - Description: By default, this will be something like ↑↑ or ↘ etc but you can override them with any glyph as seen
    above

[templates]: ../../configuration/templates.md
[nightscout]: http://www.nightscout.info/
[nf-search]: https://www.nerdfonts.com/cheat-sheet
