package gvaclient

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"path"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/carmo-evan/strtotime"
	"github.com/gocarina/gocsv"
	"github.com/google/uuid"
	"github.com/pkg/errors"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
)

// Incident is a record of an incident of gun violence
type Incident struct {
	IncidentID   string    `csv:"Incident ID"`
	IncidentDate time.Time `csv:"Incident Date"`
	State        string    `csv:"State"`
	CityOrCounty string    `csv:"City Or County"`
	Address      string    `csv:"Address"`
	Killed       int       `csv:"# Killed"`
	Injured      int       `csv:"# Injured"`
	Operations   string    `csv:"Operations"`
	Longitude    *float64
	Latitude     *float64
}

// QueryClient is a client to query the Gun Violence Archive
type QueryClient struct {
	client  *http.Client
	rootURL string
	log     *zap.SugaredLogger
}

// NewQueryClient creates a new QueryClient
func NewQueryClient(rootURL string, log *zap.SugaredLogger) (*QueryClient, error) {
	jar, err := cookiejar.New(&cookiejar.Options{})
	if err != nil {
		return nil, errors.Wrap(err, "failed to init cookie jar")
	}

	client := &http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
		Jar: jar,
	}
	return &QueryClient{
		client,
		rootURL,
		log,
	}, nil
}

// QueryIncidents queries incidents from the Gun Violence Archive
func (qc *QueryClient) QueryIncidents(opts ...QueryOption) ([]Incident, error) {
	queryID, err := qc.query(append(opts, withResultType(resultsTypeIncidents))...)
	if err != nil {
		return nil, errors.Wrap(err, "failed to start query")
	}

	reader, err := qc.getRecords(queryID)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get records")
	}
	defer reader.Close()

	incidents, err := parseIncidentRecords(reader)
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse incidents")
	}

	coordinates, err := qc.getIncidentCoordinates(queryID)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get coordinates")
	}

	return mergeIncidentResults(incidents, coordinates), nil
}

type incidentRecord struct {
	IncidentID   string `csv:"Incident ID"`
	IncidentDate string `csv:"Incident Date"`
	State        string `csv:"State"`
	CityOrCounty string `csv:"City Or County"`
	Address      string `csv:"Address"`
	Killed       int    `csv:"# Killed"`
	Injured      int    `csv:"# Injured"`
	Operations   string `csv:"Operations"`
}

type incidentCoordinates struct {
	IncidentID string
	Longitude  float64
	Latitude   float64
}

type progress struct {
	Status     bool   `json:"status"`
	Percentage int    `json:"percentage,string"`
	Message    string `json:"message"`
}

// query registers a new query and returns a unique identifier
func (qc *QueryClient) query(opts ...QueryOption) (QueryID, error) {
	queryID := uuid.New().String()
	options := &QueryOptions{
		queryData: url.Values{
			"query[query_id]": []string{queryID},
			"form_id":         []string{"gva_entry_query"},
			"op":              []string{"Search"},
		},
	}
	for _, f := range opts {
		f(options)
	}

	queryURL, err := url.Parse(qc.rootURL)
	if err != nil {
		return "", errors.Wrap(err, "failed to parse root URL")
	}

	queryURL.Path = path.Join(queryURL.Path, "query")

	qc.log.Debugw("registering query",
		"queryID", queryID,
		"form", options.queryData,
	)
	resp, err := qc.client.PostForm(queryURL.String(), options.queryData)
	if err != nil {
		return "", errors.Wrap(err, "failed to post query")
	}

	defer resp.Body.Close()
	if resp.StatusCode != 302 {
		return "", errors.Wrap(err, "received unexpected result from query")
	}

	return QueryID(queryID), nil
}

