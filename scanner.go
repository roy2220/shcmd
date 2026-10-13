// Package shcmd parses shell-style command text into commands, one
// argument slice per command. It handles quoting, backslash escapes,
// line continuations, comments, and optional environment variable expansion,
// following basic POSIX shell rules for lexing.
package shcmd

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"unsafe"
)

// A Scanner parses shell-style command text. The zero value is ready
// for use.
type Scanner struct {
	// ExpandEnvVar indicates whether environment variables in
	// the form $NAME or ${NAME} should be expanded.
	ExpandEnvVar bool
	// Getenv specifies a function to retrieve the value of an
	// environment variable. If nil, os.Getenv is used.
	Getenv func(string) string

	cursor              cursor
	additionalLineCount int
	words               []word
	buffer              []byte
	textFromEnvVar      string
	hasTextFromEnvVar   bool
}

type word struct {
	Kind       wordKind
	DataOffset int
	DataLength int
}

type wordKind int

const (
	wordOrdinary wordKind = 1 + iota
	wordEndOfCommand
	wordEndOfText
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

// SkipLines skips the next n lines without parsing them. The number of
// lines actually skipped may be less than n when the end of the text is
// reached.
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
	// ErrBadSubstitution is returned by NextCommand when a bad
	// substitution is detected.
	ErrBadSubstitution = errors.New("shcmd: bad substitution")
)

// NextCommand parses and returns the next command as a slice of
// arguments, with each argument fully unquoted and unescaped. The
// second return value reports whether a command was found; it is
// false when the text is exhausted. Blank lines and comments are
// skipped.
//
// If a command was found, the slice contains at least one argument.
func (s *Scanner) NextCommand() ([]string, bool, error) {
	c0 := s.cursor

	s.words = s.words[:0]
	s.buffer = s.buffer[:0]
	s.textFromEnvVar = ""
	s.hasTextFromEnvVar = false

	for commandHasStarted := false; ; {
		word, err := s.nextWord(commandHasStarted)
		if err != nil {
			// restore the cursor
			s.cursor = c0
			return nil, false, err
		}
		if word.Kind == wordEndOfText {
			return nil, false, nil
		}
		if word.Kind == wordEndOfCommand {
			// copy the whole buffer at once
			data := string(s.buffer)
			command := make([]string, len(s.words))
			for i, w := range s.words {
				command[i] = data[w.DataOffset : w.DataOffset+w.DataLength]
			}
			return command, true, nil
		}
		s.words = append(s.words, word)
		commandHasStarted = true
	}
}

func (s *Scanner) nextWord(commandHasStarted bool) (word, error) {
	wordHasStarted := false
	dataOffset := len(s.buffer)
Loop:
	for {
		isAtEndOfTextFromEnvVar := false
		if token, ok := s.readTextFromEnvVar(); ok {
			if token != "" {
				if token == splitter {
					if wordHasStarted {
						return word{
							Kind:       wordOrdinary,
							DataOffset: dataOffset,
							DataLength: len(s.buffer) - dataOffset,
						}, nil
					}
				} else {
					s.buffer = append(s.buffer, token...)
					wordHasStarted = true
				}
				continue Loop
			}
			isAtEndOfTextFromEnvVar = true
		}

		pc := s.cursor
		b, ok := s.cursor.NextByte()
		if !ok {
			if wordHasStarted {
				return word{
					Kind:       wordOrdinary,
					DataOffset: dataOffset,
					DataLength: len(s.buffer) - dataOffset,
				}, nil
			}
			if commandHasStarted {
				return word{Kind: wordEndOfCommand}, nil
			}
			return word{Kind: wordEndOfText}, nil
		}

		switch b {
		case ' ', '\t', '\n':
			if wordHasStarted {
				// move back the cursor
				s.cursor = pc
				return word{
					Kind:       wordOrdinary,
					DataOffset: dataOffset,
					DataLength: len(s.buffer) - dataOffset,
				}, nil
			}
			if b == '\n' && commandHasStarted {
				return word{Kind: wordEndOfCommand}, nil
			}

		case '#':
			if !wordHasStarted && !isAtEndOfTextFromEnvVar {
				s.cursor.SkipUntilNewline()
				continue Loop
			}

			s.buffer = append(s.buffer, '#')
			wordHasStarted = true

		case '\\':
			b, ok = s.cursor.NextByte()
			if !ok {
				return word{}, fmt.Errorf("%w at line %v", ErrUnterminatedBackslashEscape, s.cursor.LineCount+1)
			}
			if b == '\n' {
				// line continuation
			} else {
				s.buffer = append(s.buffer, b)
				wordHasStarted = true
			}

		case '\'':
			pc = s.cursor
			for {
				b, ok = s.cursor.NextByte()
				if !ok {
					return word{}, fmt.Errorf("%w at line %v", ErrUnterminatedSingleQuotedString, pc.LineCount+1)
				}
				if b == '\'' {
					wordHasStarted = true
					continue Loop
				}
				s.buffer = append(s.buffer, b)
			}

		case '"':
			pc = s.cursor
			for {
				b, ok = s.cursor.NextByte()
				if !ok {
					return word{}, fmt.Errorf("%w at line %v", ErrUnterminatedDoubleQuotedString, pc.LineCount+1)
				}
				switch b {
				case '"':
					wordHasStarted = true
					continue Loop

				case '\\':
					b, ok = s.cursor.NextByte()
					if !ok {
						return word{}, fmt.Errorf("%w at line %v", ErrUnterminatedBackslashEscape, s.cursor.LineCount+1)
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
					if s.ExpandEnvVar {
						name, ok, err := s.nextName()
						if err != nil {
							return word{}, err
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

		case '$':
			if s.ExpandEnvVar {
				name, ok, err := s.nextName()
				if err != nil {
					return word{}, err
				}
				if ok {
					s.textFromEnvVar = s.getEnvVar(name)
					s.hasTextFromEnvVar = true
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

const splitter = " "

func (s *Scanner) readTextFromEnvVar() (string, bool) {
	if !s.hasTextFromEnvVar {
		return "", false
	}
	if s.textFromEnvVar == "" {
		s.hasTextFromEnvVar = false
		return "", true
	}
	switch s.textFromEnvVar[0] {
	case ' ', '\t', '\n':
		i := 1
	Loop1:
		for ; i < len(s.textFromEnvVar); i++ {
			switch s.textFromEnvVar[i] {
			case ' ', '\t', '\n':
			default:
				break Loop1
			}
		}
		s.textFromEnvVar = s.textFromEnvVar[i:]
		return splitter, true
	default:
		i := 1
	Loop2:
		for ; i < len(s.textFromEnvVar); i++ {
			switch s.textFromEnvVar[i] {
			case ' ', '\t', '\n':
				break Loop2
			default:
			}
		}
		token := s.textFromEnvVar[:i]
		s.textFromEnvVar = s.textFromEnvVar[i:]
		return token, true
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
	for l := 1; ; {
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
		l++
	}
}

func (s *Scanner) getEnvVar(name []byte) string {
	if f := s.Getenv; f != nil {
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

func b2s(b []byte) string { return unsafe.String(unsafe.SliceData(b), len(b)) }
