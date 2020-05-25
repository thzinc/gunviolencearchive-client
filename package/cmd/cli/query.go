package cli

import (
	"gunviolencearchive-client/package/gvaclient"
	"os"
	"time"

	"github.com/carmo-evan/strtotime"
	"github.com/gocarina/gocsv"
	"github.com/spf13/cobra"
)

var (
	rootURL      string
	inState      string
	fromDate     string
	toDate       string
	queryOptions []gvaclient.QueryOption
	queryCmd     = &cobra.Command{
		Use:   "query",
		Short: "Queries the Gun Violence Archive",
	}
	incidentCmd = &cobra.Command{
		Use:    "incidents",
		Short:  "Returns results as incidences of gun violence",
		PreRun: populateQueryOptions,
		Run: func(cmd *cobra.Command, args []string) {
			client, err := gvaclient.NewQueryClient(rootURL, log)
			if err != nil {
				log.Fatal("failed to create client", "err", err)
			}

			results, err := client.QueryIncidents(queryOptions...)
			if err != nil {
				log.Fatal("failed to get results", "err", err)
			}

			err = gocsv.Marshal(results, os.Stdout)
			if err != nil {
				log.Fatal("failed to write results to standard out", "err", err)
			}
		},
	}
)

func init() {
	queryCmd.PersistentFlags().StringVarP(&rootURL, "url", "u", "https://www.gunviolencearchive.org", "Specify a URL to use as the root for accessing the API")
	queryCmd.PersistentFlags().StringVar(&inState, "in-state", "", "Specify a state to filter results by")
	queryCmd.PersistentFlags().StringVar(&fromDate, "from", "", "Specify the start date to filter results by (inclusive)")
	queryCmd.PersistentFlags().StringVar(&toDate, "to", "", "Specify the end date to filter results by (inclusive)")
	queryCmd.AddCommand(incidentCmd)
	rootCmd.AddCommand(queryCmd)
}

func populateQueryOptions(cmd *cobra.Command, args []string) {
	queryOptions = []gvaclient.QueryOption{
		gvaclient.WithAllCriteriaMatching(true),
	}

	if inState != "" {
		queryOptions = append(queryOptions, gvaclient.WithIncidentLocation(inState, "", ""))
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
		queryOptions = append(queryOptions, gvaclient.WithIncidentDate(
			gvaclient.IsIn,
			time.Unix(from, 0).UTC(),
			time.Unix(to, 0).UTC(),
		))
	}
}
