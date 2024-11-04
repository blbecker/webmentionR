package state

import (
	"fmt"
	. "github.com/smartystreets/goconvey/convey"
	"io/fs"
	"os"
	"testing"
)

// MockFileWriter is a custom mock for testing.
type MockFileWriter struct {
	WantedErr      error
	TargetFilePath string
	Data           []byte
	Perm           os.FileMode
}

type MockFileReader struct {
	WantedErr      error
	WantedData     []byte
	TargetFilePath string
}

// WriteFile calls the mocked WriteFile function.
func (m *MockFileWriter) WriteFile(filepath string, data []byte, perm os.FileMode) error {
	m.TargetFilePath = filepath
	m.Data = data
	m.Perm = perm
	return m.WantedErr
}

func (m *MockFileReader) ReadFile(filepath string) ([]byte, error) {
	m.TargetFilePath = filepath
	return m.WantedData, m.WantedErr
}

func TestReadState(t *testing.T) {
	Convey("Given a valid state file", t, func() {
		mockReader := &MockFileReader{
			WantedData: []byte("{\"sinceID\":123}"),
			WantedErr:  nil,
		}

		// Override the global ReadFileFunc
		ReadFileFunc = mockReader.ReadFile
		Convey("When I read the state", func() {
			state, err := ReadState("dummy_path")

			Convey("Then I should get the correct state without an error", func() {
				So(err, ShouldBeNil)
				So(state.SinceID, ShouldEqual, 123)
			})
		})
	})

	Convey("Given a non-existent state file", t, func() {
		mockReader := &MockFileReader{
			WantedData: []byte(""),
			WantedErr:  fs.ErrNotExist,
		}

		// Override the global ReadFileFunc
		ReadFileFunc = mockReader.ReadFile

		Convey("When I read the state", func() {
			state, err := ReadState("dummy_path")

			Convey("Then I should get the default state without an error", func() {
				So(err, ShouldBeNil)
				So(state.SinceID, ShouldEqual, 0)
			})
		})
	})

	Convey("Given an invalid JSON state file", t, func() {
		mockReader := &MockFileReader{
			WantedData: []byte("flerp"),
			WantedErr:  nil,
		}

		// Override the global ReadFileFunc
		ReadFileFunc = mockReader.ReadFile

		Convey("When I read the state", func() {
			state, err := ReadState("dummy_path")

			Convey("Then I should receive an error and the default state", func() {
				So(err, ShouldNotBeNil)
				So(state.SinceID, ShouldEqual, 0)
			})
		})
	})
}

func TestWriteState(t *testing.T) {
	Convey("Given a valid state to write", t, func() {
		mockWriter := &MockFileWriter{
			WantedErr: nil,
		}

		// Override the global WriteFileFunc
		WriteFileFunc = mockWriter.WriteFile

		state := &State{SinceID: 456}

		Convey("When I write the state", func() {
			err := WriteState("dummy_path", state)

			Convey("Then I should succeed without errors", func() {
				So(err, ShouldBeNil)
			})

			Convey("And the buffer should contain the correct JSON", func() {
				dataString := string(mockWriter.Data)
				So(dataString, ShouldContainSubstring, "456")
			})
		})
	})

	Convey("Given a state that cannot be marshalled", t, func() {
		mockWriter := &MockFileWriter{
			WantedErr: fmt.Errorf("error marshalling state"),
		}

		state := &State{}
		WriteFileFunc = mockWriter.WriteFile
		Convey("When I write the state", func() {
			err := WriteState("dummy_path", state)

			Convey("Then I should receive an error", func() {
				So(err, ShouldNotBeNil)
			})
		})
	})

	Convey("Given a failure during writing", t, func() {
		mockWriter := &MockFileWriter{
			WantedErr: fmt.Errorf("error loading state file"),
		}

		// Override the global WriteFileFunc
		WriteFileFunc = mockWriter.WriteFile

		state := &State{SinceID: 789}

		Convey("When I write the state", func() {
			err := WriteState("dummy_path", state)

			Convey("Then I should receive an error", func() {
				So(err, ShouldNotBeNil)
			})
		})
	})
}
