package main

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)
type model struct {
	viewport    viewport.Model
	messages    []string
	textarea    textarea.Model
	senderStyle lipgloss.Style
	chatGPTChan chan chatGPTResponseMessage
	spinner spinner.Model
	isWaitingForResponse bool
	err         error
}
type errMsg struct{ err error }

func initialModel() model {
	ta := textarea.New()
	ta.Placeholder = "Send a message..."
	ta.Focus()

	ta.Prompt = "┃ "
	ta.CharLimit = 280

	ta.SetWidth(120)
	ta.SetHeight(3)

	// Remove cursor line styling
	ta.FocusedStyle.CursorLine = lipgloss.NewStyle()

	ta.ShowLineNumbers = false

	vp := viewport.New(120, 5)
	vp.SetContent(`Welcome to the chat room!
Type a message and press Enter to send.`)

	s := spinner.New()
	s.Spinner = spinner.Globe
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("5"))

	ta.KeyMap.InsertNewline.SetEnabled(false)
	return model{
		textarea:    ta,
		messages:    []string{},
		viewport:    vp,
		senderStyle: lipgloss.NewStyle().Foreground(lipgloss.Color("5")),
		chatGPTChan: make(chan chatGPTResponseMessage),
		spinner: s,
		isWaitingForResponse: false,
		err:         nil,
	}
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var (
		tiCmd tea.Cmd
		vpCmd tea.Cmd
	)

	m.textarea, tiCmd = m.textarea.Update(msg)
	m.viewport, vpCmd = m.viewport.Update(msg)

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyCtrlC, tea.KeyEsc:
			fmt.Println(m.textarea.Value())
			return m, tea.Quit
		case tea.KeyEnter:
			userText := m.textarea.Value()
			m.messages = append(m.messages, m.senderStyle.Render("You: ")+m.textarea.Value())
			m.viewport.SetContent(strings.Join(m.messages, "\n"))			
			m.textarea.Reset()
			m.viewport.GotoBottom()
			m.isWaitingForResponse = true
			return m, tea.Batch(getChatGPTResponse(userText, m.chatGPTChan), m.spinner.Tick)		
		}
	case chatGPTResponseMessage:
		gptMsg := chatGPTResponseMessage(msg)
		if gptMsg.err != nil{
			m.messages = append(m.messages, m.senderStyle.Render("ChatGPT Error: ")+gptMsg.err.Error())
		} else {
			m.messages = append(m.messages, m.senderStyle.Render("ChatGPT: ")+gptMsg.chatGptResponse)
		}
		m.isWaitingForResponse = false
		m.viewport.SetContent(strings.Join(m.messages, "\n"))
		m.textarea.Reset()
		m.viewport.GotoBottom()
	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	// We handle errors just like any other message
	case errMsg:
		m.err = msg
		return m, nil
	}

	return m, tea.Batch(tiCmd, vpCmd, waitForActivity(m.chatGPTChan))
}
func waitForActivity(sub chan chatGPTResponseMessage) tea.Cmd{
	return func() tea.Msg{
		return chatGPTResponseMessage(<-sub)
	}
}
func (m model) View() string {
	spinnerView := ""
	if m.isWaitingForResponse {
		spinnerView = m.spinner.View()+ " Thinking..."
	}
	return fmt.Sprintf(
		"%s\n%s\n%s",
		m.viewport.View(),
		spinnerView,
		m.textarea.View(),
	) + "\n\n"
}


// For messages that contain errors it's often handy to also implement the
// error interface on the message.
func (e errMsg) Error() string { return e.err.Error() }

