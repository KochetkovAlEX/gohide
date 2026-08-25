package parser

import (
	"crypto/tls"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"
)

func ParseDataFromUrl(url string) (string, error) {
	transport := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	client := &http.Client{
		Transport: transport,
		Timeout:   30 * time.Second,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			// origin port 80
			if req.URL.Scheme == "http" {
				req.URL.Scheme = "https" // new port 443
			}

			// check redirects
			if len(via) >= 10 {
				return fmt.Errorf("[ERROR] To many redirect")
			}
			return nil
		},
	}
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("User-Agent", "curl/8.1.0")

	resp, err := client.Do(req)
	if err != nil {
		log.Fatal("Request error ", err)
	}

	defer resp.Body.Close()

	fmt.Println("[Status]", resp.StatusCode)

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	return string(body), nil
}
