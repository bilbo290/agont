package main

import (
	"agont/tools"
	"bufio"
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/bedrockruntime"
	"github.com/aws/aws-sdk-go-v2/service/bedrockruntime/types"
)

const region = "ap-southeast-1"

// Each model provider defines their own individual request and response formats.
// For the format, ranges, and default values for the different models, refer to:
// https://docs.aws.amazon.com/bedrock/latest/userguide/model-parameters.html

type ClaudeRequest struct {
	Prompt            string `json:"prompt"`
	MaxTokensToSample int    `json:"max_tokens_to_sample"`
	// Omitting optional request parameters
}

type ClaudeResponse struct {
	Completion string `json:"completion"`
}

func main() {
	// Initialize AWS Bedrock client

	ctx := context.Background()
	sdkConfig, err := config.LoadDefaultConfig(ctx, config.WithRegion(region))
	if err != nil {
		fmt.Println("Couldn't load default configuration. Have you set up your AWS account?")
		fmt.Println(err)
		return
	}

	client := bedrockruntime.NewFromConfig(sdkConfig)

	scanner := bufio.NewScanner(os.Stdin)
	getUserMessage := func() (string, bool) {
		if !scanner.Scan() {
			return "", false
		}
		return scanner.Text(), true
	}

	// get argument from run command for --stream
	stream := false
	if len(os.Args) > 1 && os.Args[1] == "--stream" {
		stream = true
	}

	agent := NewAgent(client, getUserMessage, stream)
	err = agent.Run(ctx)
	if err != nil {
		fmt.Println("Error running agent:", err)
	}
}

type Agent struct {
	client         *bedrockruntime.Client
	getUserMessage func() (string, bool)
	stream         bool
	tools          types.ToolConfiguration
}

func NewAgent(client *bedrockruntime.Client, getUserMessage func() (string, bool), stream bool) *Agent {
	// getLocalTime := func() string {
	// 	return time.Now().Format("2006-01-02 15:04:05")
	// }

	// minimalSchema := map[string]interface{}{
	// 	"type": "object",
	// 	"properties": map[string]interface{}{
	// 		"title": map[string]interface{}{
	// 			"type":        "string",
	// 			"description": "A sample title to be generated", // Simple description focused on generation
	// 		},
	// 	},
	// 	"required": []string{"title"},
	// }

	// toolName := "generate_sample_title" // Naming reflects the task
	// toolDesc := "Use this tool to generate a sample title."
	// toolSpec := types.ToolMemberToolSpec{
	// 	Value: types.ToolSpecification{
	// 		Name:        &toolName,
	// 		Description: &toolDesc,
	// 		InputSchema: &types.ToolInputSchemaMemberJson{
	// 			// Use NewLazyDocument to correctly serialize the schema map
	// 			Value: document.NewLazyDocument(minimalSchema),
	// 		},
	// 	},
	// }
	//

	getLocalTimeConfig := tools.GetLocalTimeToolConfig()
	tools := []types.Tool{&getLocalTimeConfig}
	// 3. Define the minimal tool configuration (NO ToolChoice)
	toolConfig := types.ToolConfiguration{
		Tools: tools,
		// Intentionally omit ToolChoice to allow natural selection
	}

	return &Agent{
		client:         client,
		getUserMessage: getUserMessage,
		stream:         stream,
		tools:          toolConfig,
	}
}

func (a *Agent) Run(ctx context.Context) error {
	//conversation :=
	fmt.Println("Chat with Bedrock (use 'ctrl-c' to quit), stream-mode:", a.stream)
	conversation := []types.Message{}
	readUserInput := true
	for {
		//fmt.Println("Skipping user input", readUserInput)
		if readUserInput {
			fmt.Print("\u001b[94mYou\u001b[0m: ")
			userInput, ok := a.getUserMessage()
			if !ok {
				break
			}
			//fmt.Println("What you type: ", userInput)
			userMsg := &types.ContentBlockMemberText{
				Value: userInput,
			}
			content := []types.ContentBlock{}

			content = append(content, userMsg)
			conversation = append(conversation, types.Message{
				Role:    types.ConversationRoleUser,
				Content: content,
			})
		}

		message, err := a.converse(ctx, conversation)
		if err != nil {
			fmt.Println("Error running inference:", err)
			return err
		}
		// message, err := a.runInference(ctx, conversation)
		// if err != nil {
		// 	fmt.Println("Error running inference:", err)
		// 	return err
		// }
		// // append response back to conversation
		var toolResults = []types.ContentBlock{}
		for _, block := range message {
			conversation = append(conversation, types.Message{
				Role:    types.ConversationRoleAssistant,
				Content: block.Content,
			})
			for _, content := range block.Content {
				switch content.(type) {
				case *types.ContentBlockMemberText:
					//fmt.Printf("\u001b[92mAgont\u001b[0m: %s\n", content.(*types.ContentBlockMemberText).Value)
				case *types.ContentBlockMemberImage:
					//fmt.Printf("\u001b[92mAgont\u001b[0m: Image\n")

				case *types.ContentBlockMemberToolUse:
					contentToolUse := content.(*types.ContentBlockMemberToolUse)
					out, err := tools.Execute(*contentToolUse.Value.Name, contentToolUse.Value.ToolUseId, contentToolUse.Value.Input)
					if err != nil {
						break
					}
					toolResults = append(toolResults, &out)
					//fmt.Println("how many time")
				default:
					fmt.Printf("\u001b[92mAgont\u001b[0m: Unknown\n")
				}
			}

		}
		//fmt.Println("conversation", conversation)
		// for _, message := range conversation {
		// 	for _, content := range message.Content {
		// 		//fmt.Println("DEBUG", message.Role, content)
		// 	}
		// 	//fmt.Println("DEBUG", message.Role, message.Content)
		// }
		if len(toolResults) == 0 {
			readUserInput = true
			continue
		}
		readUserInput = false
		conversation = append(conversation, types.Message{
			Role:    types.ConversationRoleUser,
			Content: toolResults,
		})
		//fmt.Print("\u001b[33mtools\u001b[0m: ", toolResults)
		// switch v := message.(type) {
		// case *types.ContentBlockMemberText:
		// 	fmt.Printf("\u001b[92mAgont\u001b[0m: %s\n", v.Value)
		// case *types.ContentBlockMemberImage:
		// 	fmt.Printf("\u001b[92mAgont\u001b[0m: Image\n")
		// default:
		// 	fmt.Printf("\u001b[92mAgont\u001b[0m: Unknown\n")
		// }
	}
	return nil
}

