package shcmd

import (
	"bytes"
	"errors"
	"fmt"
	"os"
)

// A Scanner parses shell-style command text. The zero value is ready
// for use.
type Scanner struct {
	Options Options

	cursor              cursor
	additionalLineCount int
	words               []token
	buffer              []byte
	envText             string
	hasEnvText          bool
}

// Options configures a Scanner. The zero value is ready for use.
type Options struct {
	// ExpandEnvVars indicates whether environment variables in
	// the form $NAME or ${NAME} should be expanded.
	ExpandEnvVars bool
	// Getenv specifies a function to retrieve the value of an
	// environment variable. If nil, os.Getenv is used.
	Getenv func(string) string
}

type token struct {
	Kind       tokenKind
	DataOffset int
	DataLength int
}

type tokenKind int

const (
	tokenWord tokenKind = 1 + iota
	tokenEndOfCommand
	tokenEndOfText
)

// SetText sets the text for the Scanner to parse. It resets the
// line count. Internal buffers are kept for reuse. The text will
// not be modified by the Scanner.
func (s *Scanner) SetText(text []byte) {
	s.cursor = cursor{Text: text}
	if n := len(text); n >= 1 && text[n-1] != '\n' {
		s.additionalLineCount = 1
	} else {
		s.additionalLineCount = 0
	}
}

// SkipLines skips the next n lines without parsing them but counts them as
// processed. The number of lines actually skipped may be less than n when
// the end of the text is reached.
func (s *Scanner) SkipLines(n int) { s.cursor.SkipLines(n) }

var (
	// ErrUnterminatedBackslashEscape is returned by NextCommand when a
	// backslash at the end of the input is not followed by any character.
	ErrUnterminatedBackslashEscape = errors.New("shcmd: unterminated backslash-escape")
	// ErrUnterminatedSingleQuotedString is returned by NextCommand when a
	// single-quoted string is not terminated.
	ErrUnterminatedSingleQuotedString = errors.New("shcmd: unterminated single-quoted string")
	// ErrUnterminatedDoubleQuotedString is returned by NextCommand when a
	// double-quoted string is not terminated.
	ErrUnterminatedDoubleQuotedString = errors.New("shcmd: unterminated double-quoted string")
	// ErrUnterminatedDollarSingleQuotedString is returned by NextCommand when a
	// dollar-single-quoted string is not terminated.
	ErrUnterminatedDollarSingleQuotedString = errors.New("shcmd: unterminated dollar-single-quoted string")
	// ErrBadSubstitution is returned by NextCommand when a bad
	// substitution is detected.
	ErrBadSubstitution = errors.New("shcmd: bad substitution")
)

// NextCommand parses and returns the next command as a slice of
// arguments, with each argument fully unquoted and unescaped.
// Blank lines and comments are skipped. The second return value
// reports whether a command was found; it is false when the text is
// exhausted.
//
// If a command was found, the slice contains at least one argument.
func (s *Scanner) NextCommand() ([]string, bool, error) {
	c0 := s.cursor

	s.words = s.words[:0]
	s.buffer = s.buffer[:0]
	s.envText = ""
	s.hasEnvText = false

	for commandHasStarted := false; ; {
		token, err := s.nextToken(commandHasStarted)
		if err != nil {
			// restore the cursor
			s.cursor = c0
			return nil, false, err
		}
		switch token.Kind {
		case tokenWord:
			s.words = append(s.words, token)
			commandHasStarted = true
			continue

		case tokenEndOfCommand:
			// copy the whole buffer at once
			data := string(s.buffer)
			command := make([]string, len(s.words))
			for i, v := range s.words {
				command[i] = data[v.DataOffset : v.DataOffset+v.DataLength]
			}
			return command, true, nil

		default: // tokenEndOfText:
			return nil, false, nil
		}
	}
}

