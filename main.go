package main

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"net"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"charm.land/wish/v2"
	"charm.land/wish/v2/activeterm"
	"charm.land/wish/v2/bubbletea"
	"charm.land/wish/v2/logging"
	"github.com/charmbracelet/ssh"
)

const (
	host = "localhost"
	port = "23234"
)

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~
// |					MAIN FUNCTION					|
// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

func main() {
	s, err := wish.NewServer(
		wish.WithAddress(net.JoinHostPort(host, port)),
		wish.WithHostKeyPath(".ssh/id_ed25519"),
		wish.WithMiddleware(
			bubbletea.Middleware(fortune),
			activeterm.Middleware(), // Bubble Tea apps usually require a PTY.
			logging.Middleware(),
		),
	)
	if err != nil {
		fmt.Errorf("Could not start server", "error", err)
	}

	done := make(chan os.Signal, 1)
	signal.Notify(done, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)
	fmt.Printf("Starting SSH server", "host", host, "port", port)
	go func() {
		if err = s.ListenAndServe(); err != nil && !errors.Is(err, ssh.ErrServerClosed) {
			fmt.Errorf("Could not start server", "error", err)
			done <- nil
		}
	}()

	<-done
	// log.Info("Stopping SSH server")
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer func() { cancel() }()
	if err := s.Shutdown(ctx); err != nil && !errors.Is(err, ssh.ErrServerClosed) {
		// log.Error("Could not stop server", "error", err)
	}
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~
// |					MIDDLEWARE						|
// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// You can wire any Bubble Tea model up to the middleware with a function that
// handles the incoming ssh.Session. Here we just grab the terminal info and
// pass it to the new model. You can also return tea.ProgramOptions (such as
// tea.WithAltScreen) on a session by session basis.

func fortune(s ssh.Session) (tea.Model, []tea.ProgramOption) {
	pty, _, _ := s.Pty()
	m := model{
		term:      pty.Term,
		width:     pty.Window.Width,
		height:    pty.Window.Height,
		infoStyle: lipgloss.NewStyle().Foreground(lipgloss.Color("#851104")).Align(lipgloss.Center),
		txtStyle:  lipgloss.NewStyle().Foreground(lipgloss.Color("#1d77b8")).Align(lipgloss.Center),
		quitStyle: lipgloss.NewStyle().Foreground(lipgloss.Color("#d75218")),
		window:    tea.WindowSizeMsg{Width: pty.Window.Width, Height: pty.Window.Height},
		bg:        "light",
	}
	return m, []tea.ProgramOption{}
}

// Just a generic tea.Model to demo terminal information of ssh.
type model struct {
	term      string
	profile   string
	width     int
	height    int
	bg        string
	infoStyle lipgloss.Style
	txtStyle  lipgloss.Style
	quitStyle lipgloss.Style
	window    tea.WindowSizeMsg
	message   string
}

func (m model) Init() tea.Cmd {
	// default values
	return tea.Batch(
		tea.RequestBackgroundColor,
	)
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.ColorProfileMsg:
		m.profile = msg.String()
	case tea.BackgroundColorMsg:
		if msg.IsDark() {
			m.bg = "dark"
		}
	case tea.WindowSizeMsg:
		m.height = msg.Height
		m.width = msg.Width
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "esc":
			return m, tea.Quit
		case "c", "enter":
			specialNumber := createSeed()
			fortune := fetchAFortune(specialNumber)
			m.message = fortune
		}
	}
	return m, nil
}

func createSeed() int {
	now := time.Now()
	timeString := now.Format("2006-01-02 15:04:05 MST")
	hash := sha256.Sum256([]byte(timeString))
	seed := int(hash[0])%253 + 1
	return seed
}

func fetchAFortune(seed int) string {
	content, err := os.ReadFile("fortunes.json")
	if err != nil {
	}
	var fortunes []string
	err = json.Unmarshal(content, &fortunes)
	if err != nil {
	}
	return fortunes[seed]
}

func rules() string {

	s := "Press ENTER/'c' to have your fortune told 🥠"

	return s
}

const asciiChars = " .:-=+*#%@"

func renderCookie() ([]string, error) {

	file, err := os.Open("cookie.png")
	if err != nil {
		fmt.Println("Error opening file:", err)
	}
	defer file.Close()
	img, _, err := image.Decode(file)
	if err != nil {
		fmt.Println("Error decoding image:", err)
		return nil, err
	}
	bounds := img.Bounds()
	width, height := bounds.Max.X, bounds.Max.Y
	lines := make([]string, 0, height)

	for y := 0; y <= height; y++ {
		var line strings.Builder
		line.Grow(width)

		for x := 0; x <= width; x++ {
			r, g, b, a := img.At(bounds.Min.X+x, bounds.Min.Y+y).RGBA()

			if a == 0 {
				line.WriteByte(' ')
				continue
			}

			color := lipgloss.Color(fmt.Sprintf("#%02X%02X%02X",
				uint8(r>>8),
				uint8(g>>8),
				uint8(b>>8),
			))
			// if color != lipgloss.Color("#FFFFFF") {
			line.WriteString(
				lipgloss.NewStyle().
					Foreground(color).
					Background(color).
					Align(lipgloss.Center).
					Render(".."),
			)
			// }
		}
		lines = append(lines, line.String())
	}
	return lines, err
}

func (m model) View() tea.View {

	background := lipgloss.Color("#f5d79f")

	ascii, err := renderCookie()
	if err != nil {
		ascii = []string{"<failed to load sprite>"}
	}

	sprite := strings.Join(ascii, "\n")
	s := rules()
	rulesBox := lipgloss.NewStyle().
		Border(lipgloss.NormalBorder(), true, true, true, true).
		BorderForeground(lipgloss.Color("#d75218")).
		BorderBackground(background).
		Foreground(lipgloss.Color("#782907")).
		PaddingRight(1).
		PaddingLeft(1).
		Background(background).
		Bold(true).
		Align(lipgloss.Center).
		Render(s)

	content := rulesBox +
		"\n\n" +
		m.infoStyle.Render(m.message) +
		"\n\n" +
		// lipgloss.NewStyle().Background(background).Render(sprite) +
		sprite +
		"\n" +
		m.quitStyle.Render("Press 'q' to quit")

	bg := lipgloss.NewStyle().
		Background(background)
		// Foreground(lipgloss.Color("#cdd6f4"))

	fortunecookies := lipgloss.NewStyle().
		Background(background).
		Border(lipgloss.DoubleBorder(), true, true, false, true).
		BorderBackground(background).
		BorderForeground(lipgloss.Color("#782907")).
		Width(m.width).
		Height(m.height).
		Render(lipgloss.Place(
			m.width,
			m.height,
			lipgloss.Center,
			lipgloss.Center,
			content,
		))

	v := tea.NewView((lipgloss.Place(
		m.width,
		m.height,
		lipgloss.Center,
		lipgloss.Center,
		fortunecookies,
		lipgloss.WithWhitespaceChars(" "),
		lipgloss.WithWhitespaceStyle(bg),
	)))
	v.AltScreen = true
	return v
}
