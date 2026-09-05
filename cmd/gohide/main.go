package main

import (
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

// FetchConfigsForSub requests and parses endpoints from a specific subscription URL
func FetchConfigsForSub(url string) ([]vpn.RawConfig, error) {
	var results []vpn.RawConfig
	decode, err := parser.ParseDataFromUrl(url)
	if err != nil {
		return nil, err
	}

	for _, value := range parser.DecodeString(decode) {
		cfgStruct, err := vpn.ParseLine(value)
		if err != nil {
			continue
		}
		results = append(results, cfgStruct)
	}
	return results, nil
}

func main() {
	// Prepare temporary sing-box binary based on current OS
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

	// Callback to trigger UI re-render when subscription database changes
	// reloadSubsCallback := func() {
	// 	subs, _ := storage.LoadSubscriptions()
	// 	p.Send(func(model tea.Model) tea.Model {
	// 		if m, ok := model.(tui.Model); ok {
	// 			m.UpdateSubscriptions(subs)
	// 			return m
	// 		}
	// 		return model
	// 	})
	// }

	// Dynamic on-demand configuration loader callback
	loadConfigsCallback := func(url string) ([]vpn.RawConfig, error) {
		return FetchConfigsForSub(url)
	}

	// Initial fetch of local subscriptions from JSON
	subs, err := storage.LoadSubscriptions()
	if err != nil {
		subs = []storage.Subscription{}
	}

	// Initialize UI model with updated arguments matching state machine needs
	initialModel := tui.NewModel(subs, execPath, nil, loadConfigsCallback)
	p = tea.NewProgram(initialModel)
	if _, err := p.Run(); err != nil {
		log.Fatalf("UI execution error: %v", err)
	}
}
