package migration

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"regexp"
	"strings"
	"sync"

	"github.com/glugox/unogo/log"
)

type parserState int
type stateMachine parserState

const (
	start                   parserState = iota // 0
	gooseUp                                    // 1
	gooseStatementBeginUp                      // 2
	gooseStatementEndUp                        // 3
	gooseDown                                  // 4
	gooseStatementBeginDown                    // 5
	gooseStatementEndDown                      // 6
)

func (s *stateMachine) Get() parserState {
	return parserState(*s)
}
func (s *stateMachine) Set(new parserState) {
	log.Debug("StateMachine: %v => %v", *s, new)
	*s = stateMachine(new)
}

const scanBufSize = 4 * 1024 * 1024

var matchEmptyLines = regexp.MustCompile(`^\s*$`)

var bufferPool = sync.Pool{
	New: func() interface{} {
		return make([]byte, scanBufSize)
	},
}

func parseSQLMigration(r io.Reader, direction bool) (stmts []string, useTx bool, err error) {
	var buf bytes.Buffer
	scanBuf := bufferPool.Get().([]byte)
	defer bufferPool.Put(scanBuf)

	scanner := bufio.NewScanner(r)
	scanner.Buffer(scanBuf, scanBufSize)

	stateMachine := stateMachine(start)
	useTx = true

	for scanner.Scan() {
		line := scanner.Text()
		if verbose {
			log.Debug(line)
		}

		if strings.HasPrefix(line, "--") {
			cmd := strings.TrimSpace(strings.TrimPrefix(line, "--"))

			switch cmd {
			case "+goose Up":
				switch stateMachine.Get() {
				case start:
					stateMachine.Set(gooseUp)
				default:
					return nil, false, fmt.Errorf("duplicate '-- +goose Up' annotations; stateMachine=%v, see https://github.com/pressly/goose#sql-migrations", stateMachine)
				}
				continue

			case "+goose Down":
				switch stateMachine.Get() {
				case gooseUp, gooseStatementEndUp:
					stateMachine.Set(gooseDown)
				default:
					return nil, false, fmt.Errorf("must start with '-- +goose Up' annotation, stateMachine=%v, see https://github.com/pressly/goose#sql-migrations", stateMachine)
				}
				continue

			case "+goose StatementBegin":
				switch stateMachine.Get() {
				case gooseUp, gooseStatementEndUp:
					stateMachine.Set(gooseStatementBeginUp)
				case gooseDown, gooseStatementEndDown:
					stateMachine.Set(gooseStatementBeginDown)
				default:
					return nil, false, fmt.Errorf("'-- +goose StatementBegin' must be defined after '-- +goose Up' or '-- +goose Down' annotation, stateMachine=%v, see https://github.com/pressly/goose#sql-migrations", stateMachine)
				}
				continue

			case "+goose StatementEnd":
				switch stateMachine.Get() {
				case gooseStatementBeginUp:
					stateMachine.Set(gooseStatementEndUp)
				case gooseStatementBeginDown:
					stateMachine.Set(gooseStatementEndDown)
				default:
					return nil, false, errors.New("'-- +goose StatementEnd' must be defined after '-- +goose StatementBegin', see https://github.com/pressly/goose#sql-migrations")
				}

			case "+goose NO TRANSACTION":
				useTx = false
				continue

			default:
				// Ignore comments.
				log.Debug("StateMachine: ignore comment")
				continue
			}
		}

		// Ignore empty lines.
		if matchEmptyLines.MatchString(line) {
			log.Debug("StateMachine: ignore empty line")
			continue
		}

		// Write SQL line to a buffer.
		if _, err := buf.WriteString(line + "\n"); err != nil {
			return nil, false, fmt.Errorf("failed to write to buf: %w", err)
		}

		// Read SQL body one by line, if we're in the right direction.
		//
		// 1) basic query with semicolon; 2) psql statement
		//
		// Export statement once we hit end of statement.
		switch stateMachine.Get() {
		case gooseUp, gooseStatementBeginUp, gooseStatementEndUp:
			if !direction /*down*/ {
				buf.Reset()
				log.Debug("StateMachine: ignore down")
				continue
			}
		case gooseDown, gooseStatementBeginDown, gooseStatementEndDown:
			if direction /*up*/ {
				buf.Reset()
				log.Debug("StateMachine: ignore up")
				continue
			}
		default:
			return nil, false, fmt.Errorf("failed to parse migration: unexpected state %q on line %q, see https://github.com/pressly/goose#sql-migrations", stateMachine, line)
		}

		switch stateMachine.Get() {
		case gooseUp:
			if endsWithSemicolon(line) {
				stmts = append(stmts, buf.String())
				buf.Reset()
				log.Debug("StateMachine: store simple Up query")
			}
		case gooseDown:
			if endsWithSemicolon(line) {
				stmts = append(stmts, buf.String())
				buf.Reset()
				log.Debug("StateMachine: store simple Down query")
			}
		case gooseStatementEndUp:
			stmts = append(stmts, buf.String())
			buf.Reset()
			log.Debug("StateMachine: store Up statement")
			stateMachine.Set(gooseUp)
		case gooseStatementEndDown:
			stmts = append(stmts, buf.String())
			buf.Reset()
			log.Debug("StateMachine: store Down statement")
			stateMachine.Set(gooseDown)
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, false, fmt.Errorf("failed to scan migration: %w", err)
	}
	// EOF

	switch stateMachine.Get() {
	case start:
		return nil, false, errors.New("failed to parse migration: must start with '-- +goose Up' annotation, see https://github.com/pressly/goose#sql-migrations")
	case gooseStatementBeginUp, gooseStatementBeginDown:
		return nil, false, errors.New("failed to parse migration: missing '-- +goose StatementEnd' annotation")
	}

	if bufferRemaining := strings.TrimSpace(buf.String()); len(bufferRemaining) > 0 {
		return nil, false, fmt.Errorf("failed to parse migration: state %q, direction: %v: unexpected unfinished SQL query: %q: missing semicolon?", stateMachine, direction, bufferRemaining)
	}

	return stmts, useTx, nil
}

// Checks the line to see if the line has a statement-ending semicolon
// or if the line contains a double-dash comment.
func endsWithSemicolon(line string) bool {
	scanBuf := bufferPool.Get().([]byte)
	defer bufferPool.Put(scanBuf)

	prev := ""
	scanner := bufio.NewScanner(strings.NewReader(line))
	scanner.Buffer(scanBuf, scanBufSize)
	scanner.Split(bufio.ScanWords)

	for scanner.Scan() {
		word := scanner.Text()
		if strings.HasPrefix(word, "--") {
			break
		}
		prev = word
	}

	return strings.HasSuffix(prev, ";")
}
