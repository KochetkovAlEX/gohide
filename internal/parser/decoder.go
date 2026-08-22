package parser

import (
	"encoding/base64"
	"log"
	"net/url"
	"strings"
)

func decodeFromBase64(decodeString string) string {
	rawContent := strings.TrimSpace(decodeString)
	decode, err := base64.StdEncoding.DecodeString(rawContent)
	if err != nil {
		log.Fatal(err)
	}

	return string(decode)
}

func decodeFromUTF8(base64String string) []string {
	encodedLines := strings.Split(base64String, "\n")

	for i, value := range encodedLines {
		value = strings.TrimSpace(value)
		decodedValue, err := url.PathUnescape(value)
		if err != nil {
			encodedLines[i] = value
		} else {
			encodedLines[i] = decodedValue
		}
	}
	return encodedLines
}

func DecodeString(decodeString string) []string {
	base64String := decodeFromBase64(decodeString)
	return decodeFromUTF8(base64String)
}
