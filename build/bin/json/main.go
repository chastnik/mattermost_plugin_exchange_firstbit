package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
)

type Manifest struct {
	ID               string    `json:"id"`
	Version          string    `json:"version"`
	MinServerVersion string    `json:"min_server_version"`
	Server           *struct{} `json:"server"`
	Webapp           *struct{} `json:"webapp"`
}

func main() {
	if len(os.Args) != 2 {
		fmt.Fprintf(os.Stderr, "Usage: %s <field>\n", os.Args[0])
		os.Exit(1)
	}

	field := os.Args[1]

	input, err := io.ReadAll(os.Stdin)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading input: %v\n", err)
		os.Exit(1)
	}

	var manifest Manifest
	if err := json.Unmarshal(input, &manifest); err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing JSON: %v\n", err)
		os.Exit(1)
	}

	switch field {
	case "id":
		fmt.Print(manifest.ID)
	case "version":
		fmt.Print(manifest.Version)
	case "min_server_version":
		fmt.Print(manifest.MinServerVersion)
	case "has_server":
		if manifest.Server != nil {
			fmt.Print("1")
		} else {
			fmt.Print("0")
		}
	case "has_webapp":
		if manifest.Webapp != nil {
			fmt.Print("1")
		} else {
			fmt.Print("0")
		}
	case "executable":
		fmt.Print("plugin-linux-amd64")
	default:
		fmt.Fprintf(os.Stderr, "Unknown field: %s\n", field)
		os.Exit(1)
	}
}
