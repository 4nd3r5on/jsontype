package jsontype

import (
	"encoding/json"
	"fmt"
	"log/slog"
)

type parser struct {
	Root      *FieldInfo
	seenPaths map[string]*FieldInfo
	logger    *slog.Logger
}

func ParseStream(
	s Stream,
	parseObjects, ignoreObjects [][]string,
	maxDepth int,
	logger *slog.Logger,
) (root *FieldInfo, err error) {
	if logger == nil {
		logger = slog.Default()
	}

	p := parser{
		seenPaths: make(map[string]*FieldInfo),
		logger:    logger,
	}

	p.logger.Info("starting JSON stream parsing",
		"maxDepth", maxDepth,
		"parseObjectsCount", len(parseObjects),
		"ignoreObjectsCount", len(ignoreObjects))

	// depth = 0, current path = [], current key = ""
	err = p.getParseToken(
		s,
		parseObjects, ignoreObjects,
		maxDepth,
		[]string{},
		nil,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to parse JSON stream: %w", err)
	}

	p.logger.Info("successfully completed JSON stream parsing")
	return p.Root, nil
}

// Use this function when previous token is already parsed
//
//	like for example key in an object is already read for path and we need to read the value
func (p *parser) getParseToken(
	s Stream,
	parseObjects, ignoreObjects [][]string,
	maxDepth int,
	currentPath []string,
	parent *FieldInfo,
) error {
	pathStr := PathToString(currentPath)

	if !p.shouldParse(parseObjects, ignoreObjects, maxDepth, currentPath) {
		p.logger.Debug("skipping value at path", "path", pathStr)
		err := s.SkipValue()
		if err != nil {
			return fmt.Errorf("failed to skip value by path %s: %w", pathStr, err)
		}
		return nil
	}

	p.logger.Debug("reading token", "path", pathStr)
	token, err := s.Token()
	if err != nil {
		return fmt.Errorf("failed to read token by path %s: %w", pathStr, err)
	}

	return p.parseToken(s, token, parseObjects, ignoreObjects, maxDepth, currentPath, parent)
}

// Call this function when
func (p *parser) parseToken(
	s Stream,
	token json.Token,
	parseObjects, ignoreObjects [][]string,
	maxDepth int,
	currentPath []string,
	parent *FieldInfo,
) error {
	pathStr := PathToString(currentPath)

	switch t := token.(type) {
	case json.Delim:
		switch t {
		case '{':
			p.logger.Debug("parsing object", "path", pathStr)
			return p.parseObject(s, parseObjects, ignoreObjects, maxDepth, currentPath, parent)
		case '[':
			p.logger.Debug("parsing array", "path", pathStr)
			return p.parseArray(s, parseObjects, ignoreObjects, maxDepth, currentPath, parent)
		}
	case nil:
		p.logger.Debug("detected null", "path", pathStr)
		p.recordType(parent, currentPath, TypeNull) // null
	case bool:
		p.logger.Debug("detected bool", "path", pathStr, "value", t)
		p.recordType(parent, currentPath, TypeBool)
	case float64:
		// Determine if it's int32, int64, or float64
		detectedType := detectNumberType(t)
		p.logger.Debug("detected number", "path", pathStr, "type", detectedType, "value", t)
		p.recordType(parent, currentPath, detectedType)
	case json.Number:
		detectedType := detectNumberTypeFromString(string(t))
		p.logger.Debug("detected number (json.Number)", "path", pathStr, "type", detectedType, "value", t)
		p.recordType(parent, currentPath, detectedType)
	case string:
		p.logger.Debug("detected string", "path", pathStr, "length", len(t))
		p.recordType(parent, currentPath, TypeString)
	}
	return nil
}

