package webmention

import (
	"context"
	"fmt"
	"github.com/go-faker/faker/v4"
	"github.com/go-faker/faker/v4/pkg/interfaces"
	"github.com/go-faker/faker/v4/pkg/options"
	"sync"
	"testing"
	"time"

	"github.com/charmbracelet/log"
	. "github.com/smartystreets/goconvey/convey"
)

// generateFakeMentions creates a slice of Mention objects with faker-generated data
func generateFakeMentions(count int) []Mention {
	mentions := make([]Mention, count)
	for i := 0; i < count; i++ {
		var mention Mention
		err := faker.FakeData(&mention, options.WithRandomMapAndSliceMaxSize(20), options.WithRandomIntegerBoundaries(interfaces.RandomIntegerBoundary{
			Start: 1,
			End:   10000,
		}))
		if err != nil {
			log.Printf("error generating mention: %s", err.Error())
		}
		mentions = append(mentions, mention)
	}

	faker.ResetUnique()
	return mentions
}

// MockClient simulates the behavior of Client for controlled testing.
type MockClient struct {
	mentions   []Mention
	perPage    int
	pages      int
	page       int
	errOnFetch error // Simulate an error condition
	fetchDelay time.Duration
}

// GetMentions returns mentions based on mock state and increments call count.
func (mc *MockClient) GetMentions() (*Response, error) {
	if mc.errOnFetch != nil {
		return nil, mc.errOnFetch
	}
	if mc.page == mc.pages {
		return &Response{}, nil
	}

	result := &Response{Children: mc.mentions}
	mc.page++
	return result, nil
}

// TestFetchWorker tests the DoFetch function using Convey
func TestFetchWorker(t *testing.T) {
	Convey("Given a DoFetch with a mock client", t, func() {
		mentionChannel := make(chan Mention, 10) // Buffered to prevent blocking
		mockMentions := []Mention{
			{WMID: 1, WMSource: "https://test.example.com"},
			{WMID: 2},
			{WMID: 3},
		}
		fetchWorker := FetchWorker{}

		Convey("When DoFetch completes successfully", func() {
			mockClient := &MockClient{
				mentions: mockMentions,
				pages:    1,
				perPage:  2,
			}
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			go func() {
				err := fetchWorker.DoFetch(ctx, mockClient, mentionChannel)
				if err != nil {
					log.Error("err DoFetching", "err", err.Error())
				}
			}()

			var mentionsReceived []Mention
			for mention := range mentionChannel {
				mentionsReceived = append(mentionsReceived, mention)
			}

			// Test that all expected mentions were received
			So(len(mentionsReceived), ShouldEqual, 3)
			So(mentionsReceived, ShouldResemble, mockMentions)
		})
		Convey("DoFetch Updates Observers", func() {
			observer := MetricsObserver{}
			fetchWorker.AddObserver(&observer)

			mockClient := &MockClient{
				mentions: mockMentions,
				pages:    1,
				perPage:  2,
			}
			log.Infof("FetchWorker has %d observers", len(fetchWorker.observers))

			go func() {
				log.Infof("FetchWorker has %d observers", len(fetchWorker.observers))
				err := fetchWorker.DoFetch(context.TODO(), mockClient, mentionChannel)
				if err != nil {
					log.Error("err DoFetching", "err", err.Error())
				}
			}()

			for _ = range mentionChannel {

			}

			// Test that all expected observations were made
			So(observer.GetMetrics().MentionsSeen, ShouldEqual, 3)
			So(observer.GetMetrics().MaxID, ShouldEqual, 3)
			So(observer.GetMetrics().AllSenders, ShouldContain, "https://test.example.com")
		})
		Convey("When DoFetch times out", func() {
			mockClient := &MockClient{
				mentions:   mockMentions,
				pages:      1,
				perPage:    2,
				fetchDelay: 10 * time.Second,
			}
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			go func() {
				err := fetchWorker.DoFetch(ctx, mockClient, mentionChannel)
				if err != nil {
					log.Error("err DoFetching", "err", err.Error())
				}
			}()

			var mentionsReceived []Mention
			for mention := range mentionChannel {
				mentionsReceived = append(mentionsReceived, mention)
			}

			// Test that all expected mentions were received
			So(len(mentionsReceived), ShouldEqual, 3)
			So(mentionsReceived, ShouldResemble, mockMentions)
		})

		Convey("When DoFetch is canceled", func() {
			mockClient := &MockClient{mentions: mockMentions}
			ctx, cancel := context.WithCancel(context.Background())

			go fetchWorker.DoFetch(ctx, mockClient, mentionChannel)

			// Cancel the context to simulate interruption
			cancel()

			mentionsReceived := []Mention{}
			for mention := range mentionChannel {
				mentionsReceived = append(mentionsReceived, mention)
			}

			// Test that no mentions were sent after cancellation
			So(mentionsReceived, ShouldBeEmpty)
		})

		Convey("When DoFetch encounters an error from the client", func() {
			mockClient := &MockClient{errOnFetch: fmt.Errorf("client fetch error")}
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			err := fetchWorker.DoFetch(ctx, mockClient, mentionChannel)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, "client fetch error")
		})
	})
}

