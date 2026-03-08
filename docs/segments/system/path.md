---
title: Path
description: Display the current path.
---

## Segment Type

`path`

## What

Display the current path.

## Sample Configuration

```yaml
prompt:
  - segments: ["path"]

path:
  type: "path"
  style: "powerline"
  powerline_symbol: ""
  foreground: "#ffffff"
  background: "#61AFEF"
  options:
    style: "folder"
    mapped_locations:
      C:\temp: ""
```

## Options

- `folder_separator_icon`
  - Type: `string`
  - Default: `/`
  - Description: the symbol to use as a separator between folders
- `folder_separator_template`
  - Type: `string`
  - Description: the [template][templates] to use as a separator between folders
- `home_icon`
  - Type: `string`
  - Default: `~`
  - Description: the icon to display when at `$HOME`
- `folder_icon`
  - Type: `string`
  - Default: `..`
  - Description: the icon to use as a folder indication
- `windows_registry_icon`
  - Type: `string`
  - Default: `\uF013`
  - Description: the icon to display when in the Windows registry
- `style`
  - Type: `enum`
  - Default: `agnoster`
  - Description: how to display the current path
- `mixed_threshold`
  - Type: `number`
  - Default: `4`
  - Description: the maximum length of a path segment that will be displayed when using `Mixed`
- `max_depth`
  - Type: `number`
  - Default: `1`
  - Description: maximum path depth to display before shortening when using `agnoster_short`
- `max_width`
  - Type: `any`
  - Default: `0`
  - Description: maximum path length to display when using `powerlevel` or `agnoster`, can leverage [templates]
- `hide_root_location`
  - Type: `boolean`
  - Default: `false`
  - Description: hides the root location if it doesn't fit in the last `max_depth` folders when using `agnoster_short`
- `cycle`
  - Type: `[]string`
  - Description: a list of color overrides to cycle through when coloring individual path folders, for example
    `["#ffffff,#111111"]`
- `cycle_folder_separator`
  - Type: `boolean`
  - Default: `false`
  - Description: colorize the `folder_separator_icon` as well when using a cycle
- `folder_format`
  - Type: `string`
  - Default: `%s`
  - Description: format to use on individual path folders
- `edge_format`
  - Type: `string`
  - Default: `%s`
  - Description: format to use on the first and last folder of the path
- `left_format`
  - Type: `string`
  - Default: `%s`
  - Description: format to use on the first folder of the path - defaults to `edge_format`
- `right_format`
  - Type: `string`
  - Default: `%s`
  - Description: format to use on the last folder of the path - defaults to `edge_format`
- `gitdir_format`
  - Type: `string`
  - Description: format to use for a git root directory
- `display_cygpath`
  - Type: `boolean`
  - Default: `false`
  - Description: display the Cygwin style path using `cygpath -u $PWD`
- `display_root`
  - Type: `boolean`
  - Default: `false`
  - Description: display the root `/` on Unix systems
- `dir_length`
  - Type: `number`
  - Default: `1`
  - Description: the length of the directory name to display when using `fish`
- `full_length_dirs`
  - Type: `number`
  - Default: `1`
  - Description: indicates how many full length directory names should be displayed when using `fish`

## Mapped Locations

Allows you to override a location with custom text or an icon. `prompto` checks whether the current path starts with the
configured value and replaces that prefix when it matches. To avoid issues with nested overrides, `prompto` sorts mapped
locations before applying replacements.

- `mapped_locations_enabled`
  - Type: `boolean`
  - Default: `true`
  - Description: replace known locations in the path with the replacements before applying the style
- `mapped_locations`
  - Type: `object`
  - Description: custom glyph or text for specific paths. These mappings still apply when
    `mapped_locations_enabled` is `false`

For example, to swap out `C:\Users\Leet\GitHub` with a GitHub icon, you can do the following:

```yaml
type: "path"
mapped_locations:
  "C:\\Users\\Leet\\GitHub": ""
```

### How it works

- To make mapped locations work cross-platform, use `/` as the path separator. `prompto` will
  automatically match effective separators based on the running operating system.
- If you want to match all child directories, you can use `*` as a wildcard, for example:
  `"C:/Users/Bill/*": "$"` will turn `C:/Users/Bill/Downloads` into `$/Downloads` but leave `C:/Users/Bill` unchanged.
