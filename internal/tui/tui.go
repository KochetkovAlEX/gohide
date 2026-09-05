package tui

import (
	"crypto/tls"
	"fmt"
	"net"
	"os"
	"os/exec"
	"time"

	"gohide/internal/storage"
	"gohide/internal/vpn"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/atotto/clipboard"
)

type pingMsg struct {
	index int
	rtt   time.Duration
	err   error
}

type uiState int

const (
	stateList uiState = iota // Экран списка серверов
	stateAdd                 // Экран добавления подписки
)

type Model struct {
	servers       []vpn.RawConfig
	pings         []string
	cursor        int
	offset        int
	height        int
	activeIdx     int
	selectedMode  string
	activeMode    string
	activeCmd     *exec.Cmd
	activeCfgPath string
	execPath      string

	// Поля для работы с подписками
	state        uiState
	inputName    string
	inputURL     string
	activeInput  int // 0 - Name, 1 - URL
	inputErr     string
	onReloadSubs func() // Колбэк для перезагрузки данных в main
}

func NewModel(configs []vpn.RawConfig, execPath string, onReload func()) Model {
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
		selectedMode: "proxy",
		activeMode:   "",
		execPath:     execPath,
		state:        stateList,
		activeInput:  0,
		onReloadSubs: onReload,
	}
}

func (m Model) Init() tea.Cmd {
	return nil
}

