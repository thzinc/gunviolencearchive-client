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

	"github.com/gocarina/gocsv"
	"github.com/google/uuid"
	"github.com/pkg/errors"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
)

type IncidentRecord struct {
	IncidentID   string `csv:"Incident ID"`
	IncidentDate string `csv:"Incident Date"`
	State        string `csv:"State"`
	CityOrCounty string `csv:"City Or County"`
	Address      string `csv:"Address"`
	Killed       int    `csv:"# Killed"`
	Injured      int    `csv:"# Injured"`
	Operations   string `csv:"Operations"`
}

type IncidentCoordinates struct {
	IncidentID string
	Longitude  float64
	Latitude   float64
}

type progress struct {
	Status     bool   `json:"status"`
	Percentage int    `json:"percentage,string"`
	Message    string `json:"message"`
}

type QueryClient struct {
	client  *http.Client
	rootURL string
	log     *zap.SugaredLogger
}

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

func (qc *QueryClient) Query(opts ...QueryOption) (QueryID, error) {
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

	qc.log.Debug("registering query",
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

func (qc *QueryClient) GetRecords(queryID QueryID) (io.ReadCloser, error) {
	exportCsvURL, err := url.Parse(qc.rootURL)
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse root URL")
	}

	exportCsvURL.Path = path.Join(exportCsvURL.Path, "query", string(queryID), "export-csv")
	qc.log.Debug("kicking off export to CSV")
	resp, err := qc.client.Get(exportCsvURL.String())
	if err != nil {
		return nil, errors.Wrap(err, "failed to export query")
	}
	defer resp.Body.Close()

	batchURL, err := url.Parse(resp.Header.Get("Location"))
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse batch URL")
	}

	qc.log.Debug("starting batch process")
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
			qc.log.Debug("requesting progress update")
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

			qc.log.Debug("received update on progress", "progress", prog)

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
	qc.log.Debug("finishing batch process")
	resp, err = qc.client.Get(batchURL.String())
	if err != nil {
		return nil, errors.Wrap(err, "failed to finish export batch")
	}
	// defer resp.Body.Close()

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
	qc.log.Debug("downloading CSV result")
	resp, err = qc.client.Get(downloadURL.String())
	if err != nil {
		return nil, errors.Wrap(err, "failed to finish export batch")
	}
	return resp.Body, errors.Wrap(err, "failed to read CSV stream")
}

func (qc *QueryClient) GetIncidentCoordinates(queryID QueryID) ([]IncidentCoordinates, error) {
	mapURL, err := url.Parse(qc.rootURL)
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse root URL")
	}

	mapURL.Path = path.Join(mapURL.Path, "query", string(queryID), "map")
	qc.log.Debug("getting map")
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
	coordinates := []IncidentCoordinates{}
	for _, line := range strings.Split(rawCoordinates, "||") {
		parts := strings.Split(line, "|")
		lonStr, latStr, id := parts[0], parts[1], parts[2]

		lon, err := strconv.ParseFloat(lonStr, 64)
		if err != nil {
			return nil, errors.Wrap(err, fmt.Sprintf("failed to parse longitude for incident %s", id))
		}

		lat, err := strconv.ParseFloat(latStr, 64)
		if err != nil {
			return nil, errors.Wrap(err, fmt.Sprintf("failed to parse latitude for incident %s", id))
		}

		coordinates = append(coordinates, IncidentCoordinates{
			IncidentID: id,
			Longitude:  lon,
			Latitude:   lat,
		})
	}

	return coordinates, nil
}

func ParseIncidentRecords(readCloser io.Reader) ([]IncidentRecord, error) {
	var incidents []IncidentRecord
	err := gocsv.Unmarshal(readCloser, &incidents)
	if err != nil {
		return nil, errors.Wrap(err, "failed to unmarshal incidents")
	}

	return incidents, nil
}
