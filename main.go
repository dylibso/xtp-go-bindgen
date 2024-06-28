package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/extism/go-sdk"
)

// TODO this is wonk
// CopyFile copies a file from src to dst. If the dst file already exists, it will be overwritten.
func CopyFile(src, dst string) error {
	// Open the source file for reading
	sourceFile, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("could not open source file: %v", err)
	}
	defer sourceFile.Close()

	// Create the destination file
	destFile, err := os.Create(dst)
	if err != nil {
		return fmt.Errorf("could not create destination file: %v", err)
	}
	defer destFile.Close()

	// Copy the contents from source to destination
	_, err = io.Copy(destFile, sourceFile)
	if err != nil {
		return fmt.Errorf("error copying contents: %v", err)
	}

	// Sync to ensure all contents are written to disk
	err = destFile.Sync()
	if err != nil {
		return fmt.Errorf("error syncing destination file: %v", err)
	}

	return nil
}

func main() {
	schemaCtx := `
  {"project": {"name": "hello", "description": "a new plugin that does something"},
  "exports": [
    {"name": "export1", "input": { "type": "string"}},
    {"name": "export2", "input": { "type": "buffer"} },
    {"name": "export3", "input": { "contentType": "application/json"}, "output": {"type": "string"} },
    {"name": "export4", "input": { "contentType": "application/json"}, "output": {"contentType": "application/json"} }
  ]
  }
  `

	manifest := extism.Manifest{
		Wasm: []extism.Wasm{
			extism.WasmFile{
				Path: "./dist/plugin.wasm",
			},
		},
		Config: map[string]string{
			"ctx": schemaCtx,
		},
	}

	ctx := context.Background()
	config := extism.PluginConfig{EnableWasi: true}
	plugin, err := extism.NewPlugin(ctx, manifest, config, []extism.HostFunction{})

	if err != nil {
		fmt.Printf("Failed to initialize plugin: %v\n", err)
		os.Exit(1)
	}

	root := "template"
	output := "output"

	// TODO this is kind of dangerous
	os.RemoveAll(output)
	os.Mkdir(output, os.ModePerm)

	err = filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		// skip root path
		if path == root {
			return nil
		}

		mirrorPath := strings.Replace(path, root, output, 1)

		name := info.Name()
		if info.IsDir() {
			fmt.Println("+ Creating directory " + mirrorPath)
			os.Mkdir(mirrorPath, os.ModePerm)
		} else {
			if strings.HasSuffix(strings.ToLower(name), ".ejs") {
				sourceFile, err := os.Open(path)
				if err != nil {
					return fmt.Errorf("could not open source file: %v", err)
				}
				defer sourceFile.Close()

				data, err := io.ReadAll(sourceFile)
				if err != nil {
					panic(err)
				}

				exit, result, err := plugin.Call("render", data)
				if err != nil {
					fmt.Println(err)
					os.Exit(int(exit))
				}

				// TODO this won't work with casing and could replace the wrong thing first
				mirrorPath = strings.Replace(mirrorPath, ".ejs", "", 1)

				err = os.WriteFile(mirrorPath, result, os.ModePerm)
				if err != nil {
					panic(err)
				}
				fmt.Println("! Rendered template to " + mirrorPath)
			} else {
				fmt.Println("- Copy file to " + mirrorPath)
				CopyFile(path, mirrorPath)
			}
		}

		return nil
	})

	if err != nil {
		log.Fatalf("Error walking the path %q: %v\n", root, err)
	}
}
