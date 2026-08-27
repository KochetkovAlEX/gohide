package tui

import (
	"crypto/tls"
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
	offset        int
	height        int
	activeIdx     int
	selectedMode  string // Выбранный режим для следующего запуска ("proxy" или "tun")
	activeMode    string // В каком режиме реально запущен текущий сервер
	activeCmd     *exec.Cmd
	activeCfgPath string
	execPath      string
}

func NewModel(configs []vpn.RawConfig, execPath string) Model {
	pings := make([]string, len(configs))
	for i := range pings {
		pings[i] = "-"
	}
	return Model{
		servers:      configs,
		pings:        pings,
		cursor:       0,
		offset:       0,
		activeIdx:    -1,
		selectedMode: "proxy", // По умолчанию всегда стартуем с proxy
		activeMode:   "",
		execPath:     execPath,
	}
}

func (m Model) Init() tea.Cmd {
	return nil
}

func (m Model) pingAll() tea.Cmd {
	var cmds []tea.Cmd
	for i, s := range m.servers {
		m.pings[i] = "Ping..."

		sni := s.SNI // Замени на свое поле, если оно называется иначе (например, s.Host)
		if sni == "" {
			sni = s.Address
		}

		cmds = append(cmds, m.pingServerCmd(i, s.Address, s.Port, sni))
	}
	return tea.Batch(cmds...)
}

func (m Model) pingServerCmd(index int, host, port, sni string) tea.Cmd {
	return func() tea.Msg {
		start := time.Now()

		dialer := &net.Dialer{
			Timeout: 3 * time.Second,
		}

		tlsConfig := &tls.Config{
			ServerName:         sni,
			InsecureSkipVerify: true,
		}

		conn, err := tls.DialWithDialer(dialer, "tcp", net.JoinHostPort(host, port), tlsConfig)
		if err != nil {
			return pingMsg{index: index, err: err}
		}

		_ = conn.Close()
		return pingMsg{index: index, rtt: time.Since(start)}
	}
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

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
				if m.cursor < m.offset {
					m.offset = m.cursor
				}
			}

		case "down", "j":
			if m.cursor < len(m.servers)-1 {
				m.cursor++
				if m.cursor >= m.offset+visibleLines {
					m.offset = m.cursor - visibleLines + 1
				}
			}

		case "p":
			return m, m.pingAll()

		case "right", "l":
			m.pings[m.cursor] = "Ping..."
			return m, m.pingServerCmd(m.cursor, m.servers[m.cursor].Address, m.servers[m.cursor].Port, m.servers[m.cursor].SNI)

		case "t":
			// Глобальный переключатель режима
			if m.selectedMode == "proxy" {
				m.selectedMode = "tun"
			} else {
				m.selectedMode = "proxy"
			}
			return m, nil

		case "enter", "space":
			if m.activeIdx == m.cursor {
				// Выключаем, если нажали на уже активный сервер
				m.stopSingBox()
			} else {
				// Тушим старый процесс перед запуском нового
				m.stopSingBox()

				chosen := m.servers[m.cursor]
				var configPath string
				var err error

				// Собираем конфиг на основе глобального флага
				if m.selectedMode == "tun" {
					configPath, err = vpn.BuildTunConfig(chosen)
				} else {
					configPath, err = vpn.BuildConfig(chosen)
				}

				if err != nil {
					return m, nil
				}

				cmd := exec.Command(m.execPath, "run", "-c", configPath)
				if err := cmd.Start(); err == nil {
					m.activeIdx = m.cursor
					m.activeMode = m.selectedMode
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
	m.activeMode = ""
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

	// Показываем текущий выбранный режим запуска прямо в заголовке
	modeDisplay := "PROXY"
	if m.selectedMode == "tun" {
		modeDisplay = "TUN"
	}
	s := titleStyle.Render(fmt.Sprintf("🛡️  GoHide VPN Client | Mode: %s", modeDisplay)) + "\n\n"

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

	// Статус отображает режим, в котором РЕАЛЬНО работает текущее подключение
	if m.activeIdx != -1 {
		activeModeStr := "PROXY"
		if m.activeMode == "tun" {
			activeModeStr = "TUN"
		}
		s += statusStyle.Foreground(lipgloss.Color("10")).Render(fmt.Sprintf(" STATUS: connect to [%s] (Mode: %s)", m.servers[m.activeIdx].Name, activeModeStr))
	} else {
		s += statusStyle.Foreground(lipgloss.Color("7")).Render(" STATUS: VPN off")
	}

	s += "\n\n " + lipgloss.NewStyle().Foreground(lipgloss.Color("8")).Render("↑/↓: Choose | Enter: On/Off | t: Change mode | →: Ping | p: Ping All | q: Quite")

	v := tea.NewView(s)
	v.AltScreen = true
	return v
}
