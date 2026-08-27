package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"gohide/internal/bin"
	"gohide/internal/parser"
	"gohide/internal/tui"
	"gohide/internal/vpn"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/atotto/clipboard"
	"github.com/joho/godotenv"
)

// Мини-модель TUI для поля ввода ключа
type inputModel struct {
	url  string
	done bool
}

func (m inputModel) Init() tea.Cmd { return nil }

func (m inputModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	// Ловим системное событие вставки (если терминал его поддерживает)
	case tea.PasteMsg:
		m.url += msg.String()
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "enter":
			m.done = true
			return m, tea.Quit

		case "backspace":
			// Безопасное удаление символов
			runes := []rune(m.url)
			if len(runes) > 0 {
				m.url = string(runes[:len(runes)-1])
			}

		case "ctrl+c", "esc":
			os.Exit(0)

		case "ctrl+v":
			// Попытка прочитать из буфера ОС
			text, err := clipboard.ReadAll()
			if err == nil {
				m.url += strings.TrimSpace(text)
			}

		default:
			s := msg.String()
			if len([]rune(s)) == 1 {
				m.url += s
			} else if s == "space" {
				m.url += " "
			}
		}
	}
	return m, nil
}

func (m inputModel) View() tea.View {
	if m.done {
		return tea.NewView("") // Очищаем экран перед запуском основного TUI
	}
	title := lipgloss.NewStyle().Foreground(lipgloss.Color("12")).Bold(true).Render("🔑 Config url:")
	s := fmt.Sprintf("\n%s\n\n> %s\n\n(Ctrl+V for paste url | Enter for continue)", title, m.url)
	return tea.NewView(s)
}

func main() {
	_ = godotenv.Load()
	url := os.Getenv("URL")

	if url == "" {
		p := tea.NewProgram(inputModel{})
		finalModel, err := p.Run()
		if err != nil {
			log.Fatalf("Input error: %v", err)
		}

		url = finalModel.(inputModel).url

		// Сохраняем введенный ключ в файл .env
		envMap, err := godotenv.Read()
		if err != nil {
			envMap = make(map[string]string) // Если файла нет, создаем новую мапу
		}
		envMap["URL"] = url
		_ = godotenv.Write(envMap, ".env")
	}

	decode, err := parser.ParseDataFromUrl(url)
	if err != nil {
		log.Fatal(err)
	}

	var cfgArray []vpn.RawConfig
	for _, value := range parser.DecodeString(decode) {
		cfgStruct, err := vpn.ParseLine(value)
		if err != nil {
			continue
		} else {
			cfgArray = append(cfgArray, cfgStruct)
		}
	}

	// Подготавливаем временный бинарник sing-box
	binaryBytes := bin.SingBoxBinary
	tempDir := os.TempDir()

	// Определяем имя файла в зависимости от ОС
	execName := "sing-box"
	if runtime.GOOS == "windows" {
		execName += ".exe"
		binaryBytes = bin.SingBoxWidows
	}
	execPath := filepath.Join(tempDir, execName)

	// Права 0755 критически важны для Linux (дает право на исполнение)
	writeErr := os.WriteFile(execPath, binaryBytes, 0755)
	if writeErr != nil {
		fmt.Printf("[ERROR] Sing-box unpacking error: %v\n", writeErr)
		return
	}
	defer os.Remove(execPath)
	// Инициализируем и запускаем основной интерфейс
	initialModel := tui.NewModel(cfgArray, execPath)

	p := tea.NewProgram(initialModel)
	if _, err := p.Run(); err != nil {
		log.Fatalf("Ошибка запуска интерфейса: %v", err)
	}
}
