# System Info

## Segment Type

`sysinfo`

## What

Display SysInfo.

## Sample Configuration

```yaml
prompt:
  - segments: ["sysinfo"]

sysinfo:
  type: "sysinfo"
  powerline_symbol: ""
  foreground: "#ffffff"
  background: "#8f43f3"
  template: "  {{ round .PhysicalPercentUsed .Precision }}% "
  options:
    precision: 2
  style: "powerline"
```

## Options

- `Precision`
  - Type: `int`
  - Default: `2`
  - Description: The precision used for any float values

## Template

### Default Template

```template
 {{ round .PhysicalPercentUsed .Precision }}
```

### Properties

- `.PhysicalTotalMemory`
  - Type: `int`
  - Description: is the total of used physical memory
- `.PhysicalAvailableMemory`
  - Type: `int`
  - Description: is the total available physical memory (i.e. the amount immediately available to processes)
- `.PhysicalFreeMemory`
  - Type: `int`
  - Description: is the total of free physical memory (i.e. considers memory used by the system for any reason [e.g.
    caching] as occupied)
- `.PhysicalPercentUsed`
  - Type: `float64`
  - Description: is the percentage of physical memory in usage
- `.SwapTotalMemory`
  - Type: `int`
  - Description: is the total of used swap memory
- `.SwapFreeMemory`
  - Type: `int`
  - Description: is the total of free swap memory
- `.SwapPercentUsed`
  - Type: `float64`
  - Description: is the percentage of swap memory in usage
- `.Load1`
  - Type: `float64`
  - Description: is the current load1 (can be empty on windows)
- `.Load5`
  - Type: `float64`
  - Description: is the current load5 (can be empty on windows)
- `.Load15`
  - Type: `float64`
  - Description: is the current load15 (can be empty on windows)
- `.Disks`
  - Type: `[]struct`
  - Description: an array of [IOCountersStat][ioinfo] object, you can use any property it has e.g. `.Disks.disk0.IoTime`

[ioinfo]: https://github.com/shirou/gopsutil/blob/e0ec1b9cda4470db704a862282a396986d7e930c/disk/disk.go#L32
