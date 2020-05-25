package cli

import (
	"gunviolencearchive-client/package/gvaclient"
	"io"
	"os"
	"time"

	"github.com/carmo-evan/strtotime"
	"github.com/gocarina/gocsv"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

type Client interface {
	Query(opts ...gvaclient.QueryOption) (gvaclient.QueryID, error)
	GetRecords(gvaclient.QueryID) (io.ReadCloser, error)
	GetIncidentCoordinates(gvaclient.QueryID) ([]gvaclient.IncidentCoordinates, error)
}

var (
	rootURL  string
	inState  string
	fromDate string
	toDate   string
	queryCmd = &cobra.Command{
		Use:   "query",
		Short: "Queries the Gun Violence Archive",
		Run: func(cmd *cobra.Command, args []string) {
			queryOptions := []gvaclient.QueryOption{
				gvaclient.WithAllCriteriaMatching(true),
				gvaclient.WithResultType(gvaclient.Incidents),
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

			incidents, coordinates, err := getIncidentResults(rootURL, queryOptions)
			if err != nil {
				log.Fatal("failed to get results", "err", err)
			}

			results := mergeIncidentResults(incidents, coordinates)

			err = gocsv.Marshal(results, os.Stdout)
			if err != nil {
				log.Fatal("failed to write results to standard out", "err", err)
			}
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

func getIncidentResults(root string, queryOptions []gvaclient.QueryOption) ([]gvaclient.IncidentRecord, []gvaclient.IncidentCoordinates, error) {
	var c Client
	c, err := gvaclient.NewQueryClient(root, log)
	if err != nil {
		return nil, nil, errors.Wrap(err, "failed to create client")
	}

	queryID, err := c.Query(queryOptions...)
	if err != nil {
		return nil, nil, errors.Wrap(err, "failed to start query")
	}

	reader, err := c.GetRecords(queryID)
	if err != nil {
		return nil, nil, errors.Wrap(err, "failed to get records")
	}
	defer reader.Close()

	incidents, err := gvaclient.ParseIncidentRecords(reader)
	if err != nil {
		return nil, nil, errors.Wrap(err, "failed to parse incidents")
	}

	coordinates, err := c.GetIncidentCoordinates(queryID)
	if err != nil {
		return nil, nil, errors.Wrap(err, "failed to get coordinates")
	}

	return incidents, coordinates, nil
}

type incidentResult struct {
	IncidentID   string
	IncidentDate time.Time
	State        string
	CityOrCounty string
	Address      string
	Killed       int
	Injured      int
	Operations   string
	Longitude    *float64
	Latitude     *float64
}

func mergeIncidentResults(incidents []gvaclient.IncidentRecord, coordinates []gvaclient.IncidentCoordinates) []incidentResult {
	coordMap := map[string]gvaclient.IncidentCoordinates{}
	for _, coord := range coordinates {
		coordMap[coord.IncidentID] = coord
	}

	results := []incidentResult{}
	for _, incident := range incidents {
		incidentDateSeconds, _ := strtotime.Parse(incident.IncidentDate, time.Now().Unix())
		result := incidentResult{
			IncidentID:   incident.IncidentID,
			IncidentDate: time.Unix(incidentDateSeconds, 0).UTC(),
			State:        incident.State,
			CityOrCounty: incident.CityOrCounty,
			Address:      incident.Address,
			Killed:       incident.Killed,
			Injured:      incident.Injured,
			Operations:   incident.Operations,
		}
		coord, ok := coordMap[incident.IncidentID]
		if ok {
			result.Latitude = &coord.Latitude
			result.Longitude = &coord.Longitude
		}

		results = append(results, result)
	}

	return results
}