func (m Model) pingAll() tea.Cmd {
	var cmds []tea.Cmd
	for i, s := range m.servers {
		m.pings[i] = "Ping..."
		sni := s.SNI
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
		dialer := &net.Dialer{Timeout: 3 * time.Second}
		tlsConfig := &tls.Config{ServerName: sni, InsecureSkipVerify: true}
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

		// Перед case tea.KeyMsg: вставьте этот блок:
	case tea.PasteMsg:
		if m.state == stateAdd {
			if m.activeInput == 0 {
				m.inputName += msg.String()
			} else {
				m.inputURL += msg.String()
			}
		}
		return m, nil

	case tea.KeyMsg:
		// ----------------------------------------------------
		// ЛОГИКА ЭКРАНА ДОБАВЛЕНИЯ ПОДПИСКИ
		// ----------------------------------------------------
		if m.state == stateAdd {
			switch msg.String() {
			case "esc":
				m.inputName = ""
				m.inputURL = ""
				m.inputErr = ""
				m.state = stateList
				return m, nil

			case "tab", "up", "down":
				if m.activeInput == 0 {
					m.activeInput = 1
				} else {
					m.activeInput = 0
				}
				return m, nil

			case "ctrl+v":
				text, err := clipboard.ReadAll()
				if err == nil {
					if m.activeInput == 0 {
						m.inputName += text
					} else {
						m.inputURL += text
					}
				}
				return m, nil

			case "enter":
				if m.inputName == "" || m.inputURL == "" {
					m.inputErr = "Поля не могут быть пустыми!"
					return m, nil
				}
				err := storage.SaveSubscription(m.inputName, m.inputURL)
				if err != nil {
					m.inputErr = err.Error()
					return m, nil
				}

				// Сохраняем перед очисткой состояния
				m.inputName = ""
				m.inputURL = ""
				m.inputErr = ""
				m.state = stateList

				// Вызываем перезагрузку серверов в main.go
				if m.onReloadSubs != nil {
					m.onReloadSubs()
				}

				return m, tea.ClearScreen

			case "backspace":
				if m.activeInput == 0 {
					runes := []rune(m.inputName)
					if len(runes) > 0 {
						m.inputName = string(runes[:len(runes)-1])
					}
				} else {
					runes := []rune(m.inputURL)
					if len(runes) > 0 {
						m.inputURL = string(runes[:len(runes)-1])
					}
				}
				return m, nil

			default:
				s := msg.String()
				if len(s) > 1 && s != "space" {
					return m, nil
				}

				if s == "space" {
					s = " "
				}

				if len([]rune(s)) == 1 {
					if m.activeInput == 0 {
						m.inputName += s
					} else {
						m.inputURL += s
					}
				}
				return m, nil

			}
		}

		// ----------------------------------------------------
		// ЛОГИКА ОСНОВНОГО ЭКРАНА СЛУЖБЫ
		// ----------------------------------------------------
		visibleLines := m.height - 8
		if visibleLines < 1 {
			visibleLines = 1
		}

		switch msg.String() {
		case "ctrl+c", "q":
			m.stopSingBox()
			return m, tea.Quit

		case "a":
			m.state = stateAdd
			m.activeInput = 0
			m.inputErr = ""
			return m, nil

		case "up", "k":
			if len(m.servers) == 0 {
				return m, nil
			}
			if m.cursor > 0 {
				m.cursor--
				if m.cursor < m.offset {
					m.offset = m.cursor
				}
			}

		case "down", "j":
			if len(m.servers) == 0 {
				return m, nil
			}
			if m.cursor < len(m.servers)-1 {
				m.cursor++
				if m.cursor >= m.offset+visibleLines {
					m.offset = m.cursor - visibleLines + 1
				}
			}

		case "p":
			if len(m.servers) > 0 {
				return m, m.pingAll()
			}

		case "right", "l":
			if len(m.servers) > 0 {
				m.pings[m.cursor] = "Ping..."
				return m, m.pingServerCmd(m.cursor, m.servers[m.cursor].Address, m.servers[m.cursor].Port, m.servers[m.cursor].SNI)
			}

		case "t":
			if m.selectedMode == "proxy" {
				m.selectedMode = "tun"
			} else {
				m.selectedMode = "proxy"
			}
			return m, nil

		case "enter", "space":
			if len(m.servers) == 0 {
				return m, nil
			}
			if m.activeIdx == m.cursor {
				m.stopSingBox()
			} else {
				m.stopSingBox()

				chosen := m.servers[m.cursor]
				var configPath string
				var err error

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

func (m *Model) UpdateServers(newConfigs []vpn.RawConfig) {
	m.servers = newConfigs
	pings := make([]string, len(newConfigs))
	for i := range pings {
		pings[i] = "-"
	}
	m.pings = pings
	m.cursor = 0
	m.offset = 0
	m.activeIdx = -1
}

func (m Model) View() tea.View {
	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("12")).
		Border(lipgloss.NormalBorder(), false, false, true, false).
		Padding(0, 1, 0, 1).
		MarginBottom(1)

	// ----------------------------------------------------
	// РЕНДЕРИНГ ЭКРАНА ДОБАВЛЕНИЯ ПОДПИСКИ (ОБНОВЛЕННЫЙ)
	// ----------------------------------------------------
	if m.state == stateAdd {
		s := titleStyle.Render("➕ Add New Subscription") + "\n\n"

		nameLabel := "  Название подписки:"
		urlLabel := "  Ссылка (URL):"

		displayFieldsStyle := lipgloss.NewStyle().PaddingLeft(2).Foreground(lipgloss.Color("7"))

		var nameContent, urlContent string

		if m.activeInput == 0 {
			nameLabel = lipgloss.NewStyle().Foreground(lipgloss.Color("12")).Bold(true).Render("> Название подписки:")
			nameContent = lipgloss.NewStyle().Foreground(lipgloss.Color("15")).Underline(true).Render(m.inputName)
			urlContent = m.inputURL
		} else {
			urlLabel = lipgloss.NewStyle().Foreground(lipgloss.Color("12")).Bold(true).Render("> Ссылка (URL):")
			nameContent = m.inputName
			urlContent = lipgloss.NewStyle().Foreground(lipgloss.Color("15")).Underline(true).Render(m.inputURL)
		}

		if nameContent == "" {
			nameContent = lipgloss.NewStyle().Foreground(lipgloss.Color("8")).Render("Введите имя...")
		}
		if urlContent == "" {
			urlContent = lipgloss.NewStyle().Foreground(lipgloss.Color("8")).Render("Вставьте или введите ссылку...")
		}

		s += fmt.Sprintf("%s\n%s\n\n", nameLabel, displayFieldsStyle.Render(nameContent))
		s += fmt.Sprintf("%s\n%s\n\n", urlLabel, displayFieldsStyle.Render(urlContent))

		if m.inputErr != "" {
			s += lipgloss.NewStyle().Foreground(lipgloss.Color("9")).Render(fmt.Sprintf("Ошибка: %s", m.inputErr)) + "\n\n"
		}

		s += lipgloss.NewStyle().Foreground(lipgloss.Color("8")).Render(" Tab/Arrows: Switch | Ctrl+V: Paste | Enter: Save | Esc: Cancel")
		v := tea.NewView(s)
		v.AltScreen = true
		return v
	}

	// ----------------------------------------------------
	// РЕНДЕРИНГ ОСНОВНОГО СПИСКА СЕРВЕРОВ
	// ----------------------------------------------------
	statusStyle := lipgloss.NewStyle().Bold(true).Padding(0, 1).MarginTop(1)

	modeDisplay := "PROXY"
	if m.selectedMode == "tun" {
		modeDisplay = "TUN"
	}
	s := titleStyle.Render(fmt.Sprintf("🛡️  GoHide VPN Client | Mode: %s", modeDisplay)) + "\n\n"

	if len(m.servers) == 0 {
		s += lipgloss.NewStyle().Foreground(lipgloss.Color("11")).Render(" Список серверов пуст. Нажмите 'a', чтобы добавить подписку.") + "\n"
	} else {
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
	}
	if m.activeIdx != -1 {
		activeModeStr := "PROXY"
		if m.activeMode == "tun" {
			activeModeStr = "TUN"
		}
		s += statusStyle.Foreground(lipgloss.Color("10")).Render(fmt.Sprintf(" STATUS: connect to [%s] (Mode: %s)", m.servers[m.activeIdx].Name, activeModeStr))
	} else {
		s += statusStyle.Foreground(lipgloss.Color("7")).Render(" STATUS: VPN off")
	}
	s += "\n\n " + lipgloss.NewStyle().Foreground(lipgloss.Color("8")).Render("↑/↓: Choose | Enter: On/Off | t: Mode | a: Add Subs | →: Ping | p: Ping All | q: Quit")
	v := tea.NewView(s)
	v.AltScreen = true
	return v
}