// getRecords returns a reader of CSV data for a given QueryID
func (qc *QueryClient) getRecords(queryID QueryID) (io.ReadCloser, error) {
	exportCsvURL, err := url.Parse(qc.rootURL)
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse root URL")
	}

	exportCsvURL.Path = path.Join(exportCsvURL.Path, "query", string(queryID), "export-csv")
	qc.log.Debugw("kicking off export to CSV")
	resp, err := qc.client.Get(exportCsvURL.String())
	if err != nil {
		return nil, errors.Wrap(err, "failed to export query")
	}
	defer resp.Body.Close()

	batchURL, err := url.Parse(resp.Header.Get("Location"))
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse batch URL")
	}

	qc.log.Debugw("starting batch process")
	resp, err = qc.client.Get(batchURL.String())
	if err != nil {
		return nil, errors.Wrap(err, "failed to start export batch")
	}
	defer resp.Body.Close()

	query := batchURL.Query()
	query.Set("op", "do")
	batchURL.RawQuery = query.Encode()
	group, _ := errgroup.WithContext(context.Background())
	group.Go(func() error {
		// TODO: do something with a context here
		for {
			qc.log.Debugw("requesting progress update")
			resp, err = qc.client.PostForm(batchURL.String(), nil)
			if err != nil {
				return err
			}
			defer resp.Body.Close()
			dec := json.NewDecoder(resp.Body)

			var prog progress
			err = dec.Decode(&prog)
			if err != nil {
				return err
			}

			qc.log.Debugw("received update on progress", "progress", prog)

			if prog.Percentage == 100 {
				return nil
			}
		}
	})

	err = group.Wait()
	if err != nil {
		return nil, errors.Wrap(err, "failed to get status on export batch")
	}

	query = batchURL.Query()
	query.Set("op", "finished")
	batchURL.RawQuery = query.Encode()
	qc.log.Debugw("finishing batch process")
	resp, err = qc.client.Get(batchURL.String())
	if err != nil {
		return nil, errors.Wrap(err, "failed to finish export batch")
	}

	finalURL, err := url.Parse(resp.Header.Get("Location"))
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse export result URL")
	}

	downloadURL, err := url.Parse(qc.rootURL)
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse root URL")
	}

	downloadURL.Path = path.Join(downloadURL.Path, "export-finished", "download")
	downloadURL.RawQuery = finalURL.RawQuery
	qc.log.Debugw("downloading CSV result")
	resp, err = qc.client.Get(downloadURL.String())
	if err != nil {
		return nil, errors.Wrap(err, "failed to finish export batch")
	}
	return resp.Body, errors.Wrap(err, "failed to read CSV stream")
}

// getIncidentCoordinates gets the lon/lat coordinates of incidents for a given QueryID
func (qc *QueryClient) getIncidentCoordinates(queryID QueryID) ([]incidentCoordinates, error) {
	mapURL, err := url.Parse(qc.rootURL)
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse root URL")
	}

	mapURL.Path = path.Join(mapURL.Path, "query", string(queryID), "map")
	qc.log.Debugw("getting map")
	resp, err := qc.client.Get(mapURL.String())
	if err != nil {
		return nil, errors.Wrap(err, "failed to get map")
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, errors.Wrap(err, "failed to read body")
	}
	re := regexp.MustCompile(fmt.Sprintf(`"interactive-map-%s":"([\d+\-\.|]+)"`, queryID))
	result := re.FindAllStringSubmatch(string(body), -1)
	rawCoordinates := result[0][1]
	coordinates := []incidentCoordinates{}
	for _, line := range strings.Split(rawCoordinates, "||") {
		parts := strings.Split(line, "|")
		latStr, lonStr, id := parts[0], parts[1], parts[2]

		lon, err := strconv.ParseFloat(lonStr, 64)
		if err != nil {
			return nil, errors.Wrap(err, fmt.Sprintf("failed to parse longitude for incident %s", id))
		}

		lat, err := strconv.ParseFloat(latStr, 64)
		if err != nil {
			return nil, errors.Wrap(err, fmt.Sprintf("failed to parse latitude for incident %s", id))
		}

		coordinates = append(coordinates, incidentCoordinates{
			IncidentID: id,
			Longitude:  lon,
			Latitude:   lat,
		})
	}

	return coordinates, nil
}

// parseIncidentRecords parses CSV results representing incidents into an incidentRecord
func parseIncidentRecords(readCloser io.Reader) ([]incidentRecord, error) {
	var incidents []incidentRecord
	err := gocsv.Unmarshal(readCloser, &incidents)
	if err != nil {
		return nil, errors.Wrap(err, "failed to unmarshal incidents")
	}

	return incidents, nil
}

// mergeIncidentResults merges incidents and coordinates on IncidentID
func mergeIncidentResults(incidents []incidentRecord, coordinates []incidentCoordinates) []Incident {
	coordMap := map[string]incidentCoordinates{}
	for _, coord := range coordinates {
		coordMap[coord.IncidentID] = coord
	}

	results := []Incident{}
	for _, incident := range incidents {
		incidentDateSeconds, _ := strtotime.Parse(incident.IncidentDate, time.Now().Unix())
		result := Incident{
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
