package webmention

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-faker/faker/v4"

	. "github.com/smartystreets/goconvey/convey"
)

func TestClient_GetMentions(t *testing.T) {
	Convey("Given a Client with a configured domain, token, and parameters", t, func() {
		domain := "example.com"
		token := "test-token"
		sinceId := 100
		pagesize := 10
		client := &Client{
			Domain:   domain,
			Token:    token,
			SinceID:  sinceId,
			PageSize: pagesize,
		}

		Convey("When GetMentions is called", func() {
			mockResponse := Response{}
			err := faker.FakeData(&mockResponse)
			if err != nil {
				fmt.Println(err)
			}
			mockResponseBody, _ := json.Marshal(mockResponse)

			// Set up mockResponse mock server to handle requests
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

				w.WriteHeader(http.StatusOK)
				_, _ = w.Write(mockResponseBody)
			}))
			defer server.Close()

			// Override BaseUrl to point to the mock server
			BaseUrl = server.URL

			result, err := client.GetMentions()

			Convey("Then the response should match the expected output", func() {
				So(err, ShouldBeNil)
				So(result.Children, ShouldNotBeEmpty)
			})

		})

		Convey("When GetMentions encounters an error", func() {
			// Simulate an error by closing the mock server early
			mockResponse := Response{}
			err := faker.FakeData(&mockResponse)
			if err != nil {
				fmt.Println(err)
			}
			mockResponseBody, _ := json.Marshal(mockResponse)

			// Set up mockResponse mock server to handle requests
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

				w.WriteHeader(http.StatusInternalServerError)
				_, _ = w.Write(mockResponseBody)
			}))
			defer server.Close()

			BaseUrl = server.URL
			result, err := client.GetMentions()

			So(err, ShouldNotBeNil)
			So(result, ShouldBeNil)
		})
	})
}