func (s *Scanner) nextToken(commandHasStarted bool) (token, error) {
	wordHasStarted := false
	dataOffset := len(s.buffer)
Loop:
	for {
		isAtEndOfEnvText := false
		if envToken := s.nextEnvToken(); envToken.Kind != 0 {
			switch envToken.Kind {
			case envTokenLetters:
				s.buffer = append(s.buffer, envToken.Data...)
				wordHasStarted = true
				continue Loop

			case envTokenEndOfWord:
				if wordHasStarted {
					return token{
						Kind:       tokenWord,
						DataOffset: dataOffset,
						DataLength: len(s.buffer) - dataOffset,
					}, nil
				}
				continue Loop

			default: // envTokenEndOfText:
				isAtEndOfEnvText = true
			}
		}

		pc := s.cursor
		b, ok := s.cursor.NextByte()
		if !ok {
			if wordHasStarted {
				return token{
					Kind:       tokenWord,
					DataOffset: dataOffset,
					DataLength: len(s.buffer) - dataOffset,
				}, nil
			}
			if commandHasStarted {
				return token{Kind: tokenEndOfCommand}, nil
			}
			return token{Kind: tokenEndOfText}, nil
		}

		switch b {
		case ' ', '\t', '\n':
			if wordHasStarted {
				// move back the cursor
				s.cursor = pc
				return token{
					Kind:       tokenWord,
					DataOffset: dataOffset,
					DataLength: len(s.buffer) - dataOffset,
				}, nil
			}
			if b == '\n' && commandHasStarted {
				return token{Kind: tokenEndOfCommand}, nil
			}

		case '#':
			if !wordHasStarted && !isAtEndOfEnvText {
				s.cursor.SkipUntilNewline()
			} else {
				s.buffer = append(s.buffer, '#')
				wordHasStarted = true
			}

		case '\\':
			b, ok = s.cursor.NextByte()
			if !ok {
				return token{}, fmt.Errorf("%w at line %v", ErrUnterminatedBackslashEscape, s.cursor.LineCount+1)
			}
			if b == '\n' {
				// line continuation
			} else {
				s.buffer = append(s.buffer, b)
				wordHasStarted = true
			}

		case '\'':
			if err := s.handleSingleQuotedString(); err != nil {
				return token{}, err
			}
			wordHasStarted = true

		case '"':
			if err := s.handleDoubleQuotedString(); err != nil {
				return token{}, err
			}
			wordHasStarted = true

		case '$':
			if nb, _ := s.cursor.PeekByte(); nb == '\'' {
				s.cursor.NextByte()
				if err := s.handleDollarSingleQuotedString(); err != nil {
					return token{}, err
				}
				wordHasStarted = true
				continue Loop
			}
			if s.Options.ExpandEnvVars {
				name, ok, err := s.nextName()
				if err != nil {
					return token{}, err
				}
				if ok {
					s.envText = s.getEnvVar(name)
					s.hasEnvText = true
					continue Loop
				}
			}
			s.buffer = append(s.buffer, b)
			wordHasStarted = true

		default:
			s.buffer = append(s.buffer, b)
			wordHasStarted = true
		}
	}
}

type envToken struct {
	Kind envTokenKind
	Data string
}

type envTokenKind int

const (
	envTokenLetters envTokenKind = 1 + iota
	envTokenEndOfWord
	envTokenEndOfText
)

func (s *Scanner) nextEnvToken() envToken {
	if !s.hasEnvText {
		return envToken{}
	}
	if s.envText == "" {
		s.hasEnvText = false
		return envToken{Kind: envTokenEndOfText}
	}
	var envToken envToken
	i := 1
	switch s.envText[0] {
	case ' ', '\t', '\n':
		envToken.Kind = envTokenEndOfWord
	Loop1:
		for ; i < len(s.envText); i++ {
			switch s.envText[i] {
			case ' ', '\t', '\n':
			default:
				break Loop1
			}
		}
	default:
		envToken.Kind = envTokenLetters
	Loop2:
		for ; i < len(s.envText); i++ {
			switch s.envText[i] {
			case ' ', '\t', '\n':
				break Loop2
			default:
			}
		}
	}
	envToken.Data = s.envText[:i]
	s.envText = s.envText[i:]
	return envToken
}

