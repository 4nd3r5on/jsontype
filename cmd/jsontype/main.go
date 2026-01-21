package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"log/slog"
	"os"
	"strings"

	"github.com/4nd3r5on/jsontype"
)

type multiFlag []string

func (m *multiFlag) String() string { return fmt.Sprint(*m) }
func (m *multiFlag) Set(v string) error {
	*m = append(*m, v)
	return nil
}

// parsePathList parses a comma-separated list of JSON paths
func parsePathList(s string) [][]string {
	if s == "" {
		return [][]string{}
	}

	paths := strings.Split(s, ",")
	result := make([][]string, 0, len(paths))

	for _, path := range paths {
		path = strings.TrimSpace(path)
		if path != "" {
			result = append(result, jsontype.StringToPath(path))
		}
	}

	return result
}

func main() {
	var files multiFlag
	var outPath string
	var logLevel string
	var parseObjectsStr string
	var ignoreObjectsStr string
	var maxDepth int

	flag.Var(&files, "file", "JSON file to parse (can be repeated)")
	flag.StringVar(&outPath, "out", "", "output file (default stdout)")
	flag.StringVar(&logLevel, "log-level", "info", "debug|info|warn|error")
	flag.StringVar(&parseObjectsStr, "parse-objects", "", "comma-separated JSON paths to parse (e.g., 'users,data.items')")
	flag.StringVar(&ignoreObjectsStr, "ignore-objects", "", "comma-separated JSON paths to ignore (e.g., 'metadata,debug.info')")
	flag.IntVar(&maxDepth, "max-depth", 0, "maximum depth to parse (0 = unlimited)")
	flag.Parse()

	level := slog.LevelInfo
	if err := level.UnmarshalText([]byte(logLevel)); err != nil {
		log.Fatalf("invalid log level: %s", logLevel)
	}
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: level,
	}))

	stat, _ := os.Stdin.Stat()
	hasStdin := stat.Mode()&os.ModeCharDevice == 0

	if !hasStdin && len(files) == 0 {
		flag.Usage()
		os.Exit(1)
	}

	out := os.Stdout
	if outPath != "" {
		f, err := os.Create(outPath)
		if err != nil {
			log.Fatalf("open output: %v", err)
		}
		defer f.Close()
		out = f
	}

	// Parse the path lists from command-line parameters
	parseObjects := parsePathList(parseObjectsStr)
	ignoreObjects := parsePathList(ignoreObjectsStr)

	slog.Debug("configuration",
		"parseObjects", parseObjects,
		"ignoreObjects", ignoreObjects,
		"maxDepth", maxDepth)

	merger := jsontype.NewMerger([]string{})

	process := func(r io.ReadCloser, label string) {
		defer r.Close()
		stream := jsontype.NewJSONStream(r)
		root, err := jsontype.ParseStream(stream, parseObjects, ignoreObjects, maxDepth, logger)
		if err != nil {
			log.Fatalf("parse %s: %v", label, err)
		}
		jsontype.MergeFieldInfo(merger, label, root, logger)
	}

	if hasStdin {
		slog.Debug("reading from stdin")
		process(io.NopCloser(os.Stdin), "stdin")
	}

	for _, path := range files {
		f, err := os.Open(path)
		if err != nil {
			log.Fatalf("open %s: %v", path, err)
		}
		slog.Debug("reading file", "file", path)
		process(f, path)
	}

	jsontype.PrintMergerTree(merger, "", out)
}
