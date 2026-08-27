package main

import (
	"fmt"
	"gohide/internal/bin"
	"gohide/internal/parser"
	"gohide/internal/tui"
	"gohide/internal/vpn"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	tea "charm.land/bubbletea/v2"
	_ "charm.land/lipgloss/v2"
	_ "github.com/charmbracelet/bubbles"
	"github.com/joho/godotenv"
)

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("[ERROR] Failed to load .env")
	}

	url := os.Getenv("URL")

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
	fmt.Printf("[INFO] Prepared %d configs\n", len(cfgArray))

	names, err := vpn.GetMapByCountryNames(cfgArray)
	if err != nil {
		log.Fatal(err)
	}
	// for key, _ := range names {
	// 	fmt.Println(key)
	// }

	configPath, err := vpn.BuildConfig(names["🇳🇱 ⚡️ ⭐️ LTE Авто - Нидерланды"])
	if err != nil {
		log.Fatal(err)
	}
	defer os.Remove(configPath)
	// fmt.Println(execPath)
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	binaryBytes := bin.SingBoxWidows
	tempDir := os.TempDir()
	execPath := filepath.Join(tempDir, "sing-box.exe")

	whriteErr := os.WriteFile(execPath, binaryBytes, 0755)
	if whriteErr != nil {
		fmt.Printf("[ERROR] Sing-box unpacking error: %v\n", whriteErr)
		return
	}
	defer os.Remove(execPath)

	initialModel := tui.NewModel(cfgArray, execPath)

	// Запускаем TUI. Используем AltScreen, чтобы интерфейс открывался на отдельном "экране",
	// и после выхода (q) терминал возвращался в исходное состояние.
	p := tea.NewProgram(initialModel)
	if _, err := p.Run(); err != nil {
		log.Fatalf("Ошибка запуска интерфейса: %v", err)
	}

	// // sing-box activationg.
	// cmd := exec.Command(execPath, "run", "-c", configPath)
	// // cmd.Stdout = os.Stdout
	// cmd.Stderr = os.Stderr

	// fmt.Printf("[INFO] Start sing-box with config: %s\n", configPath)
	// fmt.Println("[INFO] GoHide Start Working")
	// if err := cmd.Start(); err != nil {
	// 	fmt.Printf("[ERROR] Exec error: %v\n", err)
	// }

	// go func() {
	// 	_ = cmd.Wait()
	// }()

	// if cmd.Process != nil {
	// 	_ = cmd.Process.Kill()
	// 	_ = cmd.Wait()
	// }

}