func (s *Scanner) handleSingleQuotedString() error {
	c0 := s.cursor
	for {
		b, ok := s.cursor.NextByte()
		if !ok {
			return fmt.Errorf("%w at line %v", ErrUnterminatedSingleQuotedString, c0.LineCount+1)
		}
		if b == '\'' {
			return nil
		}
		s.buffer = append(s.buffer, b)
	}
}

func (s *Scanner) handleDoubleQuotedString() error {
	c0 := s.cursor
	for {
		b, ok := s.cursor.NextByte()
		if !ok {
			return fmt.Errorf("%w at line %v", ErrUnterminatedDoubleQuotedString, c0.LineCount+1)
		}
		switch b {
		case '"':
			return nil

		case '\\':
			b, ok = s.cursor.NextByte()
			if !ok {
				return fmt.Errorf("%w at line %v", ErrUnterminatedBackslashEscape, s.cursor.LineCount+1)
			}
			switch b {
			case '\n':
				// line continuation

			case '$', '`', '"', '\\':
				s.buffer = append(s.buffer, b)

			default:
				s.buffer = append(s.buffer, '\\')
				s.buffer = append(s.buffer, b)
			}

		case '$':
			if s.Options.ExpandEnvVars {
				name, ok, err := s.nextName()
				if err != nil {
					return err
				}
				if ok {
					s.buffer = append(s.buffer, s.getEnvVar(name)...)
					continue
				}
			}
			s.buffer = append(s.buffer, b)

		default:
			s.buffer = append(s.buffer, b)
		}
	}
}

func (s *Scanner) handleDollarSingleQuotedString() error {
	c0 := s.cursor
	for {
		b, ok := s.cursor.NextByte()
		if !ok {
			return fmt.Errorf("%w at line %v", ErrUnterminatedDollarSingleQuotedString, c0.LineCount+1)
		}
		switch b {
		case '\'':
			return nil

		case '\\':
			b, ok = s.cursor.NextByte()
			if !ok {
				return fmt.Errorf("%w at line %v", ErrUnterminatedBackslashEscape, s.cursor.LineCount+1)
			}
			switch b {
			case '"', '\'', '\\':
				s.buffer = append(s.buffer, b)

			case 'a':
				s.buffer = append(s.buffer, '\a')

			case 'b':
				s.buffer = append(s.buffer, '\b')

			case 'e':
				s.buffer = append(s.buffer, '\x1b')

			case 'f':
				s.buffer = append(s.buffer, '\f')

			case 'n':
				s.buffer = append(s.buffer, '\n')

			case 'r':
				s.buffer = append(s.buffer, '\r')

			case 't':
				s.buffer = append(s.buffer, '\t')

			case 'v':
				s.buffer = append(s.buffer, '\v')

			case 'x':
				n := -1
				for range 2 {
					nn := -1
					if nb, ok := s.cursor.PeekByte(); ok {
						switch {
						case nb >= '0' && nb <= '9':
							nn = int(nb) - '0'
						case nb >= 'A' && nb <= 'F':
							nn = 10 + int(nb) - 'A'
						case nb >= 'a' && nb <= 'f':
							nn = 10 + int(nb) - 'a'
						}
					}
					if nn < 0 {
						break
					}
					s.cursor.NextByte()
					n = max(n, 0)*16 + nn
				}
				if n >= 0 {
					s.buffer = append(s.buffer, byte(n))
				} else {
					s.buffer = append(s.buffer, '\\')
					s.buffer = append(s.buffer, 'x')
				}

			default:
				if b >= '0' && b <= '7' {
					n := int(b) - '0'
					for range 2 {
						nb, _ := s.cursor.PeekByte()
						if !(nb >= '0' && nb <= '7') {
							break
						}
						s.cursor.NextByte()
						n = n*8 + (int(nb) - '0')
					}
					s.buffer = append(s.buffer, byte(n))
				} else {
					s.buffer = append(s.buffer, '\\')
					s.buffer = append(s.buffer, b)
				}
			}

		default:
			s.buffer = append(s.buffer, b)
		}
	}
}

