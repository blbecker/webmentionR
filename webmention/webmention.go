package webmention

import (
	"encoding/json"
	"fmt"
	"github.com/charmbracelet/log"
	"net/url"
	"os"
	"sort"
	"strings"
	"time"
)

// Response Define structs based on the JSON structure
type Response struct {
	Type     string    `json:"type" faker:"oneof: 15, 16"`
	Name     string    `json:"name" faker:"oneof: entry"`
	Children []Mention `json:"children"`
}

type Mention struct {
	Type       string    `json:"type" faker:"oneof: entry"`
	Author     Author    `json:"author"`
	URL        string    `json:"url" faker:"url"`
	Published  time.Time `json:"published"`
	WMReceived time.Time `json:"wm-received"`
	WMID       int       `json:"wm-id" faker:"unique"`
	WMSource   string    `json:"wm-source" faker:"url"`
	WMTarget   string    `json:"wm-target" faker:"url"`
	WMProtocol string    `json:"wm-protocol"`
	Name       string    `json:"name"`
	Content    Content   `json:"content"`
	InReplyTo  string    `json:"in-reply-to" faker:"url"`
	WMProperty string    `json:"wm-property" faker:"oneof: in-reply-to"`
	WMPrivate  bool      `json:"wm-private"`
}

type Author struct {
	Type  string `json:"type" faker:"oneof: card"`
	Name  string `json:"name" faker:"name"`
	Photo string `json:"photo" faker:"url"`
	URL   string `json:"url" faker:"url"`
}

type Content struct {
	HTML string `json:"html" faker:"paragraph"`
	Text string `json:"text" faker:"paragraph"`
}

// GenerateSlug creates a slug based on the WMTarget URL
func (m *Mention) GenerateSlug() (string, error) {
	targetURL, err := url.Parse(m.WMTarget)
	if err != nil {
		return "", fmt.Errorf("error parsing targetURL: %v", err.Error())
	}
	slug := strings.ReplaceAll(strings.TrimPrefix(targetURL.Path, "/"), "/", "--")
	return strings.TrimSuffix(slug, "--"), nil // Remove any trailing "--" if present
}

func LoadMentions(path string) ([]Mention, error) {
	// Variable to hold mentions, initialize empty slice
	var mentions []Mention

	fileData, err := ReadFileFunc(path)
	if err != nil {
		return nil, fmt.Errorf("error reading file: %v", err.Error())
	}
	// Check if the fileData has content
	if len(fileData) > 0 {
		if err := json.Unmarshal(fileData, &mentions); err != nil {
			return nil, fmt.Errorf("error unmarshalling JSON: %w", err)
		}
	}

	return mentions, nil
}

func InsertMention(mentions []Mention, mention Mention) []Mention {
	// Check if the mention already exists in the slice
	for _, existingMention := range mentions {
		if existingMention.WMID == mention.WMID {
			log.Infof("Mention with WMID %d already exists, skipping insertion.", mention.WMID)
			return mentions // Exit without modifying the output
		}
	}

	// Append the new mention to the list
	mentions = append(mentions, mention)

	// Sort mentions by WMID in descending order
	sort.Slice(mentions, func(i, j int) bool {
		return mentions[i].WMID > mentions[j].WMID
	})
	return mentions
}

// Save adds the current mention to a JSON file using provided io.Reader and io.Writer
func Save(filepath string, mentions []Mention) error {
	// Marshal the updated mentions list back to JSON
	data, err := json.MarshalIndent(mentions, "", "  ")
	if err != nil {
		return fmt.Errorf("error marshalling JSON: %w", err)
	}

	// Write the updated JSON data to the provided io.Writer
	if err := WriteFileFunc(filepath, data, 0644); err != nil {
		return fmt.Errorf("error writing to file: %v", err.Error())
	}

	return nil
}

var WriteFileFunc = os.WriteFile

// FileWriter defines the WriteFile method for saving data to the filesystem.
type FileWriter interface {
	WriteFile(filepath string, data []byte, perm os.FileMode) error
}

var ReadFileFunc = os.ReadFile

type FileReader interface {
	ReadFile(filepath string) ([]byte, error)
}
