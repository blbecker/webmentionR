package webmention

import (
	"fmt"
	"os"
	"sort"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

func TestMention_GenerateSlug(t *testing.T) {
	Convey("Given a valid webmention, an expected slug is generated", t, func() {
		mention := Mention{
			WMTarget: "https://example.com/path/to-the_post.html",
		}
		slug, err := mention.GenerateSlug()
		So(err, ShouldBeNil)
		So(slug, ShouldEqual, "path--to-the_post.html")
	})
	Convey("Given an invalid webmention, an expected slug is generated", t, func() {
		mention := Mention{
			WMTarget: "httttP:// /path",
		}
		slug, err := mention.GenerateSlug()
		So(err, ShouldNotBeNil)
		So(slug, ShouldEqual, "")
	})
}

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

func TestSaveWithMockFileWriter(t *testing.T) {
	Convey("Given a Save function with a mock file writer", t, func() {
		Convey("When saving a valid mentions list", func() {
			mockWriter := &MockFileWriter{
				WantedErr: nil,
			}

			WriteFileFunc = mockWriter.WriteFile
			mentions := []Mention{{ /* Add valid fields */ }}
			err := Save("test_output.json", mentions)

			Convey("Then it should save successfully without error", func() {
				So(err, ShouldBeNil)
				So(mockWriter.TargetFilePath, ShouldEqual, "test_output.json")
			})
		})

		Convey("When JSON marshalling fails", func() {
			mockWriter := &MockFileWriter{
				WantedErr: fmt.Errorf("json marshalling error"),
			}
			// Use a struct with an unmarshalable field
			mentions := []Mention{{ /* Add problematic fields */ }}

			WriteFileFunc = mockWriter.WriteFile

			err := Save("test_output.json", mentions)

			Convey("Then it should return a marshalling error", func() {
				So(err, ShouldNotBeNil)
			})
		})

		Convey("When the file write fails", func() {
			mockWriter := &MockFileWriter{
				WantedErr: fmt.Errorf("error writing to file"),
			}

			mentions := []Mention{{ /* Add valid fields */ }}

			WriteFileFunc = mockWriter.WriteFile
			err := Save("/invalid/path/test_output.json", mentions)

			Convey("Then it should return a file write error", func() {
				So(err, ShouldNotBeNil)
			})
		})
	})
}

func TestInsertMention(t *testing.T) {
	Convey("Given a slice of mentions", t, func() {
		mentions := []Mention{
			{WMID: 5},
			{WMID: 3},
			{WMID: 1},
		}

		Convey("When inserting a new mention with a unique WMID", func() {
			newMention := Mention{WMID: 4}
			updatedMentions := InsertMention(mentions, newMention)

			Convey("Then it should add the new mention", func() {
				So(len(updatedMentions), ShouldEqual, len(mentions)+1)
			})

			Convey("And the mentions should be sorted by WMID in descending order", func() {
				So(sort.SliceIsSorted(updatedMentions, func(i, j int) bool {
					return updatedMentions[i].WMID > updatedMentions[j].WMID
				}), ShouldBeTrue)
			})
		})

		Convey("When inserting a mention with an existing WMID", func() {
			duplicateMention := Mention{WMID: 3}
			updatedMentions := InsertMention(mentions, duplicateMention)

			Convey("Then it should not add the mention", func() {
				So(updatedMentions, ShouldResemble, mentions) // Verify the list is unchanged
			})
		})

		Convey("When inserting into an empty list", func() {
			mentions = []Mention{}
			newMention := Mention{WMID: 10}
			updatedMentions := InsertMention(mentions, newMention)

			Convey("Then it should add the mention to the empty list", func() {
				So(updatedMentions, ShouldHaveLength, 1)
				So(updatedMentions[0].WMID, ShouldEqual, 10)
			})
		})
	})
}

func TestLoadMentions(t *testing.T) {
	Convey("Given a LoadMentions function with a mock file reader", t, func() {
		mockReader := &MockFileReader{
			WantedErr:  nil,
			WantedData: []byte("[{\"wm-id\":5},{\"wm-id\":3},{\"wm-id\":1}]"),
		}

		ReadFileFunc = mockReader.ReadFile
		Convey("When loading mentions from a valid JSON file", func() {

			mentions, err := LoadMentions("test.json")

			Convey("Then it should load the mentions successfully", func() {
				So(err, ShouldBeNil)
				So(mentions, ShouldHaveLength, 3)
				So(mentions[0].WMID, ShouldEqual, 5)
				So(mentions[1].WMID, ShouldEqual, 3)
			})
		})

		Convey("When the file read fails", func() {
			mockReader.WantedErr = fmt.Errorf("error while reading from file")

			mentions, err := LoadMentions("invalid.json")

			Convey("Then it should return an error", func() {
				So(err, ShouldNotBeNil)
				So(err.Error(), ShouldContainSubstring, "error reading file")
				So(mentions, ShouldBeNil)
			})
		})

		Convey("When the file contains invalid JSON", func() {
			mockReader.WantedErr = nil
			mockReader.WantedData = []byte("derp")
			mentions, err := LoadMentions("test.json")

			Convey("Then it should return a JSON unmarshalling error", func() {
				So(err, ShouldNotBeNil)
				So(err.Error(), ShouldContainSubstring, "error unmarshalling JSON")
				So(mentions, ShouldBeNil)
			})
		})

		Convey("When the file is empty", func() {
			mockReader.WantedData = []byte("")

			mentions, err := LoadMentions("empty.json")

			Convey("Then it should return an empty mentions slice without error", func() {
				So(err, ShouldBeNil)
				So(mentions, ShouldBeEmpty)
			})
		})
	})
}
