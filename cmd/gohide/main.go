package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"runtime"

	"gohide/internal/bin"
	"gohide/internal/parser"
	"gohide/internal/storage"
	"gohide/internal/tui"
	"gohide/internal/vpn"

	tea "charm.land/bubbletea/v2"
)

// Функция, которая собирает и парсит конфиги из всех сохраненных в JSON подписок
func loadAllConfigs() []vpn.RawConfig {
	var combinedConfigs []vpn.RawConfig

	subs, err := storage.LoadSubscriptions()
	if err != nil {
		fmt.Printf("[ERROR] Ошибка загрузки подписок: %v\n", err)
		return combinedConfigs
	}

	for _, sub := range subs {
		decode, err := parser.ParseDataFromUrl(sub.URL)
		if err != nil {
			// Если одна подписка недоступна, логируем и идем дальше
			fmt.Printf("[WARN] Не удалось загрузить подписку '%s': %v\n", sub.Name, err)
			continue
		}

		for _, value := range parser.DecodeString(decode) {
			cfgStruct, err := vpn.ParseLine(value)
			if err != nil {
				continue
			}
			// Для наглядности добавляем имя подписки перед именем самого сервера
			cfgStruct.Name = fmt.Sprintf("[%s] %s", sub.Name, cfgStruct.Name)
			combinedConfigs = append(combinedConfigs, cfgStruct)
		}
	}

	return combinedConfigs
}

func main() {
	// Сразу собираем доступные сервера из сохраненных подписок
	cfgArray := loadAllConfigs()

	// Подготавливаем временный бинарник sing-box
	binaryBytes := bin.SingBoxBinary
	tempDir := os.TempDir()

	execName := "sing-box"
	if runtime.GOOS == "windows" {
		execName += ".exe"
		binaryBytes = bin.SingBoxWidows
	}
	execPath := filepath.Join(tempDir, execName)

	writeErr := os.WriteFile(execPath, binaryBytes, 0755)
	if writeErr != nil {
		log.Fatalf("[ERROR] Sing-box unpacking error: %v\n", writeErr)
	}
	defer os.Remove(execPath)

	var p *tea.Program

	// Функция обновления (колбэк), вызываемая из TUI после успешного добавления подписки
	reloadCallback := func() {
		newConfigs := loadAllConfigs()
		// Изменяем состояние программы внутри запущенного процесса Bubbletea
		p.Send(func(model tea.Model) tea.Model {
			if m, ok := model.(tui.Model); ok {
				m.UpdateServers(newConfigs)
				return m
			}
			return model
		})
	}

	// Инициализируем основную модель с передачей колбэка
	initialModel := tui.NewModel(cfgArray, execPath, reloadCallback)

	p = tea.NewProgram(initialModel)
	if _, err := p.Run(); err != nil {
		log.Fatalf("Ошибка запуска интерфейса: %v", err)
	}
}
