package webmention

import (
	"context"
	"fmt"
	"path/filepath"
	"slices"
	"sync"
	"time"

	"github.com/charmbracelet/log"
)

var SaveFunc = Save

type Saver interface {
	Save(filepath string, mentions []Mention) error
}

var LoadFunc = LoadMentions

type Loader interface {
	LoadMentions(context.Context) ([]Mention, error)
}

type FetchWorker struct {
	observers []MentionObserver
}

func (fetchWorker *FetchWorker) AddObserver(observer MentionObserver) {
	if !slices.Contains(fetchWorker.observers, observer) {
		log.Debug("Adding observer")
		fetchWorker.observers = append(fetchWorker.observers, observer)
	}
}

type Fetchable interface {
	DoFetch(context.Context, Getter, chan<- Mention) error
}

func DoFetch(ctx context.Context, client Client, mentionChannel chan Mention, worker Fetchable) error {
	return worker.DoFetch(ctx, &client, mentionChannel)
}

// DoFetch fetches mentions and sends them on mentionChannel until the source returns an empty response
func (fetchWorker *FetchWorker) DoFetch(ctx context.Context, client Getter, mentionChannel chan<- Mention) error {
	log.Debug("starting fetch worker", "observers", len(fetchWorker.observers))
	defer func() {
		log.Debug("finished fetch worker")
		close(mentionChannel)
	}()

	result, err := client.GetMentions()
	if err != nil {
		return fmt.Errorf("error getting mentions: %v", err.Error())
	}
	for len(result.Children) > 0 {
		select {
		case <-ctx.Done():
			return ctx.Err() // return when context is canceled
		default:
			// Send mentions to the channel, or return error if unable to proceed
			for _, child := range result.Children {
				fetchWorker.updateObservers(child)
				select {
				case mentionChannel <- child:
				case <-ctx.Done():
					return ctx.Err()
				}
			}

			result, err = client.GetMentions()
			if err != nil {
				return fmt.Errorf("error getting mentions: %w", err)
			}
		}
	}
	return nil
}

func (fetchWorker *FetchWorker) updateObservers(child Mention) {
	for _, observer := range fetchWorker.observers {
		observer.Update(child)
	}
}

func (fetchWorker *FetchWorker) AddObservers(observers []MentionObserver) {
	for _, observer := range observers {
		fetchWorker.AddObserver(observer)
	}
}

type MetricsResponse struct {
	MaxID            int
	EarliestReceived time.Time
	LatestReceived   time.Time
	AllSenders       []string
	MentionsSeen     int
	UniqueMentions   []int
}

type MetricsObserver struct {
	MaxID            int
	EarliestReceived time.Time
	LatestReceived   time.Time
	AllSenders       []string
	MentionsSeen     int
	UniqueMentions   []int
	mu               sync.RWMutex
}

// Update performs the necessary operations on an observed mention to main it's set of metrics.
func (m *MetricsObserver) Update(mention Mention) {
	m.mu.Lock()
	defer m.mu.Unlock()

	log.Info("Observing mention", "WMID", mention.WMID)

	if m.MaxID == 0 || mention.WMID > m.MaxID {
		m.MaxID = mention.WMID
	}

	if m.EarliestReceived.Equal(time.Time{}) || m.EarliestReceived.After(mention.WMReceived) {
		m.EarliestReceived = mention.WMReceived
	}

	if m.LatestReceived.Equal(time.Time{}) || m.LatestReceived.Before(mention.WMReceived) {
		m.LatestReceived = mention.WMReceived
	}

	if !slices.Contains(m.AllSenders, mention.WMSource) {
		m.AllSenders = append(m.AllSenders, mention.WMSource)
	}

	if !slices.Contains(m.UniqueMentions, mention.WMID) {
		m.UniqueMentions = append(m.UniqueMentions, mention.WMID)
	}

	m.MentionsSeen++
}

func (m *MetricsObserver) GetMetrics() MetricsResponse {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return MetricsResponse{
		MaxID:            m.MaxID,
		EarliestReceived: m.EarliestReceived,
		LatestReceived:   m.LatestReceived,
		AllSenders:       m.AllSenders,
		MentionsSeen:     m.MentionsSeen,
		UniqueMentions:   m.UniqueMentions,
	}
}

type MentionObserver interface {
	Update(mention Mention)
}

type PersistenceWorker struct {
	observers []MentionObserver
}

func (w *PersistenceWorker) AddObserver(observer MentionObserver) {
	if !slices.Contains(w.observers, observer) {
		w.observers = append(w.observers, observer)
	}
}

type Persistable interface {
	DoPersist([]Mention, *sync.WaitGroup) error
}

func DoPersist(fetchedMentions []Mention, s *sync.WaitGroup, persistable Persistable) error {
	return persistable.DoPersist(fetchedMentions, s)
}

func (w *PersistenceWorker) DoPersist(fetchedMentions []Mention, s *sync.WaitGroup) error {
	defer func() {
		s.Done()
		log.Debug("ending persist worker")
	}()
	log.Debug("starting persist worker")

	if len(fetchedMentions) == 0 {
		return fmt.Errorf("got an empty list of mentions")
	}

	slug, err := fetchedMentions[0].GenerateSlug()
	filePath := filepath.Join("data", "webmentions", fmt.Sprintf("%s.json", slug))

	// Save or update the webmention
	previouslyRetrievedMentions, err := LoadFunc(filePath)
	if err != nil {
		return fmt.Errorf("failed to save webmention: %v", err)
	}

	for _, m := range fetchedMentions {
		log.Debugf("Inserting mention %d", m.WMID)
		previouslyRetrievedMentions = InsertMention(previouslyRetrievedMentions, m)
		w.updateObservers(m)
	}

	log.Infof("Saving %d mentions to %s", len(previouslyRetrievedMentions), filePath)
	err = SaveFunc(filePath, previouslyRetrievedMentions)
	if err != nil {
		return fmt.Errorf("failed to save webmention: %v", err)

	}
	return nil
}

func (w *PersistenceWorker) updateObservers(mention Mention) {
	for _, o := range w.observers {
		o.Update(mention)
	}
}
