package main

import (
	"fmt"
	"gohide/internal/parser"
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

	// show all urls in config
	for _, value := range parser.DecodeString(decode) {
		fmt.Println(value)
	}
}
