package sandbox

import "github.com/mark3labs/mcp-go/mcp"

// withEntrypoint creates an entrypoint parameter for the tool.
func withEntrypoint(name string, description string) mcp.ToolOption {
	return mcp.WithString(name, mcp.Required(), mcp.Description(description))
}

// withAdditionalFiles creates a files parameter for the tool.
func withAdditionalFiles() mcp.ToolOption {
	return mcp.WithArray("files",
		mcp.Description("Files to be included in the sandbox"),
		mcp.Items(map[string]any{
			"type": "object",
			"properties": map[string]any{
				"filename": map[string]any{
					"type": "string",
				},
				"content": map[string]any{
					"type": "string",
				},
			},
		}),
	)
}

// withFile adds a file parameter to the tool.
func withFile(name string, description string, required bool) mcp.ToolOption {
	if required {
		return mcp.WithString(name, mcp.Description(description), mcp.Required())
	}
	return mcp.WithString(name, mcp.Description(description))
}
