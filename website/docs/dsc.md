---
id: dsc
title: Desired State Configuration
sidebar_label: 🖥️ Desired State Configuration
---

Prompto supports Desired State Configuration (DSC) for declarative configuration management, enabling automated
deployment and consistent configuration across multiple systems.

## Concept

Prompto DSC builds on the traditional Prompto configuration approach by adding automation and orchestration
capabilities. Instead of manually configuring your prompt, you can define the desired state declaratively and let DSC
ensure your system matches that state.

DSC works with **resources** that represent different aspects of your Prompto setup:

- **Configuration Resource**: Manages your Prompto configuration files
- **Shell Resource**: Handles shell initialization and integration
- **Font Resource**: Tracks installed Nerd Fonts

These resources can be used standalone through the CLI or integrated with orchestration tools like WinGet and
Microsoft DSC for automated deployments.

## Overview

DSC support in Prompto provides:

- **Declarative configuration**: Define the desired state of your Prompto setup
- **Automated deployment**: Configure Prompto as part of system provisioning workflows
- **Shell integration**: Automatically configure shell initialization for bash, zsh, fish, PowerShell, and more
- **Font management**: Track installed Nerd Fonts through DSC state
- **Orchestration support**: Integration with WinGet and Microsoft DSC tools

DSC functionality is available through the `prompto` CLI and can be used standalone or with orchestration tools.

## DSC Resources

Prompto provides the following DSC resources:

### Configuration Resource

Manages Prompto configuration files.

**Operations**: `get`, `set`, `export`, `schema`

```bash
# Get current configuration state
prompto config dsc get

# Apply a configuration
prompto config dsc set --state '{"states":[{"source":"~/mytheme.omp.json","format":"json"}]}'

# Get configuration schema
prompto config dsc schema
```

### Shell Resource

Manages shell initialization and integration.

**Operations**: `get`, `set`, `export`, `schema`

```bash
# Get current shell configurations
prompto init bash dsc get

# Configure shell initialization
prompto init bash dsc set --state '{"states":[{"name":"bash","command":"prompto init bash --config ~/mytheme.omp.json"}]}'
```

### Font Resource

Tracks Nerd Fonts installed through Prompto.

**Operations**: `get`, `export`, `schema`

```bash
# Get installed fonts
prompto font dsc get

# Get font schema
prompto font dsc schema
```

## Direct CLI Usage

You can use the DSC commands directly from the command line for configuration management.

### Managing Configurations

#### Get State

Retrieve the current configuration state:

```bash
prompto config dsc get
```

Example output:

```json
{
  "states": [
    {
      "format": "json",
      "source": "~/mytheme.omp.json"
    }
  ]
}
```

#### Set State

Apply a new configuration state:

```bash
prompto config dsc set --state '{"states":[{"source":"~/mytheme.omp.json","format":"json"}]}'
```

This creates or updates the configuration file at the specified location with the provided format.

#### Schema

Get the JSON schema for the configuration resource:

```bash
prompto config dsc schema
```

Use this to understand the structure and available options for configuration states.

### Managing shell Integration

#### Bash

Configure Prompto initialization for bash:

```bash
# Get current state
prompto init bash dsc get

# Set initialization
prompto init bash dsc set --state '{"states":[{"name":"bash","command":"prompto init bash --config ~/mytheme.omp.json"}]}'
```

This automatically updates your `.bashrc` or `.bash_profile` with the Prompto initialization command.

#### Zsh

Configure Prompto initialization for zsh:

```bash
# Get current state
prompto init zsh dsc get

# Set initialization
prompto init zsh dsc set --state '{"states":[{"name":"zsh","command":"prompto init zsh --config ~/mytheme.omp.json"}]}'
```

This automatically updates your `.zshrc` with the Prompto initialization command.

#### PowerShell

Configure Prompto initialization for PowerShell:

