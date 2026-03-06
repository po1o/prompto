---
title: MCP Validator Function
description: 'Azure Function implementing the Model Context Protocol server for validating prompto configurations'
---

## Overview

This directory contains the Azure Function that implements the Model Context Protocol (MCP) server for
validating prompto configurations.

## Endpoints

- `POST /api/mcp` - MCP server endpoint that handles validation requests
- `GET /api/mcp` - Returns server information and available tools

## Supported Tools

### validate_config

Validate an prompto configuration.

- Supports JSON, YAML, and TOML formats
- Returns detailed validation errors with JSON paths

### validate_segment

Validate a segment snippet (individual prompt segment).

- Validates against the segment schema definition
- Useful for testing individual segments before adding them to a full configuration
- Supports JSON, YAML, and TOML formats

## Usage

### As an MCP Server

Configure your MCP client to connect to this server:

```json
{
  "mcpServers": {
    "prompto-validator": {
      "url": "https://prompto.dev/api/mcp",
      "transport": "http"
    }
  }
}
```

### Direct HTTP Requests

#### Get Server Info

```bash
curl https://prompto.dev/api/mcp
```

#### List Available Tools

```bash
curl -X POST https://prompto.dev/api/mcp \
  -H "Content-Type: application/json" \
  -d '{
    "jsonrpc": "2.0",
    "method": "tools/list",
    "id": 1
  }'
```

#### Validate a Configuration

```bash
curl -X POST https://prompto.dev/api/mcp \
  -H "Content-Type: application/json" \
  -d '{
    "jsonrpc": "2.0",
    "method": "tools/call",
    "params": {
      "name": "validate_config",
      "arguments": {
        "content": "{\"$schema\":\"https://raw.githubusercontent.com/JanDeDobbeleer/prompto/main/
themes/schema.json\",\"blocks\":[]}",
        "format": "json"
      }
    },
    "id": 1
  }'
```

#### Validate a Segment Snippet

```bash
curl -X POST https://prompto.dev/api/mcp \
  -H "Content-Type: application/json" \
  -d '{
    "jsonrpc": "2.0",
    "method": "tools/call",
    "params": {
      "name": "validate_segment",
      "arguments": {
        "content": "{\"type\":\"path\",\"style\":\"powerline\",\"foreground\":\"#ffffff\",\"background\":\"#61AFEF\",\"template\":\" {{ .Path }} \"}",
        "format": "json"
      }
    },
    "id": 2
  }'
```

## Response Format

The validation result includes:

- `valid`: Boolean indicating if the configuration is valid
- `errors`: Array of validation errors (if any)
- `warnings`: Array of warnings (best practices, deprecations)
- `detectedFormat`: The detected or specified format
- `parsedConfig`: The parsed configuration object (for debugging)

Example response:

```json
{
  "jsonrpc": "2.0",
  "result": {
    "content": [
      {
        "type": "text",
        "text": "{
          \"valid\": true,
          \"errors\": [],
          \"warnings\": [
            {
              \"path\": \"$schema\",
              \"message\": \"Consider adding \\\"$schema\\\" property for better editor support.\",
              \"type\": \"recommendation\"
            }
          ],
          \"detectedFormat\": \"json\",
          \"parsedConfig\": {...}
        }"
      }
    ]
  },
  "id": 1
}
```

## Development

To test locally:

```bash
cd website/api
npm install
npm start
```

Then send requests to `http://localhost:7071/api/mcp`

## Publishing to MCP Registry

This server is published to the [MCP Registry](https://github.com/modelcontextprotocol/registry) using GitHub Actions.

### Publishing

Publishing is triggered automatically when you push a version tag (same as prompto releases):

```bash
git tag v9.0.0
git push origin v9.0.0
```

The workflow will:

1. Extract version from the tag (e.g., `v9.0.0` → `9.0.0`)
2. Update `server.json` version to match
3. Validate the `server.json` file
4. Authenticate with the MCP Registry using GitHub OIDC
5. Publish the server to the registry

**Note**: The MCP server version will stay in sync with prompto versions automatically.

### Files

- `server.json` - MCP Registry server configuration
- `server.schema.json` - JSON schema for validation
- `validate-server.js` - Validation script
- `.github/workflows/publish-mcp.yml` - GitHub Actions workflow

### Validating server.json Locally

```bash
cd website/api
npm install
cd mcp
node validate-server.js
```
