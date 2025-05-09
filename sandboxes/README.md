# Creating Your Own Sandbox

_Creating sandboxes is easy_. Let's walk through creating a sandbox named `my-sandbox`, a simple Linux environment with a few pre-installed tools.

First, create a new directory `my-sandbox` inside `$XDG_CONFIG_HOME/sandbox-mcp/sandboxes` to store the configuration:

```bash
mkdir $XDG_CONFIG_HOME/sandbox-mcp/sandboxes/my-sandbox
cd $XDG_CONFIG_HOME/sandbox-mcp/sandboxes/my-sandbox
```

Inside the directory, add a `Dockerfile` that sets up the sandbox:

```dockerfile
# Use a lightweight Debian image
FROM debian:12-slim

# Install some basic command line tools
# and remove the apt cache to save space
RUN apt-get update && \
    apt-get install -y --no-install-recommends \
        curl \
        git \
        vim \
        less \
        procps \
        iputils-ping \
        net-tools \
        iproute2 \
        dnsutils \
        openssl \
        ca-certificates && \
    rm -rf /var/lib/apt/lists/*

# Create a new non-root user named sandbox
RUN adduser --home /sandbox --disabled-password sandbox

# Switch to the sandbox user
# and set the working directory to /sandbox
USER sandbox
WORKDIR /sandbox
```

Then, build and tag the Docker image:

```bash
docker build --tag sandbox-mcp/my-sandbox:latest .
```

In addition to the Dockerfile, a sandbox must have a JSON configuration file `config.json`. This file stores the configuration of the sandbox, as shown below for `my-sandbox`:

```json
{
	"id": "my-sandbox",
	"name": "My Sandbox",
	"description": "A simple Linux sandbox.",
	"hints": {
		"isDestructive": false,
		"isIdempotent": true
	},
	"version": "0.1.0",
	"image": "sandbox-mcp/my-sandbox:latest",
	"user": "sandbox",
	"entrypoint": "main.sh",
	"timeout": 60,
	"command": [
		"sh",
		"main.sh"
	],
	"parameters": {
		"additionalFiles": true
	},
	"security": {
		"readOnly": true,
		"capDrop": [
			"all"
		],
		"securityOpt": [
			"no-new-privileges:true"
		],
		"network": "bridge"
	},
	"resources": {
		"cpu": 1,
		"memory": 64,
		"processes": 64,
		"files": 96
	},
	"mount": {
		"workdir": "/sandbox",
		"tmpdirPrefix": "sandbox-mcp-",
		"scriptPerms": "0755",
		"readOnly": true
	}
}
```

Each of these properties is explained below:

- `id`: Unique identifier for the sandbox.
- `name`: Human-readable name for the sandbox.
- `description`: Human-readable description for the sandbox.
- `hints`: Additional metadata about the sandbox behavior to help clients understand how to use it.
	- `isReadOnly`: If `true`, indicates that the sandbox is read-only. Defaults to the value of `readOnly` in the `security` and `mount` configuration.
	- `isDestructive`: If `true`, indicates that the sandbox performs destructive updates. Defaults to `false`.
	- `isIdempotent`: If `true`, indicates that calling the sandbox multiple times with the same parameters will have the same effect. Defaults to `true`.
	- `isExternalInteraction`: If `true`, indicates that the sandbox could interact with external entities. Defaults to the `network` property in the `security` configuration (`false` if `network` is set to `none`, `true` otherwise).
- `version`: Semantic version of the sandbox. It does not do much right now.
- `image`: Docker image and tag to use for the sandbox.
- `user`: User to run the sandbox as.
- `entrypoint`: File where the input from the client is stored to be executed as described by `command`. For example, the [`shell` sandbox](./shell/config.json) has an `entrypoint` of `main.sh`, and the [`go` sandbox](./go/config.json) has an `entrypoint` of `main.go`.
- `timeout`: Maximum execution time of the sandbox in seconds to prevent running indefinitely.
- `before`: Command to run before the client input is executed. See the [`apisix` sandbox](./apisix/config.json) for an example.
- `command`: Command to execute in the sandbox. It typically contains the `entrypoint` file and additional arguments. For example, the [`shell` sandbox](./shell/config.json) has a `command` of `["sh", "main.sh"]`, and the [`go` sandbox](./go/config.json) has a `command` of `["go", "run", "main.go"]`.
- `parameters`: Additional parameters to accept from the client.
	- `additionalFiles`: If `true`, allows the client to pass additional files to the sandbox.
	- `files`: Additional required files to be passed along with the `entrypoint`. For example, the [`go` sandbox](./go/config.json) has a `files` property of `go.mod` to include the `go.mod` file in the sandbox.
- `security`: Security configuration for the sandbox. Directly translates to Docker container configurations.
	- `readOnly`: If `true`, the sandbox is read-only.
	- `capDrop`: Capabilities to drop from the sandbox.
	- `securityOpt`: Security options to pass to the sandbox.
	- `network`: Network mode to use for the sandbox.
- `resources`: Resource configuration for the sandbox.
	- `cpu`: CPU limit for the sandbox.
	- `memory`: Memory limit for the sandbox.
	- `processes`: Process limit for the sandbox.
	- `files`: File descriptor limit for the sandbox.
- `mount`: Mount configuration for the sandbox.
	- `workdir`: Working directory for the sandbox.
	- `tmpdirPrefix`: Prefix for the temporary directory created for the sandbox.
	- `scriptPerms`: Permissions for the `entrypoint` file.
	- `readOnly`: If `true`, the sandbox (volume mount) is read-only.

After configuring the sandbox, you can reload the MCP host/client application (e.g., Cursor IDE or Claude Desktop) to apply the changes. You will see `my-sandbox` in the list of available tools.

Feel free to share the sandboxes you create with the community!
