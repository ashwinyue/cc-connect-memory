package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	"github.com/chenhg5/cc-connect/core/memory"
)

// runMemory handles the memory subcommand.
func runMemory(args []string) {
	fs := flag.NewFlagSet("memory", flag.ExitOnError)
	project := fs.String("project", "", "Project name (required)")
	dataDir := fs.String("data-dir", "workspace", "Data directory")
	help := fs.Bool("help", false, "Show help")

	if err := fs.Parse(args); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	if *help || *project == "" {
		printMemoryUsage()
		os.Exit(1)
	}

	// Get remaining args as subcommand
	subcommand := ""
	if fs.NArg() > 0 {
		subcommand = fs.Arg(0)
	}

	switch subcommand {
	case "compact":
		runMemoryCompact(*dataDir, *project)
	case "status":
		runMemoryStatus(*dataDir, *project)
	default:
		// Default: show status
		runMemoryStatus(*dataDir, *project)
	}
}

func printMemoryUsage() {
	fmt.Println(`Usage: cc-connect memory [options] [command]

Manage memory for projects.

Commands:
  compact   Compress/merge similar memories
  status    Show memory statistics

Options:
  --project string   Project name (required)
  --data-dir dir    Data directory (default: workspace)

Examples:
  cc-connect memory --project moon compact
  cc-connect memory --project moon status
  cc-connect memory --project moon --data-dir ~/.cc-connect status`)
}

// runMemoryCompact compresses memories for a project.
func runMemoryCompact(dataDir, project string) {
	memDir := filepath.Join(dataDir, "bot", project)

	// Check if memory directory exists
	if _, err := os.Stat(memDir); os.IsNotExist(err) {
		fmt.Fprintf(os.Stderr, "Error: memory directory not found for project %q\n", project)
		fmt.Fprintf(os.Stderr, "Expected: %s\n", memDir)
		os.Exit(1)
	}

	// Create store
	store := memory.NewStoreFS(memDir, nil, slog.Default())
	store.SetCompactor(memory.NewSimpleCompactor())

	// Run compaction
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	result, err := store.Compact(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: compaction failed: %v\n", err)
		os.Exit(1)
	}

	// Print results
	fmt.Printf("Memory compaction completed!\n")
	fmt.Printf("  Before: %d memories\n", result.BeforeCount)
	fmt.Printf("  After:  %d memories\n", result.AfterCount)
	fmt.Printf("  Removed: %d\n", len(result.MergedIDs))
}

// runMemoryStatus shows memory statistics for a project.
func runMemoryStatus(dataDir, project string) {
	memDir := filepath.Join(dataDir, "bot", project)

	// Check if memory directory exists
	if _, err := os.Stat(memDir); os.IsNotExist(err) {
		fmt.Fprintf(os.Stderr, "Error: memory directory not found for project %q\n", project)
		fmt.Fprintf(os.Stderr, "Expected: %s\n", memDir)
		os.Exit(1)
	}

	// Create store
	store := memory.NewStoreFS(memDir, nil, slog.Default())

	// Get all memories
	memories, err := store.GetAll(context.Background(), nil)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: failed to list memories: %v\n", err)
		os.Exit(1)
	}

	// Print statistics
	fmt.Printf("Memory Status for project: %s\n", project)
	fmt.Printf("  Total memories: %d\n", len(memories))

	// Count by date
	dateCount := make(map[string]int)
	for _, mem := range memories {
		date := mem.CreatedAt[:10] // YYYY-MM-DD
		dateCount[date]++
	}

	fmt.Printf("  Memories by date:\n")
	for date, count := range dateCount {
		fmt.Printf("    %s: %d\n", date, count)
	}
}
