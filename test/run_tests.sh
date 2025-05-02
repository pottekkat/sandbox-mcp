#!/bin/bash
set -e

# Directory containing test files
# NOTE: Relative to the root
TEST_DIR="test/test_files"

# Variable to track if any tests failed
FAILED=0

echo "Running tests..."

for test_file in "$TEST_DIR"/*.json; do
    # Skip config.json as it's not a test file
    if [[ "$(basename "$test_file")" == "config.json" ]]; then
        continue
    fi
    
    tool_name=$(basename "$test_file" .json)
    echo "→ Testing tool: $tool_name"

    # Check if the tool exists in the server
    if ! npx @modelcontextprotocol/inspector --cli --config test/config.json --server sandbox-mcp --method tools/list | jq -e ".tools[] | select(.name == \"$tool_name\")" > /dev/null; then
        echo "  ⚠️  Tool '$tool_name' does not exist on the server, skipping tests"
        continue
    fi

    test_count=$(jq length "$test_file")
    test_failed=0
    
    for (( i=0; i<$test_count; i++ )); do
        echo "  • Test case $((i+1))/$test_count"
        
        # Create base command using the inspector CLI
        cmd_parts=("npx" "@modelcontextprotocol/inspector" "--cli" "--config" "test/config.json" "--server" "sandbox-mcp" "--method" "tools/call" "--tool-name" "$tool_name")
        
        # Get all keys in the request object and add them as tool arguments
        for key in $(jq -r ".[$i].request | keys[]" "$test_file"); do
            type=$(jq -r ".[$i].request[\"$key\"] | type" "$test_file")
            
            if [ "$type" = "string" ]; then
                value=$(jq -r ".[$i].request[\"$key\"]" "$test_file")
                cmd_parts+=("--tool-arg" "$key=$value")
            elif [ "$type" = "array" ] && [ "$key" = "files" ]; then
                files_count=$(jq -r ".[$i].request.files | length" "$test_file")
                files_arg="files=["
                
                for (( j=0; j<$files_count; j++ )); do
                    filename=$(jq -r ".[$i].request.files[$j].filename" "$test_file")
                    content=$(jq -r ".[$i].request.files[$j].content" "$test_file")
                    files_arg+="{\"filename\":\"$filename\",\"content\":\"$content\"}"
                    [ $j -lt $((files_count-1)) ] && files_arg+=","
                done
                
                files_arg+="]"
                cmd_parts+=("--tool-arg" "$files_arg")
            fi
        done
        
        # Execute the inspector CLI command and capture output
        output=$("${cmd_parts[@]}" 2>&1)
        
        # Extract expected and actual values
        expected_text=$(jq -r ".[$i].response.text" "$test_file" | tr -d '\r')
        # By default, the response is not an error
        expected_is_error=$(jq -r ".[$i].response.isError // false" "$test_file")
        
        # Validate JSON response
        if ! echo "$output" | jq -e . >/dev/null 2>&1; then
            echo "    ❌ Invalid JSON response"
            echo "    Output: $output"
            test_failed=1
            continue
        fi
        
        # Extract text content and error status from the response
        actual_text=$(echo "$output" | jq -r '.content[] | select(.type == "text") | .text' | tr -d '\r')
        actual_is_error=$(echo "$output" | jq -r '.isError // false')
        
        # Check if response matches expectations
        if [[ "$actual_text" == *"$expected_text"* ]] && [ "$expected_is_error" = "$actual_is_error" ]; then
            echo "    ✓ Passed"
        else
            echo "    ✗ Failed"
            [ "$expected_is_error" != "$actual_is_error" ] && echo "      Error flag mismatch: expected=$expected_is_error, actual=$actual_is_error"
            echo "      Expected text: '$expected_text'"
            echo "      Actual text:   '$actual_text'"
            test_failed=1
        fi
    done
    
    if [ $test_failed -eq 0 ]; then
        echo "  ✅ All tests passed for $tool_name"
    else
        echo "  ❌ Tests failed for $tool_name"
        FAILED=1
    fi
    
    echo ""
done

if [ $FAILED -eq 0 ]; then
    echo "✅ All tests completed successfully"
    exit 0
else
    echo "❌ Some tests failed"
    exit 1
fi