package state

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"sync"

	"github.com/charmbracelet/log"
)

type State struct {
	mu      sync.RWMutex
	SinceID int `json:"sinceID"`
}

//===  Bindings for tests

// WriteFileFunc binds to the WriteFile function used to save the statefile.
var WriteFileFunc = os.WriteFile

// FileWriter defines the WriteFile method for saving data to the filesystem.
type FileWriter interface {
	WriteFile(filepath string, data []byte, perm os.FileMode) error
}

var ReadFileFunc = os.ReadFile

type FileReader interface {
	ReadFile(filepath string) ([]byte, error)
}

//=== Bindings for tests

func ReadState(stateFilePath string) (*State, error) {
	var fetchState State

	if stateFilePath != "" {
		fileData, err := ReadFileFunc(stateFilePath)

		if err != nil {
			if errors.Is(err, fs.ErrNotExist) {
				log.Info("Initializing new state at '%s'", stateFilePath)
				// Swallow the error if it's a non-existent file. We'll create it at the end.
				return &fetchState, nil
			}
			return &fetchState, fmt.Errorf("could not open state file: %w", err)
		}

		err = json.Unmarshal(fileData, &fetchState)
		if err != nil {
			return &fetchState, fmt.Errorf("error parsing stateFile: %v", err.Error())
		}
	}

	return &fetchState, nil
}

func WriteState(filepath string, s *State) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Marshal the updated state-file back to JSON
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return fmt.Errorf("error marshalling JSON: %w", err)
	}

	// Write the updated JSON data to the file
	if err := WriteFileFunc(filepath, data, 0644); err != nil {
		return fmt.Errorf("error writing to file: %v", err.Error())
	}

	return nil
}
