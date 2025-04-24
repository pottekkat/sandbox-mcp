# Creating Your Own Sandbox

Creating a sandbox is easy.

First, create a new directory for your sandbox inside `$XDG_CONFIG_HOME/sandbox-mcp/sandboxes`. For example, let's create a sandbox called `my-sandbox`:

```bash
mkdir $XDG_CONFIG_HOME/sandbox-mcp/sandboxes/my-sandbox
cd $XDG_CONFIG_HOME/sandbox-mcp/sandboxes/my-sandbox
```

A sandbox consists of a Dockerfile and a JSON configuration file. Both of these files are used to create and configure the sandbox.

The `my-sandbox` sandbox will be a simple Linux environment with a few tools pre-installed. So its Dockerfile will look like this:

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

Now we have to create the JSON configuration file. This file contains the configuration for the sandbox, including the name and description, command to run, and container configuration.

For our example, `my-sandbox`, we will create aa `config.json` file with the following configurations:

```json
{
	"name": "my-sandbox",
	"description": "A simple Linux sandbox.",
	"version": "0.1.0",
	"image": "sandbox-mcp/my-sandbox:latest",
	"user": "sandbox",
	"entrypoint": "main.sh",
	"timeout": 5,
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

Here's what's happening:

1. The `entrypoint` is the file where the input from the LLM is stored to be executed as described by `command`.
2. The `timeout` is the execution time limit. This can be useful to prevent the sandbox from running indefinitely.
3. The `parameters` property allows you to configure additional parameters for the sandbox tool. Here, we set `additionalFiles` to `true`, which allows the LLMs to pass additional files. Another valid value is `files`, which let's you configure any required files which should be passed along with the `entrypoint`. See the [go sandbox](/go/config.json) for an example.
4. The `security`, `resources`, and `mount` properties directly translate to Docker container configurations.

Once you have configured the sandbox, you can reload the MCP host/client application you are using (e.g., Cursor IDE or Claude Desktop) to apply the changes. You will see `my-sandbox` in the list of available tools.

Feel free to share the sandboxes your create with the community!
