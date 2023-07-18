package main

import (
	"context"
	"encoding/json"
	"log"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	openai "github.com/sashabaranov/go-openai"
)
type Role string

const (
	User      Role = "user"
	Assistant Role = "assistant"
	System    Role = "system"
)
type chatGPTMessage struct{
	Content string
	Role Role
	Err error
}
type Configuration struct {
	API_KEY string
}
var config Configuration

func initializeClient() {
	config = loadConfiguration("./config.json")
	client = openai.NewClient(config.API_KEY)
}
func loadConfiguration(file string) Configuration {
	// Open the file
	f, err := os.Open(file)
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()

	// Initialize configuration
	var config Configuration

	// Decode the file into Configuration struct
	decoder := json.NewDecoder(f)
	err = decoder.Decode(&config)
	if err != nil {
		log.Fatal(err)
	}
	return config
}

func getChatGPTResponse(messages []chatGPTMessage, sub chan chatGPTMessage) tea.Cmd{
	return func() tea.Msg{
		var openaiMessages []openai.ChatCompletionMessage
		for _, msg := range messages {
			openaiMessages = append(openaiMessages, openai.ChatCompletionMessage{
				Role:    string(msg.Role),
				Content: msg.Content,
			})
		}

		resp, err := client.CreateChatCompletion(
			context.Background(),
			openai.ChatCompletionRequest{
				Model: openai.GPT3Dot5Turbo,
				Messages: openaiMessages,
			},
		)
		if err != nil{
			sub <- chatGPTMessage{
				Content: "",
				Role: User,
				Err: err,
			}
		} else{
			sub <- chatGPTMessage{
				Content: resp.Choices[0].Message.Content,
				Role: Role(resp.Choices[0].Message.Role),
				Err: nil,
			}
		}		
		return nil		
	}
}
