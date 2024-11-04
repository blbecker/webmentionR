package fetch

import (
	c "context"
	"flag"
	"github.com/blbecker/webmentionR/webmention"
	. "github.com/smartystreets/goconvey/convey"
	"github.com/urfave/cli/v2"
	"sync"
	"testing"
)

func Test_FetchAction(t *testing.T) {
	Convey("Given a DoFetch with a mock client", t, func() {

	})
}

func Test_NewFetchContext(t *testing.T) {
	Convey("Constructing a new fetchContext", t, func() {
		Convey("returns a representative context when successful", func() {
			flags := flag.NewFlagSet("test", flag.ContinueOnError)
			context := cli.NewContext(nil, flags, nil)

			fetchContext, err := NewFetchContext(context)
			So(err, ShouldBeNil)
			So(fetchContext, ShouldNotBeNil)
		})
	})
}

type MockFetchWorker struct {
	WantedMentions []webmention.Mention
	WantedErr      error
}

type MockPersistWorker struct {
	ReceivedMentions  []webmention.Mention
	ReceivedWaitGroup *sync.WaitGroup
	WantedErr         error
}

func Test_doFetch(t *testing.T) {
	Convey("Fetching with reasonable input doesn't produce an error", t, func() {
		flags := flag.NewFlagSet("test", flag.ContinueOnError)
		cliContext := cli.NewContext(nil, flags, nil)
		fetchContext, err := NewFetchContext(cliContext)
		fw := MockFetchWorker{
			WantedMentions: []webmention.Mention{
				{
					WMID: 1,
				},
			},
			WantedErr: nil,
		}
		pw := MockPersistWorker{
			ReceivedMentions:  nil,
			ReceivedWaitGroup: nil,
			WantedErr:         nil,
		}
		FetchFunc = func(ctx c.Context, client webmention.Client, mentionChan chan webmention.Mention, worker webmention.Fetchable) error {
			defer close(mentionChan)
			for _, mention := range fw.WantedMentions {
				mentionChan <- mention
			}
			return fw.WantedErr
		}
		PersistFunc = func(fetchedMentions []webmention.Mention, s *sync.WaitGroup, persistable webmention.Persistable) error {
			defer s.Done()
			pw.ReceivedMentions = fetchedMentions
			pw.ReceivedWaitGroup = s
			return pw.WantedErr
		}
		So(err, ShouldBeNil)
		err = doFetch(fetchContext)
		So(err, ShouldBeNil)
		So(len(fw.WantedMentions), ShouldEqual, len(pw.ReceivedMentions))
		for _, mention := range fw.WantedMentions {
			So(pw.ReceivedMentions, ShouldContain, mention)
		}
	})
}

func Test_fetchAction(t *testing.T) {
	Convey("Given a DoFetch with a mock client", t, func() {

	})
}
