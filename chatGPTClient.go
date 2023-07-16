package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	openai "github.com/sashabaranov/go-openai"
)
type chatGPTResponseMessage struct{
	chatGptResponse string
	err error
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
	fmt.Printf("API Key: %s\n", config.API_KEY)

	return config
}

func getChatGPTResponse(chatMsg string, sub chan chatGPTResponseMessage) tea.Cmd{
	return func() tea.Msg{

		resp, err := client.CreateChatCompletion(
			context.Background(),
			openai.ChatCompletionRequest{
				Model: openai.GPT3Dot5Turbo,
				Messages: []openai.ChatCompletionMessage{
					{
						Role:    openai.ChatMessageRoleUser,
						Content: chatMsg,
					},
				},
			},
		)
		if err != nil{
			sub <- chatGPTResponseMessage{
				chatGptResponse: "",
				err: err,
			}
		} else{
			sub <- chatGPTResponseMessage{
				chatGptResponse: resp.Choices[0].Message.Content,
				err: nil,
			}
		}		
		return nil		
	}
}
