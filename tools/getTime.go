package tools

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/bedrockruntime/document"
	"github.com/aws/aws-sdk-go-v2/service/bedrockruntime/types"
)

func GetLocalTimeSchema() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"timezone": map[string]interface{}{
				"type":        "string",
				"description": "The timezone for the current local time",
			},
		},
		"required": []string{"timezone"},
	}
}

// GetLocalTimeInput is the input schema for GetLocalTime tool.
type GetLocalTimeInput struct {
	Timezone string `json:"timezone"` // e.g., "America/New_York"
}

// GetLocalTimeOutput is the output schema for GetLocalTime tool.
type GetLocalTimeOutput struct {
	LocalTime string `json:"local_time"` // e.g., "2024-06-10T15:04:05-07:00"
}

// GetLocalTime returns the current local time for the given timezone.
func GetLocalTime(ctx context.Context, input document.Interface) (types.ContentBlockMemberToolResult, error) {
	// parse input to GetLocalTimeInput
	var inputStruct GetLocalTimeInput
	err := input.UnmarshalSmithyDocument(&inputStruct)
	if err != nil {
		return types.ContentBlockMemberToolResult{}, fmt.Errorf("Error parsing input: %v", err)
	}

	loc, err := time.LoadLocation(inputStruct.Timezone)
	if err != nil {
		return types.ContentBlockMemberToolResult{}, err
	}
	now := time.Now().In(loc)
	//fmt.Println("Local time:", now.Format(time.RFC3339))

	result := &types.ToolResultContentBlockMemberText{
		Value: now.Format(time.RFC3339),
	}
	toolResult := types.ContentBlockMemberToolResult{
		Value: types.ToolResultBlock{
			Content: []types.ToolResultContentBlock{result},
			//ToolUseId: new(string),
			//Status:    "",
		},
	}
	return toolResult, nil
}

// Example: Get the Bedrock tool configuration for GetLocalTime.
func GetLocalTimeToolConfig() types.ToolMemberToolSpec {
	toolName := "get_local_time" // Naming reflects the task
	toolDesc := "Use this tool to generate a sample title."
	return types.ToolMemberToolSpec{
		Value: types.ToolSpecification{
			Name:        &toolName,
			Description: &toolDesc,
			InputSchema: &types.ToolInputSchemaMemberJson{
				// Use NewLazyDocument to correctly serialize the schema map
				Value: document.NewLazyDocument(GetLocalTimeSchema()),
			},
		},
	}
}
