package tools

import (
	"fmt"

	"context"

	"github.com/aws/aws-sdk-go-v2/service/bedrockruntime/document"
	"github.com/aws/aws-sdk-go-v2/service/bedrockruntime/types"
)

func Execute(toolName string, toolUseId *string, input document.Interface) (types.ContentBlockMemberToolResult, error) {
	switch toolName {
	case "get_local_time":
		out, err := GetLocalTime(context.Background(), input)
		if err != nil {
			return types.ContentBlockMemberToolResult{}, fmt.Errorf("Error: %v", err)
		}
		out.Value.ToolUseId = toolUseId
		return out, nil
	default:
		fmt.Println("Invalid tool name")
		return types.ContentBlockMemberToolResult{}, fmt.Errorf("Invalid tool name")
	}
}
