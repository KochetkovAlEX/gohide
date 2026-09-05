package storage

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

type Subscription struct {
	Name string `json:"name"`
	URL  string `json:"url"`
}

type Storage struct {
	Subscriptions []Subscription `json:"subscriptions"`
}

func getStoragePath() string {
	baseDir, err := os.Getwd()
	if err != nil {
		baseDir = "."
	}
	return filepath.Join(baseDir, "subs.json")
}

func LoadSubscriptions() ([]Subscription, error) {
	path := getStoragePath()
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return []Subscription{}, nil
	}

	bytes, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read subscriptions: %v", err)
	}

	var store Storage
	if err := json.Unmarshal(bytes, &store); err != nil {
		return nil, fmt.Errorf("failed to parse JSON: %v", err)
	}

	return store.Subscriptions, nil
}

func SaveSubscription(name, url string) error {
	subs, err := LoadSubscriptions()
	if err != nil {
		return err
	}

	for _, s := range subs {
		if s.Name == name {
			return fmt.Errorf("subscription with name '%s' already exists", name)
		}
	}

	subs = append(subs, Subscription{Name: name, URL: url})
	store := Storage{Subscriptions: subs}

	bytes, err := json.MarshalIndent(store, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to serialize JSON: %v", err)
	}

	return os.WriteFile(getStoragePath(), bytes, 0644)
}

// DeleteSubscription filters out a subscription by name and rewrites JSON
func DeleteSubscription(name string) error {
	subs, err := LoadSubscriptions()
	if err != nil {
		return err
	}

	var updatedSubs []Subscription
	found := false
	for _, s := range subs {
		if s.Name == name {
			found = true
			continue
		}
		updatedSubs = append(updatedSubs, s)
	}

	if !found {
		return fmt.Errorf("subscription '%s' not found", name)
	}

	store := Storage{Subscriptions: updatedSubs}
	bytes, err := json.MarshalIndent(store, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to serialize JSON: %v", err)
	}

	return os.WriteFile(getStoragePath(), bytes, 0644)
}
