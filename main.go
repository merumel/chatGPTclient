package main

import (
	"fmt"
	"log"
	"os"

	"github.com/charmbracelet/bubbles/textarea"
	tea "github.com/charmbracelet/bubbletea"
	openai "github.com/sashabaranov/go-openai"
)
var client *openai.Client
func main() {
	if len(os.Getenv("DEBUG")) > 0 {
		f, err := tea.LogToFile("debug.log", "debug")
		if err != nil {
			fmt.Println("fatal:", err)
			os.Exit(1)
		}
		defer f.Close()
	}
	p := tea.NewProgram(initialModel())

	if _, err := p.Run(); err != nil {
		log.Fatal(err)
	}
}


func (m model) Init() tea.Cmd {
	initializeClient()
	return tea.Batch(
		textarea.Blink,
		m.spinner.Tick,
		waitForActivity(m.chatGPTChan),
	)
}

