package main

import (
	"flag"
	"io"
	"log"
	"log/slog"
	"os"
	"strings"

	sp "github.com/4nd3r5on/go-strings-parser"

	"github.com/4nd3r5on/jsontype"
)

// parsePathList parses a comma-separated list of JSON paths
func parsePathList(s string) ([][]string, error) {
	if s == "" {
		return [][]string{}, nil
	}
	paths, err := sp.Parse(s,
		sp.WithProcessFunc(
			func(element string) (processed string, skip bool, err error) {
				return strings.TrimSpace(element), false, nil
			},
		),
	)
	if err != nil {
		return nil, err
	}
	result := make([][]string, 0, len(paths))

	for _, path := range paths {
		path = strings.TrimSpace(path)
		if path != "" {
			result = append(result, jsontype.StringToPath(path))
		}
	}

	return result, nil
}

func main() {
	var filesStr string
	var outPath string
	var logLevel string
	var parseObjectsStr string
	var ignoreObjectsStr string
	var noStringAnalysis bool
	var maxDepth int

	flag.StringVar(&filesStr, "file", "", "space-separated JSON files to parse (eg './parseme1.json ./parseme2.json')")
	flag.StringVar(&outPath, "out", "", "output file (default stdout)")
	flag.StringVar(&logLevel, "log-level", "info", "debug|info|warn|error")
	flag.BoolVar(&noStringAnalysis, "no-string-analysis", false, "will try to additionally detect types like string-uuid, string-email, etc within strings")
	flag.StringVar(&parseObjectsStr, "parse-objects", "", "space-separated JSON paths to parse (e.g., 'users data.items')")
	flag.StringVar(&ignoreObjectsStr, "ignore-objects", "", "space-separated JSON paths to ignore (e.g., 'metadata debug.info')")
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

	files, err := sp.Parse(filesStr,
		sp.WithProcessFunc(
			func(element string) (processed string, skip bool, err error) {
				return strings.TrimSpace(element), false, nil
			},
		),
	)
	if err != nil {
		log.Fatalf("failed to parse files argument: %v", err)
	}
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
	parseObjects, err := parsePathList(parseObjectsStr)
	if err != nil {
		log.Panicf("Failed to parse objects list: %v", err)
	}
	ignoreObjects, err := parsePathList(ignoreObjectsStr)
	if err != nil {
		log.Panicf("Failed to parse ignore list: %v", err)
	}

	slog.Debug("configuration",
		"parseObjects", parseObjects,
		"ignoreObjects", ignoreObjects,
		"maxDepth", maxDepth)

	merger := jsontype.NewMerger([]string{})

	process := func(r io.ReadCloser, label string) {
		defer r.Close()
		stream := jsontype.NewJSONStream(r)
		root, err := jsontype.ParseStream(stream, parseObjects, ignoreObjects, maxDepth, noStringAnalysis, logger)
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