func (p *parser) parseObject(
	s Stream,
	parseObjects, ignoreObjects [][]string,
	maxDepth int,
	objPath []string,
	parent *FieldInfo,
) error {
	pathStr := PathToString(objPath)
	p.logger.Debug("entering object", "path", pathStr)

	var objItem *FieldInfo
	objType := TypeObj

	// Used for predicting object type + early exit if first token is delim
	firstToken, err := s.Token()
	if err != nil {
		return fmt.Errorf("failed to read first token in object by path %s: %w", pathStr, err)
	}
	if IsDelim(firstToken, '}') {
		p.logger.Debug("empty object detected", "path", pathStr)
		p.recordType(parent, objPath, objType)
		return nil
	}
	firstKey, isKey := firstToken.(string)
	if !isKey {
		return fmt.Errorf("failed to parse object %s: expected key (string) or '}' as a token, got: %v", pathStr, firstToken)
	}

	if isIntegerKey(firstKey) {
		p.logger.Debug("integer key detected, treating as TypeObjInt", "path", pathStr, "firstKey", firstKey)
		objType = TypeObjInt
	}

	objItem = p.recordType(parent, objPath, objType)
	err = p.getParseToken(
		s, parseObjects, ignoreObjects,
		maxDepth,
		append(objPath, firstKey),
		objItem,
	)
	if err != nil {
		return err
	}

	var i int
	for {
		token, err := s.Token()
		if err != nil {
			return fmt.Errorf("failed to read token in object path %s on iteration %d: %w", pathStr, i, err)
		}
		if IsDelim(token, '}') {
			p.logger.Debug("closing object", "path", pathStr, "totalKeys", i+1)
			return nil
		}

		key, isKey := token.(string)
		if !isKey {
			return fmt.Errorf("failed to parse object %s on iteration %d: expected key (string) or '}' as a token, got: %v", pathStr, i, token)
		}

		iterationPath := append(objPath, key)
		err = p.getParseToken(s, parseObjects, ignoreObjects, maxDepth, iterationPath, objItem)
		if err != nil {
			return err
		}

		i++
	}
}

func (p *parser) parseArray(
	s Stream,
	parseObjects, ignoreObjects [][]string,
	maxDepth int,
	arrayPath []string,
	parent *FieldInfo,
) error {
	pathStr := PathToString(arrayPath)
	p.logger.Debug("entering array", "path", pathStr)

	arrayItem := p.recordType(parent, arrayPath, TypeArray)
	var i int
	for {
		iterationPath := append(arrayPath, fmt.Sprintf("%d", i))
		if !p.shouldParse(parseObjects, ignoreObjects, maxDepth, iterationPath) {
			p.logger.Debug("skipping array element", "path", PathToString(iterationPath))
			err := s.SkipValue()
			if err != nil {
				return fmt.Errorf("failed to skip value by path %s iteration %d: %w", pathStr, i, err)
			}
			i++
			continue
		}

		token, err := s.Token()
		if err != nil {
			return fmt.Errorf("failed to read token in array %s iteration %d: %w", pathStr, i, err)
		}
		// Parsing children until the closing array delim
		if IsDelim(token, ']') {
			p.logger.Debug("closing array", "path", pathStr, "totalElements", i)
			return nil
		}
		// Parsing whatever was on that token (maybe delim for opening some other object, maybe some primitive value)
		err = p.parseToken(s, token, parseObjects, ignoreObjects, maxDepth, iterationPath, arrayItem)
		if err != nil {
			return err
		}
		i++
	}
}

func (p *parser) recordType(
	parent *FieldInfo,
	currentPath []string,
	detectedType DetectedType,
) *FieldInfo {
	pathStr := PathToString(currentPath)
	pathCopy := make([]string, len(currentPath))
	copy(pathCopy, currentPath)

	item := &FieldInfo{
		Parent:      parent,
		Path:        pathCopy,
		Type:        detectedType,
		Children:    make([]*FieldInfo, 0),
		ChildrenMap: make(map[string]*FieldInfo, 0),
	}

	if parent != nil {
		idx := currentPath[len(currentPath)-1]
		parent.Children = append(parent.Children, item)
		parent.ChildrenMap[idx] = item
		p.logger.Debug("recorded field type", "path", pathStr, "type", detectedType, "parentPath", PathToString(parent.Path))
	} else {
		p.Root = item
		p.logger.Debug("recorded root type", "type", detectedType)
	}

	return item
}

func (p *parser) shouldParse(
	parseObjects, ignoreObjects [][]string,
	maxDepth int,
	currentPath []string,
) bool {
	if maxDepth > 0 && len(currentPath) > maxDepth {
		return false
	}
	if len(parseObjects) == 0 {
		// blacklist scenario
		for _, ignoreObject := range ignoreObjects {
			if pathMatches(ignoreObject, currentPath) {
				return false
			}
		}
		return true
	}

	// whitelist scenario
	for _, parseObject := range parseObjects {
		if !pathMatches(currentPath, parseObject) {
			return false
		}
		for _, ignoreObject := range ignoreObjects {
			if pathMatches(ignoreObject, currentPath) {
				return false
			}
		}
	}
	return true
}
