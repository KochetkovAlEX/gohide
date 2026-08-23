package main

import (
	"fmt"
	"gohide/internal/parser"
	"gohide/internal/vpn"
	"log"
	"os"

	_ "charm.land/bubbletea/v2"
	_ "charm.land/lipgloss/v2"
	_ "github.com/charmbracelet/bubbles"
	"github.com/joho/godotenv"
)

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Failed to load .env")
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
	fmt.Printf("Prepared %d configs", len(cfgArray))

	names, err := vpn.GetMapByCountryNames(cfgArray)
	if err != nil {
		log.Fatal(err)
	}
	// for key, _ := range names {
	// 	fmt.Println(key)
	// }
	vpn.BuildConfig(names["🇩🇰 ⭐️ Дания"])
}
