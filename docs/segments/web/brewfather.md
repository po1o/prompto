---
title: Brewfather
description: Calling all brewers! Keep up-to-date with the status of your Brewfather batch directly in your commandline prompt using the brewfather segment!
---

## Segment Type

`brewfather`

## What

Calling all brewers! Keep up-to-date with the status of your [Brewfather][brewfather] batch directly in your commandline
prompt using the brewfather segment!

You will need your User ID and API Key as generated in Brewfather's Settings screen, enabled with **batches.read** and
**recipes.read** scopes.

## Sample Configuration

This example uses the default segment template to show a rendition of detail appropriate to the status of the batch

Additionally, the background of the segment will turn red if the latest reading is over 4 hours old - possibly helping
indicate an issue if, for example there is a Tilt or similar device that is supposed to be logging to Brewfather every
15 minutes.

### Info

Temperature units are in degrees C and specific gravity is expressed as `X.XXX` values.

```yaml
prompt:
  - segments: ["brewfather"]

brewfather:
  type: "brewfather"
  style: "powerline"
  powerline_symbol: ""
  foreground: "#ffffff"
  background: "#33158A"
  background_templates: ["{{ if and (.Reading) (eq .Status \"Fermenting\") (gt .ReadingAge 4) }}#cc1515{{end}}"]
  options:
    user_id: "abcdefg123456"
    api_key: "qrstuvw78910"
    batch_id: "hijklmno098765"
```

## Options

- `user_id`
  - Type: `string`
  - Description: as provided by Brewfather's Generate API Key screen
- `api_key`
  - Type: [`template`][templates]
  - Description: as provided by Brewfather's Generate API Key screen
- `batch_id`
  - Type: `string`
  - Description: Get this by navigating to the desired batch on the brewfather website, the batch id is at the end of
    the URL in the address bar
- `day_icon`
  - Type: `string`
  - Default: `d`
  - Description: icon or letter to use to indicate days
- `http_timeout`
  - Type: `int`
  - Default: `20`
  - Description: in milliseconds - How long to wait for the Brewfather service to answer the request

## Icons

You can override the icons for temperature trend as used by template property `.TemperatureTrendIcon` with:

- `doubleup_icon`
  - Default: `↑↑`
  - Description: increases of more than 4°C
- `singleup_icon`
  - Default: `↑`
  - Description: increase 2-4°C
- `fortyfiveup_icon`
  - Default: `↗`
  - Description: increase 0.5-2°C
- `flat_icon`
  - Default: `→`
  - Description: change less than 0.5°C
- `fortyfivedown_icon`
  - Default: `↘`
  - Description: decrease 0.5-2°C
- `singledown_icon`
  - Default: `↓`
  - Description: decrease 2-4°C
- `doubledown_icon`
  - Default: `↓↓`
  - Description: decrease more than 4°C

You can override the default icons for batch status as used by template property `.StatusIcon` with:

- `planning_status_icon`
  - Default: `\uF8EA`
- `brewing_status_icon`
  - Default: `\uF7DE`
- `fermenting_status_icon`
  - Default: `\uF499`
- `conditioning_status_icon`
  - Default: `\uE372`
- `completed_status_icon`
  - Default: `\uF7A5`
- `archived_status_icon`
  - Default: `\uF187`

## Template

### Default Template

```template
{{ .StatusIcon }} {{ if .DaysBottledOrFermented }}{{ .DaysBottledOrFermented }}{{ .DayIcon }} {{ end }}{{ url .Recipe.Name .URL }} {{ printf \"%.1f\" .MeasuredAbv }}%{{ if and (.Reading) (eq .Status \"Fermenting\") }} {{ printf \"%.3f\" .Reading.Gravity }} {{ .Reading.Temperature }}\u00b0 {{ .TemperatureTrendIcon }}{{ end }}
```

### Properties

- `.Status`
  - Type: `string`
  - Description: One of "Planning", "Brewing", "Fermenting", "Conditioning", "Completed" or "Archived"
- `.StatusIcon`
  - Type: `string`
  - Description: Icon representing above stats. Can be overridden with properties shown above
- `.TemperatureTrendIcon`
  - Type: `string`
  - Description: Icon showing temperature trend based on latest and previous reading
- `.DaysFermenting`
  - Type: `int`
  - Description: days since start of fermentation
- `.DaysBottled`
  - Type: `int`
  - Description: days since bottled/kegged