- The character `~` at the start of a mapped location will match the user's home directory.
- The match is _case-insensitive on Windows and macOS_, but case-sensitive on other operating systems. This means that
  for user Bill, who has a user account `Bill` on Windows and `bill` on Linux, `~/Foo` might match
  `C:\Users\Bill\Foo` or `C:\Users\Bill\foo` on Windows but only `/home/bill/Foo` on Linux.

### Warning

To prevent mangling path elements, if you use any text style tags (e.g., `<lightGreen>...</>`) in replacement values,
you should avoid using a chevron character (`<`/`>`) in the `folder_separator_icon` property, and vice versa.

### Using regular expressions

For more complicated cases, you can use the `re:` prefix to use a regular expression with a capture group for matching.
This uses Golang's [regexp] package, so you can use any of the [supported syntax][regexp]. The replacement value will be
the first capture group, subsequent groups will be ignored.

For example, `"re:(C:/[0-9]+/Foo)": "#"` will match `C:\123\Foo\Bar` and replace it with `#\Bar`. The path used for
matching will always use `/`, regardless of the operating system, allowing cross platform matching.

Same as for standard replacements, the match is case insensitive on Windows and WSL mounted drives, but case-sensitive
on other operating systems.

## Style

Style sets the way the path is displayed. Based on previous experience and popular themes, there are 10 flavors.

- `agnoster`
- `agnoster_full`
- `agnoster_short`
- `agnoster_left`
- `full`
- `folder`
- `mixed`
- `letter`
- `unique`
- `powerlevel`
- `fish`

### Agnoster

Renders each intermediate folder as the `folder_icon` separated by the `folder_separator_icon`. The first and the last
(current) folder name are always displayed as-is.

### Agnoster Full

Renders each folder name separated by the `folder_separator_icon`.

### Agnoster Short

When more than `max_depth` levels deep, it renders one `folder_icon` (if `hide_root_location` is `false`, which means
the root folder does not count for depth) followed by the names of the last `max_depth` folders, separated by the
`folder_separator_icon`.

### Agnoster Left

Renders each folder as the `folder_icon` separated by the `folder_separator_icon`. Only the first folder name and its
child are displayed in full.

### Full

Display the current working directory as a full string with each folder separated by the `folder_separator_icon`.

### Folder

Display the name of the current folder.

### Mixed

Works like `agnoster`, but for any intermediate folder name that is short enough, it will be displayed as-is. The
maximum length for the folders to display is governed by the `mixed_threshold` property.

### Letter

Works like `agnoster_full`, but will write every folder name using the first letter only, except when the folder name
starts with a symbol or icon. In particular, the last (current) folder name is always displayed in full.

- `folder` will be shortened to `f`
- `.config` will be shortened to `.c`
- `__pycache__` will be shortened to `__p`
- `➼ folder` will be shortened to `➼ f`

### Unique

Works like `letter`, but will make sure every folder name is the shortest unique value.

The uniqueness refers to the displayed path, so `C:\dev\dev\dev\development` will be displayed as
`C\d\de\dev\development` (instead of `C\d\d\d\development` for `Letter`). Uniqueness does **not** refer to other folders
at the same level, so if `C:\projectA\dev` and `C:\projectB\dev` exist, then both will be displayed as `C\p\dev`.

### Powerlevel

Works like `unique`, but will stop shortening when `max_width` is reached.

### Fish

Works like `letter`, but will display the first `dir_length` characters of each folder name, except for the last number
of folders specified by `full_length_dirs`, which will be displayed in full. Inspired by the Fish shell PWD.

## Template

### Default Template

```template
 {{ .Path }}
```

### Properties

- `.Path`
  - Type: `string`
  - Description: the current directory (based on the `style` property)
- `.Parent`
  - Type: `string`
  - Description: the current directory's parent folder which ends with a path separator (designed for use with style
    `folder`, it is empty if `.Path` contains only one single element)
- `.RootDir`
  - Type: `boolean`
  - Description: true if we're at the root directory (no parent)
- `.Location`
  - Type: `string`
  - Description: the current directory (raw value)
- `.StackCount`
  - Type: `int`
  - Description: the stack count
- `.Writable`
  - Type: `boolean`
  - Description: is the current directory writable by the user or not
- `.Format`
  - Type: `function`
  - Description: format any path based on the segment's settings (e.g. `{{ .Format .Segments.Git.RelativeDir }}`)

[templates]: ../../configuration/templates.md
[regexp]: https://pkg.go.dev/regexp/syntax
