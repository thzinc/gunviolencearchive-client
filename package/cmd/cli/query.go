package cli

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"path"
	"time"

	"github.com/carmo-evan/strtotime"
	"github.com/google/uuid"
	"github.com/spf13/cobra"
	"github.com/syncromatics/go-kit/log"
)

var (
	rootURL  string
	inState  string
	fromDate string
	toDate   string
	queryCmd = &cobra.Command{
		Use:   "query",
		Short: "Queries the Gun Violence Archive",
		Run: func(cmd *cobra.Command, args []string) {
			queryURL, err := url.Parse(rootURL)
			if err != nil {
				log.Fatal("failed to parse url",
					"err", err,
					"url", rootURL,
				)
			}
			queryURL.Path = path.Join(queryURL.Path, "query")
			queryID := uuid.New().String()
			data := url.Values{
				"query[base_group][base_group_select]": []string{"And"}, // TODO: make a flag with options for "And" and "Or"
				"query[query_id]":                      []string{queryID},
				"query[results_type][select]":          []string{"incidents"}, // TODO: make a flag with options for "incidents" and "participants"
				"form_id":                              []string{"gva_entry_query"},
				"op":                                   []string{"Search"},
			}
			if inState != "" {
				criterionID := uuid.New().String()
				data.Add(fmt.Sprintf("query[filters][%s][type]", criterionID), "IncidentLocation")
				data.Add(fmt.Sprintf("query[filters][%s][outer_filter][weight]", criterionID), "0.001")
				data.Add(fmt.Sprintf("query[filters][%s][outer_filter][comparator]", criterionID), "is in")
				data.Add(fmt.Sprintf("query[filters][%s][outer_filter][filter][field][state][select]", criterionID), inState)
			}
			if fromDate != "" && toDate != "" {
				from, err := strtotime.Parse(fromDate, time.Now().Unix())
				if err != nil {
					log.Fatal("failed to parse from date",
						"err", err,
						"from", fromDate,
					)
				}
				to, err := strtotime.Parse(toDate, time.Now().Unix())
				if err != nil {
					log.Fatal("failed to parse to date",
						"err", err,
						"to", toDate,
					)
				}
				criterionID := uuid.New().String()
				data.Add(fmt.Sprintf("query[filters][%s][type]", criterionID), "IncidentDate")
				data.Add(fmt.Sprintf("query[filters][%s][outer_filter][weight]", criterionID), "0.001")
				data.Add(fmt.Sprintf("query[filters][%s][outer_filter][comparator]", criterionID), "is in")
				data.Add(fmt.Sprintf("query[filters][%s][outer_filter][filter][field][date-from]", criterionID), time.Unix(from, 0).UTC().Format("01/02/2006"))
				data.Add(fmt.Sprintf("query[filters][%s][outer_filter][filter][field][date-to]", criterionID), time.Unix(to, 0).UTC().Format("01/02/2006"))
			}
			client := &http.Client{
				CheckRedirect: func(req *http.Request, via []*http.Request) error {
					return http.ErrUseLastResponse
				},
			}
			log.Info("Sending this data")
			for k, v := range data {
				log.Info(k, "values", v)
			}
			resp, err := client.PostForm(queryURL.String(), data)
			if err != nil {
				log.Fatal("failed to query",
					"err", err,
					"url", queryURL.String(),
				)
			}
			defer resp.Body.Close()
			if resp.StatusCode != 302 {
				log.Fatal("failed to query",
					"url", queryURL.String(),
					"status", resp.StatusCode,
				)
			}

			// TODO: get list of results from query
			exportCsvURL, err := url.Parse(rootURL)
			if err != nil {
				log.Fatal("failed to parse url",
					"err", err,
					"url", rootURL,
				)
			}
			exportCsvURL.Path = path.Join(exportCsvURL.Path, "query", queryID, "export-csv")
			resp, err = client.Get(exportCsvURL.String())
			if err != nil {
				log.Fatal("failed to export query",
					"err", err,
					"url", exportCsvURL.String(),
				)
			}
			defer resp.Body.Close()
			if resp.StatusCode != 302 {
				log.Fatal("failed to export query",
					"url", queryURL.String(),
					"status", resp.StatusCode,
				)
			}
			batchURL, err := url.Parse(resp.Header.Get("Location"))
			if err != nil {
				log.Fatal("failed to parse export query batch url",
					"err", err,
					"url", resp.Header.Get("Location"),
				)
			}

			log.Debug("got location",
				"status", resp.StatusCode,
				"url", batchURL.String(),
				"location", resp.Header.Get("Location"),
			)

			resp, err = client.Get(batchURL.String())
			if err != nil {
				log.Fatal("failed to start export query batch",
					"err", err,
					"url", batchURL.String(),
				)
			}
			defer resp.Body.Close()
			log.Debug("started",
				"status", resp.StatusCode,
				"url", batchURL.String(),
				"location", resp.Header.Get("Location"),
			)

			referer := batchURL.String()
			query := batchURL.Query()
			query.Set("op", "do")
			batchURL.RawQuery = query.Encode()
			progressRequest, err := http.NewRequest("POST", batchURL.String(), nil)
			if err != nil {
				log.Fatal("failed to create request for progress",
					"err", err,
					"url", batchURL.String(),
				)
			}
			progressRequest.Header.Add("authority", batchURL.Hostname())
			progressRequest.Header.Add("accept", "application/json, text/javascript, */*; q=0.01")
			progressRequest.Header.Add("x-requested-with", "XMLHttpRequest")
			progressRequest.Header.Add("origin", rootURL) // HACK
			progressRequest.Header.Add("sec-fetch-site", "same-origin")
			progressRequest.Header.Add("sec-fetch-mode", "cors")
			progressRequest.Header.Add("sec-fetch-dest", "empty")
			progressRequest.Header.Add("referer", referer)
			progressRequest.Header.Add("accept-language", "en-US,en;q=0.9")
			resp, err = client.Do(progressRequest)
			if err != nil {
				log.Fatal("failed to get progress for export query batch",
					"err", err,
					"url", batchURL.String(),
				)
			}
			defer resp.Body.Close()
			xxx, err := ioutil.ReadAll(resp.Body)
			if err != nil {
				log.Fatal("failed to read body",
					"err", err,
				)
			}
			log.Debug(string(xxx),
				"status", resp.StatusCode,
				"url", batchURL.String(),
				"location", resp.Header.Get("Location"),
			)

			// https://www.gunviolencearchive.org/export-finished/download?uuid=7e5e6083-1dad-4cbf-bec0-a3cd39835bc6&filename=public%3A//export-b7983778-00bf-4cd6-9d3d-b8d6c5d9ffd7.csv

		},
	}
)

func init() {
	queryCmd.Flags().StringVarP(&rootURL, "url", "u", "https://www.gunviolencearchive.org", "Specify a URL to use as the root for accessing the API")
	queryCmd.Flags().StringVar(&inState, "in-state", "", "Specify a state to filter results by")
	queryCmd.Flags().StringVar(&fromDate, "from", "", "Specify the start date to filter results by (inclusive)")
	queryCmd.Flags().StringVar(&toDate, "to", "", "Specify the end date to filter results by (inclusive)")
	rootCmd.AddCommand(queryCmd)
}