- `.DaysBottledOrFermented`
  - Type: `int`
  - Description: one of the above, chosen automatically based on batch status
- `.Recipe.Name`
  - Type: `string`
  - Description: The recipe being brewed in this batch
- `.BatchName`
  - Type: `string`
  - Description: The name of this batch
- `.BatchNumber`
  - Type: `int`
  - Description: The number of this batch
- `.MeasuredAbv`
  - Type: `float`
  - Description: The ABV for the batch - either estimated from recipe or calculated from entered OG and FG values
- `.ReadingAge`
  - Type: `int`
  - Description: age in hours of most recent reading or -1 if there are no readings available

#### Reading

`.Reading` contains the most recent data from devices or manual entry as visible on the Brewfather's batch Readings
graph. If there are no readings available, `.Reading` will be null.

- `.Reading.Gravity`
  - Type: `float`
  - Description: specific gravity (in decimal point format)
- `.Reading.Temperature`
  - Type: `float`
  - Description: temperature in °C
- `.Reading.Time`
  - Type: `int`
  - Description: unix timestamp of reading
- `.Reading.Comment`
  - Type: `string`
  - Description: comment attached to this reading
- `.Reading.DeviceType`
  - Type: `string`
  - Description: source of the reading, e.g. "Tilt"
- `.Reading.DeviceID`
  - Type: `string`
  - Description: id of the device, e.g. "PINK"

#### Additional properties

- `.MeasuredOg`
  - Type: `float`
  - Description: The OG for the batch as manually entered into Brewfather
- `.MeasuredFg`
  - Type: `float`
  - Description: The FG for the batch as manually entered into Brewfather
- `.BrewDate`
  - Type: `int`
  - Description: The unix timestamp of the brew day
- `.FermentStartDate`
  - Type: `int`
  - Description: The unix timestamp when fermentation was started
- `.BottlingDate`
  - Type: `time`
  - Description: The unix timestamp when bottled/kegged
- `.TemperatureTrend`
  - Type: `float`
  - Description: The difference between the most recent and previous temperature in °C
- `.DayIcon`
  - Type: `string`
  - Description: given by "day_icon", or "d" by default

#### Hyperlink support

- `.URL`
  - Type: `string`
  - Description: the URL for the batch in the Brewfather app. You can use this to add a hyperlink to the segment if you
    are using a terminal that supports it. The default template implements this

### Advanced Templating

The built in template will provides key useful information. However, you can use the properties about the batch to build
your own. For reference, the built-in template looks like this:

```yaml
type: "brewfather"
template: "{{.StatusIcon}} {{if .DaysBottledOrFermented}}{{.DaysBottledOrFermented}}{{.DayIcon}} {{end}}[{{.Recipe.Name}}]({{.URL}}) {{printf \"%.1f\" .MeasuredAbv}}%{{ if and (.Reading) (eq .Status \"Fermenting\")}}: {{printf \"%.3f\" .Reading.Gravity}} {{.Reading.Temperature}}° {{.TemperatureTrendIcon}}{{end}}"
```

### Unit conversion

By default temperature readings are provided in degrees C, gravity readings in decimal Specific Gravity unts (X.XXX).

The following conversion functions are available to the template to convert to other units:

#### Temperature

- `.DegCToF`
  - Description: input: `float` degrees in C; output `float` degrees in F (1 decimal place)
- `.DegCToKelvin`
  - Description: input: `float` degrees in C; output `float` Kelvin (1 decimal place)

#### Gravity

- `.SGToBrix`
  - Description: input `float` SG in x.xxx decimal; output `float` Brix (2 decimal places)
- `.SGToPlato`
  - Description: input `float` SG in x.xxx decimal; output `float` Plato (2 decimal places)

_(These use the polynomial conversions from [Wikipedia][wikipedia_gravity_page])_

#### Example

```yaml
type: "brewfather"
template: "{{if .Reading}}{{.SGToBrix .Reading.Gravity}}°Bx, {{.DegCToF .Reading.Temperature}}°F{{end}}"
```

To display gravity as SG in XXXX format (e.g. "1020" instead of "1.020"), use the `mulf` template function

```yaml
type: "brewfather"
template: "{{if .Reading}}{{.mulf 1000 .Reading.Gravity}}, {{.DegCToF .Reading.Temperature}}°F{{end}}"
```

[templates]: ../../configuration/templates.md
[brewfather]: http://brewfather.app
[wikipedia_gravity_page]: https://en.wikipedia.org/wiki/Brix#Specific_gravity_2