func (s *Scanner) nextName() ([]byte, bool, error) {
	c0 := s.cursor

	nameStart := s.cursor.Text
	b, ok := s.cursor.NextByte()
	if !ok {
		return nil, false, nil
	}
	nameIsBracketed := b == '{'
	if nameIsBracketed {
		nameStart = s.cursor.Text
		b, ok = s.cursor.NextByte()
		if !ok {
			return nil, false, fmt.Errorf("%w at line %v", ErrBadSubstitution, c0.LineCount+1)
		}
	}
	switch {
	case b >= 'A' && b <= 'Z':
	case b >= 'a' && b <= 'z':
	case b == '_':
	default:
		if nameIsBracketed {
			return nil, false, fmt.Errorf("%w at line %v", ErrBadSubstitution, c0.LineCount+1)
		} else {
			// move back the cursor
			s.cursor = c0
			return nil, false, nil
		}
	}
	for l := 1; ; l++ {
		pc := s.cursor
		b, ok := s.cursor.NextByte()
		if !ok {
			if nameIsBracketed {
				return nil, false, fmt.Errorf("%w at line %v", ErrBadSubstitution, c0.LineCount+1)
			} else {
				return nameStart[:l], true, nil
			}
		}
		switch {
		case b >= 'A' && b <= 'Z':
		case b >= 'a' && b <= 'z':
		case b >= '0' && b <= '9':
		case b == '_':
		default:
			if nameIsBracketed {
				if b == '}' {
					return nameStart[:l], true, nil
				} else {
					return nil, false, fmt.Errorf("%w at line %v", ErrBadSubstitution, c0.LineCount+1)
				}
			} else {
				// move back the cursor
				s.cursor = pc
				return nameStart[:l], true, nil
			}
		}
	}
}

func (s *Scanner) getEnvVar(name []byte) string {
	if f := s.Options.Getenv; f != nil {
		return f(string(name))
	}
	return os.Getenv(b2s(name))
}

// LineCount returns the number of lines processed so far. A final
// line without a trailing newline is also counted.
func (s *Scanner) LineCount() int {
	lineCount := s.cursor.LineCount
	if s.cursor.IsAtEnd() {
		// when the text is exhausted, also count the final line if it has no trailing newline
		lineCount += s.additionalLineCount
	}
	return lineCount
}

type cursor struct {
	Text      []byte
	LineCount int
}

func (c *cursor) NextByte() (byte, bool) {
	if c.IsAtEnd() {
		c.Text = nil
		return 0, false
	}
	b := c.Text[0]
	switch b {
	case '\r':
		if len(c.Text) >= 2 && c.Text[1] == '\n' {
			c.LineCount++
			c.Text = c.Text[2:]
			return '\n', true
		}
	case '\n':
		c.LineCount++
	}
	c.Text = c.Text[1:]
	return b, true
}

func (c *cursor) PeekByte() (byte, bool) {
	if c.IsAtEnd() {
		return 0, false
	}
	b := c.Text[0]
	if b == '\r' && len(c.Text) >= 2 && c.Text[1] == '\n' {
		return '\n', true
	}
	return b, true
}

func (c *cursor) SkipLines(n int) {
	for ; n >= 1; n-- {
		i := bytes.IndexByte(c.Text, '\n')
		if i < 0 {
			c.Text = nil
			return
		}
		c.Text = c.Text[i+1:]
		c.LineCount++
	}
}

func (c *cursor) SkipUntilNewline() {
	i := bytes.IndexByte(c.Text, '\n')
	if i < 0 {
		c.Text = nil
		return
	}
	c.Text = c.Text[i:]
}

func (c *cursor) IsAtEnd() bool { return len(c.Text) == 0 }
