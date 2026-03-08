---
title: OS
description: Display OS specific info - defaults to Icon.
---

## Segment Type

`os`

## What

Display OS specific info - defaults to Icon.

## Sample Configuration

```yaml
prompt:
  - segments: ["os"]

os:
  type: "os"
  style: "plain"
  foreground: "#26C6DA"
  background: "#546E7A"
  template: " {{ if .WSL }}WSL at {{ end }}{{.Icon}}"
  options:
    macos: "mac"
```

## Options

- `macos`
  - Type: `string`
  - Default: `\uF179`
  - Description: the string to use for macOS
- `linux`
  - Type: `string`
  - Default: `\uF17C`
  - Description: the icon to use for Linux
- `windows`
  - Type: `string`
  - Default: `\uE62A`
  - Description: the icon to use for Windows
- `display_distro_name`
  - Type: `boolean`
  - Default: `false`
  - Description: display the distro name instead of icon for Linux or WSL
- `alma`
  - Type: `string`
  - Default: `\uF31D`
  - Description: the icon to use for AlmaLinux OS
- `almalinux`
  - Type: `string`
  - Default: `\uF31D`
  - Description: the icon to use for AlmaLinux OS
- `almalinux9`
  - Type: `string`
  - Default: `\uF31D`
  - Description: the icon to use for AlmaLinux OS 9
- `alpine`
  - Type: `string`
  - Default: `\uF300`
  - Description: the icon to use for Alpine Linux
- `android`
  - Type: `string`
  - Default: `\ue70e`
  - Description: the icon to use for Android
- `aosc`
  - Type: `string`
  - Default: `\uF301`
  - Description: the icon to use for AOSC OS
- `arch`
  - Type: `string`
  - Default: `\uF303`
  - Description: the icon to use for Arch Linux
- `centos`
  - Type: `string`
  - Default: `\uF304`
  - Description: the icon to use for CentOS
- `coreos`
  - Type: `string`
  - Default: `\uF305`
  - Description: the icon to use for CoreOS Container Linux
- `debian`
  - Type: `string`
  - Default: `\uF306`
  - Description: the icon to use for Debian
- `deepin`
  - Type: `string`
  - Default: `\uF321`
  - Description: the icon to use for deepin
- `devuan`
  - Type: `string`
  - Default: `\uF307`
  - Description: the icon to use for Devuan GNU+Linux
- `elementary`
  - Type: `string`
  - Default: `\uF309`
  - Description: the icon to use for elementary OS
- `endeavouros`
  - Type: `string`
  - Default: `\uF322`
  - Description: the icon to use for EndeavourOS
- `fedora`
  - Type: `string`
  - Default: `\uF30a`
  - Description: the icon to use for Fedora
- `freebsd`
  - Type: `string`
  - Default: `\udb82\udce0`
  - Description: the icon to use for FreeBSD
- `gentoo`
  - Type: `string`
  - Default: `\uF30d`
  - Description: the icon to use for Gentoo Linux
- `kali`
  - Type: `string`
  - Default: `\uf327`
  - Description: the icon to use for Kali Linux
- `mageia`
  - Type: `string`
  - Default: `\uF310`
  - Description: the icon to use for Mageia
- `manjaro`
  - Type: `string`
  - Default: `\uF312`
  - Description: the icon to use for Manjaro Linux
- `mint`
  - Type: `string`
  - Default: `\udb82\udced`
  - Description: the icon to use for Linux Mint
- `neon`
  - Type: `string`
  - Default: `\uf331`
  - Description: the icon to use for KDE neon
- `nixos`
  - Type: `string`
  - Default: `\uF313`
  - Description: the icon to use for NixOS
- `opensuse`
  - Type: `string`
  - Default: `\uF314`
  - Description: the icon to use for openSUSE
- `opensuse-tumbleweed`
  - Type: `string`
  - Default: `\uF314`
  - Description: the icon to use for openSUSE Tumbleweed
- `raspbian`
  - Type: `string`
  - Default: `\uF315`
  - Description: the icon to use for Raspberry Pi OS (Raspbian)
- `redhat`
  - Type: `string`
  - Default: `\uF316`
  - Description: the icon to use for Red Hat Enterprise Linux (RHEL)
- `rocky`
  - Type: `string`
  - Default: `\uF32B`
  - Description: the icon to use for Rocky Linux
- `sabayon`
  - Type: `string`
  - Default: `\uF317`
  - Description: the icon to use for Sabayon
- `slackware`
  - Type: `string`
  - Default: `\uF319`
  - Description: the icon to use for Slackware Linux
- `ubuntu`
  - Type: `string`
  - Default: `\uF31b`
  - Description: the icon to use for Ubuntu
- `void`
  - Type: `string`
  - Default: `\uf32e`
  - Description: the icon to use for Void Linux

## Template

### Default Template

```template
 {{ if .WSL }}WSL at {{ end }}{{.Icon}}
```

### Properties

- `.Icon`
  - Type: `string`
  - Description: the OS icon
