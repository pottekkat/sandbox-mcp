# Sandbox MCP

<img align="right" src="logo.png" alt="Sandbox MCP Logo" width="200" height="200">

Sandbox MCP is a Model Context Protocol (MCP) server that lets LLMs (MCP hosts/clients) run code and configuration in secure, isolated Docker containers.

While LLMs are really good at generating code, most can't run the code they generate. This could result in you running untested code directly on your machine, which could have unintended consequences.

Sandbox MCP gives the LLMs an easy-to-use execution environment that anyone can create and configure through a simple, AI-native MCP server that runs locally.

_Inspired by [Codapi](https://codapi.org). Some sandboxes are the same as [Codapi sandboxes](https://github.com/nalgeon/sandboxes)._

## Installation

### Download Binary

You can [download](https://github.com/pottekkat/sandbox-mcp/releases) and use the appropriate binary for your operating system and processor archetecture from the "Releases" page.

### Install via Go

Prerequisites:

- Go 1.24 or higher

```bash
go install github.com/pottekkat/sandbox-mcp/cmd/sandbox-mcp@latest
```

Get the path to the `sandbox-mcp` binary:

```bash
which sandbox-mcp
```

### Build from Source

See [Development](#development) section below.

## Usage

### Initilization

Before you use `sandbox-mcp` with LLMs, you need to initialize its configuration:

```bash
# Create the configuration directory and
# pull the default sandboxes from GitHub
sandbox-mcp --pull

# Build the Docker images for the sandboxes
sandbox-mcp --build
```

> [!NOTE]
> Make sure you have Docker installed and running.

### With MCP Hosts/Clients

Add this to your `claude_desktop_config.json` for Claude Desktop or `mcp.json` for Cursor:

```json
{
    "mcpServers": {
        "sandbox-mcp": {
            "command": "path/to/sandbox-mcp",
            "args": [
                "--stdio"
            ]
        }
    }
}
```

> [!NOTE]
> Make sure to replace `path/to/sandbox-mcp` with the actual path to the `sandbox-mcp` binary.

## Available Sandboxes

### shell

Run shell commands in a Linux environment with strict security and network constraints.

### python

Run Python code with a set of pre-installed libraries.

> [!IMPORTANT]
> ### Your Own Sandbox
> 
> You can create your own sandboxes by creating a new directory in the `sandboxes` directory with your sandbox name and adding a `Dockerfile` and `config.json` to it. See [/sandboxes/](/sandboxes/) for examples.

### network-tools

Use various network tools in an isolated Linux sandbox. The container has network access.

See [jonlabelle/docker-network-tools](https://github.com/jonlabelle/docker-network-tools) for a list of available tools.

### go

Run simple Go code in an isolated sandbox.

### javascript

Run JavaScript code using Node.js.

## Development

Fork and clone the repository:

```bash
git clone https://github.com/username/sandbox-mcp.git
```

Change into the directory:

```bash
cd sandbox-mcp
```

Install dependencies:

```bash
make deps
```

Build the project:

```bash
make build
```

Update your MCP servers configuration to point to the local build:

```json
{
    "mcpServers": {
        "sandbox-mcp": {
            "command": "/path/to/sandbox-mcp/dist/sandbox-mcp",
            "args": [
                "--stdio"
            ]
        }
    }
}
```

## License

[MIT License](LICENSE)