func (a *Agent) runInference(ctx context.Context, conversation []types.Message) (*bedrockruntime.ConverseOutput, error) {
	modelId := "arn:aws:bedrock:ap-southeast-1:851725417117:inference-profile/apac.anthropic.claude-sonnet-4-20250514-v1:0"
	response, err := a.client.Converse(ctx, &bedrockruntime.ConverseInput{
		ModelId:    &modelId,
		Messages:   conversation,
		System:     []types.SystemContentBlock{},
		ToolConfig: &a.tools,
	})
	if err != nil {
		fmt.Println("Error running inference:", err)
		return nil, err
	}

	return response, nil
}

func (a *Agent) runInferenceStream(ctx context.Context, conversation []types.Message) (*bedrockruntime.ConverseStreamOutput, error) {
	modelId := "anthropic.claude-v2"
	output, err := a.client.ConverseStream(ctx, &bedrockruntime.ConverseStreamInput{
		ModelId:    &modelId,
		Messages:   conversation,
		System:     []types.SystemContentBlock{},
		ToolConfig: &a.tools,
	})
	if err != nil {
		fmt.Println("Error running inference:", err)
		return nil, err
	}

	return output, nil
}

func (a *Agent) converse(ctx context.Context, conversation []types.Message) ([]types.Message, error) {
	switch a.stream {
	case false:
		response, err := a.runInference(ctx, conversation)
		if err != nil {
			return nil, err
		}

		// get tools
		responseText, _ := response.Output.(*types.ConverseOutputMemberMessage)
		var textBlock []types.ContentBlock
		for _, content := range responseText.Value.Content {
			//fmt.Printf("\u001b[92mAgont-Debug\u001b[0m: %s\n", tool.(*types.))
			//
			switch content := content.(type) {
			case *types.ContentBlockMemberText:
				fmt.Printf("\u001b[92mAgont\u001b[0m: %s\n", content.Value)
				textBlock = append(textBlock, content)
			case *types.ContentBlockMemberImage:
				fmt.Printf("\u001b[92mAgont\u001b[0m: Image\n")
			case *types.ContentBlockMemberToolUse:
				toolUse := content
				fmt.Printf("\u001b[92mAgont\u001b[0m: 🚀 Activating tool [%s]: %s\n", *toolUse.Value.Name, *toolUse.Value.ToolUseId)
				fmt.Printf("\u001b[92mAgont\u001b[0m: 🛠️  Input: %s\n", *&toolUse.Value.Input)
				fmt.Printf("\u001b[92mAgont\u001b[0m: ...Let me handle that for you!\n")
				//var toolInput document.Interface
				//toolUse.Value.Input.UnmarshalSmithyDocument(&toolInput)

				textBlock = append(textBlock, content)
			default:
				fmt.Printf("\u001b[92mAgont\u001b[0m: Unknown\n")
			}
		}

		return []types.Message{
			{
				Role:    types.ConversationRoleAssistant,
				Content: textBlock,
			},
		}, nil
	case true:
		output, err := a.runInferenceStream(ctx, conversation)
		if err != nil {
			return nil, err
		}
		var combinedResult string

		msg := types.Message{}

		for event := range output.GetStream().Events() {
			switch v := event.(type) {
			case *types.ConverseStreamOutputMemberMessageStart:

				msg.Role = v.Value.Role
				fmt.Print("\u001b[92mAgont\u001b[0m: ")
			case *types.ConverseStreamOutputMemberContentBlockDelta:

				textResponse := v.Value.Delta.(*types.ContentBlockDeltaMemberText)
				//handler(context.Background(), )
				fmt.Print(textResponse.Value)
				combinedResult = combinedResult + textResponse.Value

			case *types.UnknownUnionMember:
				fmt.Println("unknown tag:", v.Tag)
			}
		}
		msg.Role = types.ConversationRoleAssistant
		msg.Content = append(msg.Content, &types.ContentBlockMemberText{
			Value: combinedResult,
		})
		fmt.Println()
		return []types.Message{msg}, nil
	}

	return nil, errors.New("invalid stream value")
}

type ToolDefinition struct {
	Name        string
	Description string
}
