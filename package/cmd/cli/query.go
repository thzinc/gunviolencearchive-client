package cli

import (
	"os"
	"time"

	"github.com/thzinc/gunviolencearchive-client/package/gvaclient"

	"github.com/carmo-evan/strtotime"
	"github.com/gocarina/gocsv"
	"github.com/spf13/cobra"
)

var (
	rootURL                  string
	inState                  string
	fromDate                 string
	toDate                   string
	participantAge           int
	participantAgeComparator comparator = comparator{value: gvaclient.IsEqualTo}
	queryOptions             []gvaclient.QueryOption
	queryCmd                 = &cobra.Command{
		Use:   "query",
		Short: "Queries the Gun Violence Archive",
		Long:  "Queries and prints comma-separated values. The Gun Violence Archive places opaque limits on the number of records returned for large queries, so your results may be artificially truncated for large queries.",
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
	queryCmd.PersistentFlags().IntVar(&participantAge, "participant-age", -1, "Specify the age in years of any participant to evaluate")
	queryCmd.PersistentFlags().Var(&participantAgeComparator, "participant-age-comparator", "Specify how to compare the participant age")
	queryCmd.AddCommand(incidentCmd)
	RootCmd.AddCommand(queryCmd)
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

	if participantAge > -1 {
		queryOptions = append(queryOptions, gvaclient.WithParticipantsAge(
			participantAgeComparator.Value(),
			participantAge,
		))
	}
}
