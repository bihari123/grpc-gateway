//go:build mage
// +build mage

package main

import (
	"fmt"
	"os"
	"os/exec"
	"time"

	"github.com/fatih/color"
	"github.com/magefile/mage/mg"
	"github.com/magefile/mage/sh"
)

var (
	tools = []string{
		"github.com/bufbuild/buf/cmd/buf",
		"github.com/bufbuild/buf/cmd/protoc-gen-buf-breaking",
		"github.com/bufbuild/buf/cmd/protoc-gen-buf-lint",
		"github.com/golangci/golangci-lint/cmd/golangci-lint",
		"github.com/grpc-ecosystem/grpc-gateway/v2/protoc-gen-grpc-gateway",
		"github.com/grpc-ecosystem/grpc-gateway/v2/protoc-gen-openapiv2",
		"golang.org/x/tools/cmd/cover",
		"golang.org/x/tools/cmd/goimports",
		"google.golang.org/grpc/cmd/protoc-gen-go-grpc",
		"google.golang.org/protobuf/cmd/protoc-gen-go",
	}

	Default = Build
)

func Bench() error {
	fmt.Println("Running benchmarks...")

	if err := sh.RunV("go", "test", "-run", "XXX", "-bench", ".", "-benchmem", "-short", "./..."); err != nil {
		return err
	}

	fmt.Println("Done.")
	return nil
}

// sends the binary to the server
func Deploy() error {
	return nil
}

// Bootstrap installs tools required for development
func Bootstrap() error {
	fmt.Println("Bootstrapping tools...")
	if err := os.MkdirAll("_tools", 0755); err != nil {
		return fmt.Errorf("creating dir: %w", err)
	}

	// create module if go.mod doesnt exist
	if _, err := os.Stat("_tools/go.mod"); os.IsNotExist(err) {
		cmd := exec.Command("go", "mod", "init", "tools")
		cmd.Dir = "_tools"
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			return err
		}
	}

	install := []string{"install", "-v"}
	install = append(install, tools...)

	cmd := exec.Command("go", install...)
	cmd.Dir = "_tools"
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// Build builds the project similar to a release build
func Build() error {
	mg.Deps(Clean)
	fmt.Println("Building...")

	if err := build(buildModeProd); err != nil {
		return err
	}

	fmt.Println("Done.")
	fmt.Printf("\nRun the following to start Flipt:\n")
	fmt.Printf("\n%v\n", color.CyanString(`./bin/flipt --config config/local.yml`))
	return nil
}

// Dev builds the project for development, without bundling assets
func Dev() error {
	mg.Deps(Clean)
	fmt.Println("Building...")

	if err := build(buildModeDev); err != nil {
		return err
	}

	fmt.Println("Done.")
	fmt.Printf("\nRun the following to start Flipt server:\n")
	fmt.Printf("\n%v\n", color.CyanString(`./bin/flipt --config config/local.yml`))
	fmt.Printf("\nIn another shell, run the following to start the UI in dev mode:\n")
	fmt.Printf("\n%v\n", color.CyanString(`cd ui && npm run dev`))
	return nil
}

type buildMode uint8

const (
	// buildModeDev builds the project for development, without bundling assets
	buildModeDev buildMode = iota
	// BuildModeProd builds the project similar to a release build
	buildModeProd
)

// Proto generates protobuf files and gRPC stubs
func Proto() error {
	mg.Deps(Bootstrap)
	fmt.Println("Generating proto files...")
	return sh.RunV("buf", "generate")
}

func build(mode buildMode) error {
	buildDate := time.Now().UTC().Format(time.RFC3339)
	buildArgs := make([]string, 0)

	switch mode {
	case buildModeProd:
		buildArgs = append(buildArgs, "-tags", "assets")
	}

	gitCommit, err := sh.Output("git", "rev-parse", "HEAD")
	if err != nil {
		return fmt.Errorf("getting git commit: %w", err)
	}

	buildArgs = append(
		[]string{
			"build",
			"-trimpath",
			"-ldflags",
			fmt.Sprintf("-X main.commit=%s -X main.date=%s", gitCommit, buildDate),
		},
		buildArgs...)
	buildArgs = append(buildArgs, "-o", "./bin/flipt", "./cmd/flipt/")

	return sh.RunV("go", buildArgs...)
}

// Clean cleans up built files
func Clean() error {
	fmt.Println("Cleaning...")

	if err := sh.RunV("go", "mod", "tidy"); err != nil {
		return fmt.Errorf("tidying go.mod: %w", err)
	}

	clean := []string{"dist/*", "pkg/*", "bin/*"}
	for _, dir := range clean {
		if err := os.RemoveAll(dir); err != nil {
			return fmt.Errorf("removing dir %q: %w", dir, err)
		}
	}

	return nil
}

// Prep prepares the project for building
func Prep() error {
	fmt.Println("Preparing...")
	mg.Deps(Clean)
	return nil
}
