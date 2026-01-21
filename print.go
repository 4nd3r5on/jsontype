package jsontype

import (
	"fmt"
	"io"
	"slices"
	"sort"
	"strings"
)

func collectTypes(m map[DetectedType]struct{}) []DetectedType {
	out := make([]DetectedType, 0, len(m))
	for t := range m {
		out = append(out, t)
	}
	slices.Sort(out)
	return out
}

func collectLabels(m map[string]map[DetectedType]struct{}) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

func collectChildTypes(m *Merger) []DetectedType {
	seen := make(map[DetectedType]struct{})
	for _, ch := range m.ChildrenMap {
		for t := range ch.TypesMap {
			seen[t] = struct{}{}
		}
	}
	return collectTypes(seen)
}

func PrintMergerTree(m *Merger, prefix string, w io.Writer) {
	if m == nil {
		return
	}

	printNode(m, prefix, w)

	for _, k := range m.ChildrenKeys {
		child := m.ChildrenMap[k]
		PrintMergerTree(child, prefix+"  ", w)
	}
}

func printNode(m *Merger, prefix string, w io.Writer) {
	if len(m.TypesMap) == 0 {
		return
	}

	path := PathToString(m.Path)
	types := collectTypes(m.TypesMap)
	labels := collectLabels(m.LabeledTypesMap)

	// split container vs primitive
	var containers []DetectedType
	var primitives []DetectedType

	for _, t := range types {
		if IsContainerType(t) {
			containers = append(containers, t)
		} else {
			primitives = append(primitives, t)
		}
	}

	// ----- CONTAINERS -----
	if len(containers) > 0 {
		var rendered []string

		for _, ct := range containers {
			inner := "unknown"

			if len(m.ChildrenMap) > 0 {
				childTypes := collectChildTypes(m)
				if len(childTypes) > 0 {
					inner = strings.Join(TypesToString(childTypes), " | ")
				}
			}

			rendered = append(
				rendered,
				fmt.Sprintf("%s<%s>", ct, inner),
			)
		}

		fmt.Fprintf(
			w,
			"%s%s => %s\n",
			prefix,
			path,
			strings.Join(rendered, " | "),
		)
		return
	}

	// ----- PRIMITIVES -----
	if len(labels) > 1 && len(primitives) > 1 {
		for _, lbl := range labels {
			lblTypes := collectTypes(m.LabeledTypesMap[lbl])
			fmt.Fprintf(
				w,
				"%s%s @ %s => %s\n",
				prefix,
				path,
				lbl,
				strings.Join(TypesToString(lblTypes), " | "),
			)
		}
		return
	}

	fmt.Fprintf(
		w,
		"%s%s => %s\n",
		prefix,
		path,
		strings.Join(TypesToString(primitives), " | "),
	)
}
