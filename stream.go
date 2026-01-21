package jsontype

import (
	"encoding/json"
	"io"
)

type Stream interface {
	// NextToken returns next token from a json stream
	// A Token holds a value of one of these types:
	//
	//   - [Delim], for the four JSON delimiters [ ] { }
	//   - bool, for JSON booleans
	//   - float64, for JSON numbers
	//   - [Number], for JSON numbers
	//   - string, for JSON string literals
	//   - nil, for JSON null
	//
	// At the end of the stream returns EOF
	Token() (json.Token, error)
	More() bool
	SkipValue() error
}

type DefaultStream struct {
	*json.Decoder
}

func NewJSONStream(r io.Reader) *DefaultStream {
	return &DefaultStream{Decoder: json.NewDecoder(r)}
}

func (s *DefaultStream) SkipValue() error {
	token, err := s.Token()
	if err != nil {
		return err
	}

	switch d := token.(type) {
	case json.Delim:
		switch d {
		case '{':
			for s.More() {
				if _, err = s.Token(); err != nil { // key
					return err
				}
				if err = s.SkipValue(); err != nil {
					return err
				}
			}
			_, err = s.Token() // '}'
			return err

		case '[':
			for s.More() {
				if err = s.SkipValue(); err != nil {
					return err
				}
			}
			_, err = s.Token() // ']'
			return err
		}
	}
	return nil
}
