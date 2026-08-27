package tui

import (
	"fmt"
	"net"
	"os"
	"os/exec"
	"time"

	"gohide/internal/vpn"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

type pingMsg struct {
	index int
	rtt   time.Duration
	err   error
}

type Model struct {
	servers       []vpn.RawConfig
	pings         []string
	cursor        int
	offset        int // Индекс, с которого начинается отрисовка списка (для скролла)
	height        int // Высота терминала
	activeIdx     int
	activeCmd     *exec.Cmd
	activeCfgPath string
	execPath      string
}

func NewModel(configs []vpn.RawConfig, execPath string) Model {
	pings := make([]string, len(configs))
	for i := range pings {
		pings[i] = "-" // По умолчанию пинг не замерен
	}
	return Model{
		servers:   configs,
		pings:     pings,
		cursor:    0,
		offset:    0,
		activeIdx: -1,
		execPath:  execPath,
	}
}

func (m Model) Init() tea.Cmd {
	// Автоматический пинг при старте убран по твоей просьбе
	return nil
}

func (m Model) pingAll() tea.Cmd {
	var cmds []tea.Cmd
	for i, s := range m.servers {
		m.pings[i] = "Ping..."
		cmds = append(cmds, m.pingServerCmd(i, s.Address, s.Port))
	}
	return tea.Batch(cmds...)
}

func (m Model) pingServerCmd(index int, host, port string) tea.Cmd {
	return func() tea.Msg {
		start := time.Now()
		conn, err := net.DialTimeout("tcp", net.JoinHostPort(host, port), 2*time.Second)
		if err != nil {
			return pingMsg{index: index, err: err}
		}
		_ = conn.Close()
		return pingMsg{index: index, rtt: time.Since(start)}
	}
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	// Ловим изменение размера окна терминала для скролла
	case tea.WindowSizeMsg:
		m.height = msg.Height
		return m, nil

	case pingMsg:
		if msg.err != nil {
			m.pings[msg.index] = lipgloss.NewStyle().Foreground(lipgloss.Color("9")).Render("Error")
		} else {
			color := "10"
			if msg.rtt > 150*time.Millisecond {
				color = "11"
			}
			m.pings[msg.index] = lipgloss.NewStyle().Foreground(lipgloss.Color(color)).Render(fmt.Sprintf("%dms", msg.rtt.Milliseconds()))
		}
		return m, nil

	case tea.KeyMsg:
		// Вычисляем, сколько строк списка помещается на экране (минус шапка и подвал)
		visibleLines := m.height - 8
		if visibleLines < 1 {
			visibleLines = 1
		}

		switch msg.String() {
		case "ctrl+c", "q":
			m.stopSingBox()
			return m, tea.Quit

		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
				// Если курсор ушел выше видимой зоны, двигаем окно скролла вверх
				if m.cursor < m.offset {
					m.offset = m.cursor
				}
			}

		case "down", "j":
			if m.cursor < len(m.servers)-1 {
				m.cursor++
				// Если курсор ушел ниже видимой зоны, двигаем окно скролла вниз
				if m.cursor >= m.offset+visibleLines {
					m.offset = m.cursor - visibleLines + 1
				}
			}

		case "p":
			return m, m.pingAll()

		case "right", "l":
			// Пинг только конкретного конфига (стрелка вправо)
			m.pings[m.cursor] = "Ping..."
			return m, m.pingServerCmd(m.cursor, m.servers[m.cursor].Address, m.servers[m.cursor].Port)

		case "enter", "space":
			if m.activeIdx == m.cursor {
				m.stopSingBox()
			} else {
				m.stopSingBox()

				chosen := m.servers[m.cursor]
				configPath, err := vpn.BuildConfig(chosen)
				if err != nil {
					return m, nil
				}

				cmd := exec.Command(m.execPath, "run", "-c", configPath)
				if err := cmd.Start(); err == nil {
					m.activeIdx = m.cursor
					m.activeCmd = cmd
					m.activeCfgPath = configPath

					go func() { _ = cmd.Wait() }()
				}
			}
		}
	}
	return m, nil
}

func (m *Model) stopSingBox() {
	if m.activeCmd != nil && m.activeCmd.Process != nil {
		_ = m.activeCmd.Process.Kill()
		_ = m.activeCmd.Wait()
	}
	if m.activeCfgPath != "" {
		_ = os.Remove(m.activeCfgPath)
	}
	m.activeIdx = -1
	m.activeCmd = nil
	m.activeCfgPath = ""
}

func (m Model) View() tea.View {
	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("12")).
		Border(lipgloss.NormalBorder(), false, false, true, false).
		Padding(0, 1, 0, 1).
		MarginBottom(1)

	statusStyle := lipgloss.NewStyle().
		Bold(true).
		Padding(0, 1).
		MarginTop(1)

	s := titleStyle.Render("🛡️  GoHide VPN Client ") + "\n\n"

	visibleLines := m.height - 8
	if visibleLines < 1 {
		visibleLines = 1
	}

	end := m.offset + visibleLines
	if end > len(m.servers) {
		end = len(m.servers)
	}

	cursorColStyle := lipgloss.NewStyle().Width(4)
	nameColStyle := lipgloss.NewStyle().Width(55)
	pingColStyle := lipgloss.NewStyle().Width(12).Align(lipgloss.Right)

	for i := m.offset; i < end; i++ {
		server := m.servers[i]

		cursorStr := " "
		nameStr := server.Name

		if m.activeIdx == i {
			nameStr = lipgloss.NewStyle().Foreground(lipgloss.Color("10")).Bold(true).Render(server.Name)
		} else if m.cursor == i {
			nameStr = lipgloss.NewStyle().Foreground(lipgloss.Color("12")).Render(server.Name)
		}

		if m.cursor == i {
			cursorStr = lipgloss.NewStyle().Foreground(lipgloss.Color("12")).Render("> ")
		}

		cursorBox := cursorColStyle.Render(cursorStr)
		nameBox := nameColStyle.Render(nameStr)
		pingBox := pingColStyle.Render(m.pings[i])

		row := lipgloss.JoinHorizontal(lipgloss.Left, cursorBox, nameBox, pingBox)
		s += row + "\n"
	}

	if m.activeIdx != -1 {
		s += statusStyle.Foreground(lipgloss.Color("10")).Render(fmt.Sprintf(" STATUS: connecting to [%s]", m.servers[m.activeIdx].Name))
	} else {
		s += statusStyle.Foreground(lipgloss.Color("7")).Render(" STATUS: VPN off")
	}

	s += "\n\n " + lipgloss.NewStyle().Foreground(lipgloss.Color("8")).Render("↑/↓: Choose | Enter: On/Off | →: Ping | p: Ping all | q: Quit")

	v := tea.NewView(s)
	v.AltScreen = true
	return v
}
