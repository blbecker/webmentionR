package fetch

import (
	"context"
	"fmt"
	"github.com/blbecker/webmentionR/state"
	"github.com/blbecker/webmentionR/webmention"
	"github.com/charmbracelet/log"
	"github.com/urfave/cli/v2"
	"sync"
)

var FetchFunc = webmention.DoFetch
var PersistFunc = webmention.DoPersist

var Command = cli.Command{
	Name:    "fetch",
	Aliases: []string{"f"},
	Usage:   "fetch webmentions from the endpoint",
	Action:  fetchAction,
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:     "token",
			Aliases:  []string{"t"},
			Required: true,
		},
		&cli.StringFlag{
			Name:     "domain",
			Aliases:  []string{"d"},
			Required: true,
		},
		&cli.StringFlag{
			Name:     "destination",
			Aliases:  []string{"D"},
			Required: true,
		},
		&cli.StringFlag{
			Name:    "state-file",
			Aliases: []string{"s"},
			Value:   "./fetch.webmentions.state",
		},
		&cli.IntFlag{
			Name:  "page-size",
			Value: 10,
		},
	},
}

type Context struct {
	Domain   string
	Token    string
	State    *state.State
	PageSize int
}

// NewFetchContext constructs a fetch context representative of the passed cli.Context.
func NewFetchContext(cliContext *cli.Context) (*Context, error) {
	stateFilePath := cliContext.String("fetchState-file")
	fetchState, err := state.ReadState(stateFilePath)
	if err != nil {
		return nil, fmt.Errorf("cannot read fetchState file: %w", err)
	}

	fetchContext := Context{
		Domain:   cliContext.String("domain"),
		Token:    cliContext.String("token"),
		PageSize: cliContext.Int("page-size"),
		State:    fetchState,
	}
	return &fetchContext, err
}

// fetchAction is an adapter for doFetch implementing cli.ActionFunc for use in a cli.Command
func fetchAction(context *cli.Context) error {
	fetchContext, err := NewFetchContext(context)
	if err != nil {
		return fmt.Errorf("cannot create fetch context: %w", err)
	}
	return doFetch(fetchContext)
}

// doFetch implements the webmention retrieval and persistence functionality.
func doFetch(fetchContext *Context) error {
	client := webmention.Client{
		Domain:   fetchContext.Domain,
		Token:    fetchContext.Token,
		SinceID:  fetchContext.State.SinceID,
		PageSize: fetchContext.PageSize,
	}
	mentionChan := make(chan webmention.Mention, 10)

	observer := webmention.MetricsObserver{}
	var fetchWorker webmention.FetchWorker
	fetchWorker.AddObserver(&observer)

	var fetchErr error
	go func() {
		fetchErr = FetchFunc(context.TODO(), client, mentionChan, &fetchWorker)
	}()

	if fetchErr != nil {
		return fmt.Errorf("error fetching webmentions: %v", fetchErr)
	}

	mentionsByTarget := map[string][]webmention.Mention{}
	for thisWebmention := range mentionChan {
		mentionsByTarget[thisWebmention.WMTarget] = append(mentionsByTarget[thisWebmention.WMTarget], thisWebmention)
	}

	persistenceWorker := webmention.PersistenceWorker{}
	var wg sync.WaitGroup
	var persistenceErr error
	for _, mentions := range mentionsByTarget {
		wg.Add(1)
		go func() {
			err := PersistFunc(mentions, &wg, &persistenceWorker)

			// If any errors occur, retain it for bubble up
			if persistenceErr == nil {
				persistenceErr = err
			}
		}()
	}

	wg.Wait()
	if persistenceErr != nil {
		return fmt.Errorf("error persisting webmentions: %v", persistenceErr)
	}

	log.Info("Collected metrics: ", "metrics", observer.GetMetrics())
	return nil
}