```powershell
# Get current state
prompto init pwsh dsc get

# Set initialization
prompto init pwsh dsc set --state '{"states":[{"name":"pwsh","command":"prompto init pwsh --config ~/mytheme.omp.json"}]}'
```

This automatically updates your PowerShell profile with the Prompto initialization command.

#### Fish

Configure Prompto initialization for fish:

```bash
# Get current state
prompto init fish dsc get

# Set initialization
prompto init fish dsc set --state '{"states":[{"name":"fish","command":"prompto init fish --config ~/mytheme.omp.json"}]}'
```

This automatically updates your fish `config.fish` with the Prompto initialization command.

## Orchestration with WinGet

WinGet configuration enables you to install Prompto and apply configuration in a single declarative file.

### Basic WinGet configuration

Create a configuration file to install and configure Prompto:

```yaml title="prompto-setup.yaml"
$schema: https://raw.githubusercontent.com/PowerShell/DSC/main/schemas/v3/config/document.json
metadata:
  winget:
    processor: dscv3
resources:
  - name: Install Prompto
    type: Microsoft.WinGet.DSC/WinGetPackage
    properties:
      id: JanDeDobbeleer.OhMyPosh
      source: winget
```

Apply the configuration:

```powershell
winget configure prompto-setup.yaml
```

### Complete setup with configuration and shell

This example installs Prompto, adds your configuration, and initializes PowerShell:

```yaml title="prompto-complete.yaml"
$schema: https://raw.githubusercontent.com/PowerShell/DSC/main/schemas/v3/config/document.json
metadata:
  winget:
    processor: dscv3
resources:
  - name: Install Prompto
    type: Microsoft.WinGet.DSC/WinGetPackage
    properties:
      id: JanDeDobbeleer.OhMyPosh
      source: winget

  - name: Add Prompto configuration
    type: OhMyPosh/Config
    properties:
      states:
        - source: ~/mytheme.omp.json
          format: json

  - name: Initialize PowerShell
    type: OhMyPosh/Shell
    properties:
      states:
        - name: pwsh
          command: prompto init pwsh --config ~/mytheme.omp.json
```

Apply with:

```powershell
winget configure prompto-complete.yaml
```

### Multi-shell configuration

Initialize multiple shells with different configurations:

```yaml title="prompto-multi-shell.yaml"
$schema: https://raw.githubusercontent.com/PowerShell/DSC/main/schemas/v3/config/document.json
metadata:
  winget:
    processor: dscv3
resources:
  - name: Install Prompto
    type: Microsoft.WinGet.DSC/WinGetPackage
    properties:
      id: JanDeDobbeleer.OhMyPosh
      source: winget

  - name: Add work configuration
    type: OhMyPosh/Config
    properties:
      states:
        - source: ~/work-theme.omp.json
          format: json

  - name: Add personal configuration
    type: OhMyPosh/Config
    properties:
      states:
        - source: ~/personal-theme.omp.json
          format: json

  - name: Initialize PowerShell with work configuration
    type: OhMyPosh/Shell
    properties:
      states:
        - name: pwsh
          command: prompto init pwsh --config ~/work-theme.omp.json

  - name: Initialize Bash with personal configuration
    type: OhMyPosh/Shell
    properties:
      states:
        - name: bash
          command: prompto init bash --config ~/personal-theme.omp.json
```

## Orchestration with Microsoft DSC

Microsoft DSC (`dsc`) provides cross-platform configuration management capabilities. Prompto provides native DSC
resources that can be used in DSC configuration documents.

### Example DSC configuration

Create a configuration document for Prompto:

```yaml title="prompto-dsc.yaml"
$schema: https://aka.ms/dsc/schemas/v3/bundled/config/document.json
resources:
  - name: Add Prompto configuration
    type: OhMyPosh/Config
    properties:
      states:
        - source: ~/mytheme.omp.json
          format: json

  - name: Initialize PowerShell
    type: OhMyPosh/Shell
    properties:
      states:
        - name: pwsh
          command: prompto init pwsh --config ~/mytheme.omp.json
```