func TestFetchWorker_UpdateAll(t *testing.T) {
	Convey("DoFetch Updates Observers", t, func() {
		mockMentions := []Mention{
			{WMID: 1, WMSource: "https://test.example.com"},
			{WMID: 2},
			{WMID: 3},
		}
		fetchWorker := FetchWorker{}
		observer := MetricsObserver{}
		fetchWorker.AddObserver(&observer)

		for _, mention := range mockMentions {
			fetchWorker.updateObservers(mention)
		}

		// Test that all expected observations were made
		So(observer.GetMetrics().MentionsSeen, ShouldEqual, 3)
		So(observer.GetMetrics().MaxID, ShouldEqual, 3)
		So(observer.GetMetrics().AllSenders, ShouldContain, "https://test.example.com")
	})
}

type loadMentionsFuncMock struct {
	requestedPath  string
	wantedMentions []Mention
	wantedErr      error
}

func (m *loadMentionsFuncMock) LoadMentions(filepath string) ([]Mention, error) {
	m.requestedPath = filepath
	return m.wantedMentions, m.wantedErr
}

type saveFuncMock struct {
	requestedPath string
	savedMentions []Mention
	wantedErr     error
}

func (m *saveFuncMock) Save(filepath string, mentions []Mention) error {
	m.requestedPath = filepath
	m.savedMentions = mentions
	return m.wantedErr
}

func TestPersistenceWorker_Do(t *testing.T) {
	Convey("Given a DoPersist with mocked SaveFunc and LoadMentions", t, func() {
		var wg sync.WaitGroup
		wg.Add(1)

		// Generate mentions using faker
		mentions := generateFakeMentions(8)
		fetchedMentions := mentions[0:2]
		loadedMentions := mentions[2:]

		loadMentionsMock := loadMentionsFuncMock{
			wantedMentions: loadedMentions,
			wantedErr:      nil,
		}

		saveMock := saveFuncMock{
			wantedErr: nil,
		}

		LoadFunc = loadMentionsMock.LoadMentions
		SaveFunc = saveMock.Save

		persistenceWorker := PersistenceWorker{}

		Convey("When DoPersist completes successfully", func() {

			var err error

			// Run DoPersist with mocked functions
			go func() {
				err = persistenceWorker.DoPersist(fetchedMentions, &wg)
				if err != nil {

				}
			}()
			wg.Wait() // Wait for worker to complete

			Convey("It should not error", func() {
				So(err, ShouldBeNil)
			})
			// Verify SaveFunc was called with expected mentions
			Convey("It should call SaveFunc with the correct data", func() {
				So(saveMock.savedMentions, ShouldNotBeNil)
				So(len(saveMock.savedMentions), ShouldEqual, len(loadMentionsMock.wantedMentions))
			})
		})

		Convey("When LoadMentions returns an error, it should bubble up", func() {
			loadMentionsMock.wantedErr = fmt.Errorf("load error")
			loadMentionsMock.wantedMentions = []Mention{
				{
					WMID: 1,
				}, {
					WMID: 2,
				},
			}

			saveMock.wantedErr = nil

			var metricsObserver MetricsObserver
			persistenceWorker := PersistenceWorker{}
			persistenceWorker.AddObserver(&metricsObserver)

			var err error
			// Run DoPersist and expect it to log an error
			go func() {

				err = persistenceWorker.DoPersist(fetchedMentions, &wg)
			}()

			wg.Wait()
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, loadMentionsMock.wantedErr.Error())
		})

		Convey("When SaveFunc returns an error", func() {
			loadMentionsMock.wantedMentions = []Mention{
				{
					WMID: 1,
				}, {
					WMID: 2,
				},
			}
			loadMentionsMock.wantedErr = nil

			saveMock.wantedErr = fmt.Errorf("save error")

			var err error
			// Run DoPersist and expect it to panic
			go func() {

				err = persistenceWorker.DoPersist(fetchedMentions, &wg)
			}()
			wg.Wait()
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, saveMock.wantedErr.Error())
		})
	})
}

