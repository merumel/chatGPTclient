package main

import (
	"fmt"
	"log"
	"os"
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
	textMaxWidth int
	altscreen bool
	isReady bool
	viewportOffset int
	err         error
}
type errMsg struct{ err error }
var(
	helpStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("241")).Render
)
const useHighPerformanceRenderer = true

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
	
	s := spinner.New()
	s.Spinner = spinner.Globe
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("5"))

	ta.KeyMap.InsertNewline.SetEnabled(false)
	return model{
		textarea:    ta,
		messages:    []string{}, //needs to be new type to record senders to pass to api
		senderStyle: lipgloss.NewStyle().Foreground(lipgloss.Color("5")),
		chatGPTChan: make(chan chatGPTResponseMessage), //channel to receive chatgpt messages
		spinner: s,
		isWaitingForResponse: false, //for spinning ticker
		textMaxWidth: 120,
		altscreen: true, //fullscreen
		isReady: true, //for pager
		err:         nil,
	}
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var (
		cmds []tea.Cmd
		tiCmd tea.Cmd
		vpCmd tea.Cmd
	)
	f, err := tea.LogToFile("debug.log", "debug")
		if err != nil {
			fmt.Println("fatal:", err)
			os.Exit(1)
		}
		defer f.Close()
	

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyCtrlC, tea.KeyEsc:
			fmt.Println(m.textarea.Value())
			return m, tea.Quit
		case tea.KeyF10:
			var cmd tea.Cmd
			if m.altscreen {
				cmd = tea.ExitAltScreen
			} else {
				cmd = tea.EnterAltScreen
			}
			m.altscreen = !m.altscreen
			cmds = append(cmds, cmd)
		case tea.KeyEnter:
			userText := m.textarea.Value()
			m.messages = append(m.messages, m.senderStyle.Render("You: ")+m.textarea.Value())
			m.viewport.SetContent(strings.Join(m.messages, "\n"))			
			m.textarea.Reset()
			m.isWaitingForResponse = true
			log.Println("message content: " + strings.Join(m.messages, "\n")) 
			m.viewport.GotoBottom()
			cmds = append(cmds, getChatGPTResponse(userText, m.chatGPTChan), m.spinner.Tick)		
		}
	case tea.WindowSizeMsg:
		textAreaHeight := m.textarea.Height()
		//footerHeight := lipgloss.Height("")
		//verticalMarginHeight := textAreaHeight + 2

		if !m.isReady {
			// Since this program is using the full size of the viewport we
			// need to wait until we've received the window dimensions before
			// we can initialize the viewport. The initial dimensions come in
			// quickly, though asynchronously, which is why we wait for them
			// here.

			m.viewport = viewport.New(msg.Width, (msg.Height- textAreaHeight - 5))
			m.viewport.YPosition = textAreaHeight + 2
			m.viewport.HighPerformanceRendering = useHighPerformanceRenderer

			m.isReady = true

			// This is only necessary for high performance rendering, which in
			// most cases you won't need.
			//
			// Render the viewport one line below the header.
			m.viewport.YPosition = textAreaHeight + 3
		} else {
			m.viewport.Width = msg.Width
			m.viewport.Height = msg.Height- textAreaHeight - 5
		}

		if useHighPerformanceRenderer {
			// Render (or re-render) the whole viewport. Necessary both to
			// initialize the viewport and when the window is resized.
			//
			// This is needed for high-performance rendering only.
			cmds = append(cmds, viewport.Sync(m.viewport))
		}
		//m.viewportOffset = max(0, m.viewport.Height - len(m.messages))
		//m.viewport.YOffset = m.viewportOffset
		m.viewport.SetContent(`Welcome to the chat room!
	Type a message and press Enter to send.`)
	case chatGPTResponseMessage:
		gptMsg := chatGPTResponseMessage(msg)
		if gptMsg.err != nil{
			m.messages = append(m.messages, m.senderStyle.Render("ChatGPT Error: ")+gptMsg.err.Error())
		} else {
			m.messages = append(m.messages, m.senderStyle.Render("ChatGPT: ")+gptMsg.chatGptResponse)
		}
		//log.Println("message content: " + strings.Join(m.messages, "\n")) 
		m.isWaitingForResponse = false
		m.viewportOffset = max(0, m.viewport.Height - len(m.messages))
    	m.viewport.YOffset = m.viewportOffset
		m.viewport.SetContent(strings.Join(m.messages, "\n"))
		m.viewport.GotoBottom()
		m.textarea.Reset()
	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	// We handle errors just like any other message
	case errMsg:
		m.err = msg
		return m, nil
	}
	m.textarea, tiCmd = m.textarea.Update(msg)
	m.viewport, vpCmd = m.viewport.Update(msg)
	cmds = append(cmds, tiCmd)
	cmds = append(cmds, vpCmd)
	cmds = append(cmds, waitForActivity(m.chatGPTChan))

	return m, tea.Batch(cmds...) //the '...' is expanind the cmd slice to pass to batch

}
func waitForActivity(sub chan chatGPTResponseMessage) tea.Cmd{
	return func() tea.Msg{
		msg := <-sub
		if len(msg.chatGptResponse) > 150 {
			splittedMessages := splitIntoLines(msg.chatGptResponse, 120)
			msg.chatGptResponse = strings.Join(splittedMessages, "\n")
		}
		return msg	
	}
}
func splitIntoLines(str string, maxLen int) []string {
	var lines []string
	words := strings.Fields(str)
	var currentLine string

	for _, word := range words {
		if len(currentLine)+len(word) > maxLen {
			lines = append(lines, currentLine)
			currentLine = word
		} else {
			if currentLine != "" {
				currentLine += " "
			}
			currentLine += word
		}
	}
	if currentLine != "" {
		lines = append(lines, currentLine)
	}

	return lines
}
func max(a, b int) int {
	if a > b {
		return a
	}
	return b
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
	) + "\n"+ helpStyle("Press Enter to send message. Ctrl + c to quit. ↑/↓ to scroll up and down") +
	 "\n\n"
}


// For messages that contain errors it's often handy to also implement the
// error interface on the message.
func (e errMsg) Error() string { return e.err.Error() }