Apply the configuration using the `dsc` CLI:

```bash
dsc config set --document prompto-dsc.yaml
```

### Complete configuration with multiple shells

```yaml title="prompto-complete-dsc.yaml"
$schema: https://aka.ms/dsc/schemas/v3/bundled/config/document.json
resources:
  - name: Add primary configuration
    type: OhMyPosh/Config
    properties:
      states:
        - source: ~/primary-theme.omp.json
          format: json

  - name: Add secondary configuration
    type: OhMyPosh/Config
    properties:
      states:
        - source: ~/secondary-theme.omp.json
          format: yaml

  - name: Initialize PowerShell
    type: OhMyPosh/Shell
    properties:
      states:
        - name: pwsh
          command: prompto init pwsh --config ~/primary-theme.omp.json

  - name: Initialize Bash
    type: OhMyPosh/Shell
    properties:
      states:
        - name: bash
          command: prompto init bash --config ~/primary-theme.omp.json

  - name: Initialize Zsh
    type: OhMyPosh/Shell
    properties:
      states:
        - name: zsh
          command: prompto init zsh --config ~/secondary-theme.omp.json
```

### Resource Types

Prompto provides the following DSC resource types:

#### OhMyPosh/Config

Manages Prompto configuration files.

**Properties:**

- `states` (array): List of configuration states
  - `source` (string): Path to the configuration file
  - `format` (string): Format of the configuration file (`json`, `yaml`, `toml`)

#### OhMyPosh/Shell

Manages shell initialization.

**Properties:**

- `states` (array): List of shell configurations
  - `name` (string): Shell name (`bash`, `zsh`, `pwsh`, `fish`, etc.)
  - `command` (string): Prompto initialization command

#### OhMyPosh/Font

Tracks installed Nerd Fonts. This resource is read-only and automatically populated when fonts are installed through
Prompto.

## Configuration State Management

DSC state is stored in the Prompto cache and persists across sessions. This enables:

- **State tracking**: Prompto remembers configurations set through DSC
- **Idempotency**: Running the same DSC command multiple times produces the same result
- **State validation**: Query current state before making changes

## Advanced Usage

### Multiple configurations

You can manage multiple configuration files:

```bash
prompto config dsc set --state '{
  "states": [
    {"source":"~/work.omp.json","format":"json"},
    {"source":"~/personal.omp.json","format":"json"}
  ]
}'
```

### Shell-Specific Initialization

Initialize multiple shells with different configuration:

```bash
# Bash with one configuration
prompto init bash dsc set --state '{"states":[{"name":"bash","command":"prompto init bash --config ~/bash-theme.omp.json"}]}'

# PowerShell with another configuration
prompto init pwsh dsc set --state '{"states":[{"name":"pwsh","command":"prompto init pwsh --config ~/pwsh-theme.omp.json"}]}'
```

## Supported shells

DSC shell configuration supports the following shells:

- **bash**: Configures `.bashrc` or `.bash_profile`
- **zsh**: Configures `.zshrc`
- **fish**: Configures `~/.config/fish/config.fish`
- **pwsh**: Configures PowerShell profile (`$PROFILE`)
- **nu**: Configures `~/.config/nushell/config.nu`
- **elvish**: Configures `.elvish/rc.elv`
- **xonsh**: Configures `.xonshrc`

The shell integration automatically:

- Creates configuration files if they don't exist
- Updates existing Prompto initialization commands
- Preserves shell-specific command syntax
- Maintains proper whitespace and formatting

## See Also

- [General configuration](/docs/configuration/general) - Main configuration documentation
- [Installation](/docs/installation/windows) - Installing Prompto
- [Themes](https://github.com/po1o/prompto/tree/main/themes) - Available themes
- [WinGet configuration](https://learn.microsoft.com/windows/package-manager/configuration/) - WinGet DSC documentation
- [Microsoft DSC](https://learn.microsoft.com/powershell/dsc/overview) - Microsoft DSC documentation