func TestPersistenceWorker_AddObserver(t *testing.T) {
	Convey("PersistenceWorkers can add observers", t, func() {
		var persistenceWorker PersistenceWorker
		var metricsObserver1 MetricsObserver
		var metricsObserver2 MetricsObserver

		So(len(persistenceWorker.observers), ShouldEqual, 0)
		persistenceWorker.AddObserver(&metricsObserver1)
		So(len(persistenceWorker.observers), ShouldEqual, 1)
		persistenceWorker.AddObserver(&metricsObserver2)
		So(len(persistenceWorker.observers), ShouldEqual, 2)
		persistenceWorker.AddObserver(&metricsObserver2)
		So(len(persistenceWorker.observers), ShouldEqual, 2)
	})
}
func TestFetchWorker_AddObserver(t *testing.T) {
	Convey("PersistenceWorkers can add observers", t, func() {
		var fetchWorker FetchWorker
		var metricsObserver1 MetricsObserver
		var metricsObserver2 MetricsObserver

		So(len(fetchWorker.observers), ShouldEqual, 0)
		fetchWorker.AddObserver(&metricsObserver1)
		So(len(fetchWorker.observers), ShouldEqual, 1)
		fetchWorker.AddObserver(&metricsObserver2)
		So(len(fetchWorker.observers), ShouldEqual, 2)
		fetchWorker.AddObserver(&metricsObserver2)
		So(len(fetchWorker.observers), ShouldEqual, 2)
	})
}

func TestMetricsObserver_Update(t *testing.T) {
	Convey("Given a MetricsObserver", t, func() {
		var metricsObserver MetricsObserver
		Convey("values should initialize in an expected way", func() {
			So(metricsObserver, ShouldNotBeNil)
			So(metricsObserver.MaxID, ShouldEqual, 0)
			So(metricsObserver.EarliestReceived, ShouldEqual, time.Time{})
			So(metricsObserver.LatestReceived, ShouldEqual, time.Time{})
			So(metricsObserver.AllSenders, ShouldResemble, []string(nil))
			So(len(metricsObserver.UniqueMentions), ShouldEqual, 0)
			So(metricsObserver.MentionsSeen, ShouldEqual, 0)
		})
		Convey("updating", func() {
			now := time.Now()
			mention1 := Mention{
				WMID:       10,
				WMReceived: now,
				WMSource:   "https://mention1.net",
			}

			mention2 := Mention{
				WMID:       20,
				WMReceived: now.Add(24 * time.Hour),
				WMSource:   "https://mention2.net",
			}
			Convey("updating once should set all of the values", func() {

				metricsObserver.Update(mention1)
				So(metricsObserver, ShouldNotBeNil)
				So(metricsObserver.MaxID, ShouldEqual, mention1.WMID)
				So(metricsObserver.EarliestReceived, ShouldEqual, mention1.WMReceived)
				So(metricsObserver.LatestReceived, ShouldEqual, mention1.WMReceived)
				So(metricsObserver.AllSenders, ShouldResemble, []string{mention1.WMSource})
				So(len(metricsObserver.UniqueMentions), ShouldEqual, 1)
				So(metricsObserver.MentionsSeen, ShouldEqual, 1)
			})

			Convey("updating twice should update the values as expected", func() {
				metricsObserver.Update(mention1)
				metricsObserver.Update(mention2)

				So(metricsObserver, ShouldNotBeNil)
				So(metricsObserver.MaxID, ShouldEqual, mention2.WMID)
				So(metricsObserver.EarliestReceived, ShouldEqual, mention1.WMReceived)
				So(metricsObserver.LatestReceived, ShouldEqual, mention2.WMReceived)
				So(metricsObserver.AllSenders, ShouldResemble, []string{mention1.WMSource, mention2.WMSource})
				So(len(metricsObserver.UniqueMentions), ShouldEqual, 2)
				So(metricsObserver.MentionsSeen, ShouldEqual, 2)
			})

			Convey("updating the same mention again should update counts but not values or observations", func() {
				metricsObserver.Update(mention1)
				metricsObserver.Update(mention2)
				metricsObserver.Update(mention2)

				So(metricsObserver, ShouldNotBeNil)
				So(metricsObserver.MaxID, ShouldEqual, mention2.WMID)
				So(metricsObserver.EarliestReceived, ShouldEqual, mention1.WMReceived)
				So(metricsObserver.LatestReceived, ShouldEqual, mention2.WMReceived)
				So(metricsObserver.AllSenders, ShouldResemble, []string{mention1.WMSource, mention2.WMSource})
				So(len(metricsObserver.UniqueMentions), ShouldEqual, 2)
				So(metricsObserver.MentionsSeen, ShouldEqual, 3)
			})
		})
	})
}
