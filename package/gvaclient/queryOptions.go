package gvaclient

import (
	"fmt"
	"net/url"
	"time"

	"github.com/google/uuid"
)

// QueryOptions holds the form data to post in a query
type QueryOptions struct {
	queryData url.Values
}

// QueryID is a unique identifier of a query
type QueryID string

// QueryOption is a fluent crtierion for a query
type QueryOption func(*QueryOptions)

// WithAllCriteriaMatching indicates whether ALL crtieria must match or whether ANY criteria must match
func WithAllCriteriaMatching(all bool) QueryOption {
	return func(qo *QueryOptions) {
		value := "Or"
		if all {
			value = "And"
		}
		qo.queryData.Add("query[base_group][base_group_select]", value)
	}
}

// resultsType indicates the type of results to return
type resultsType string

const (
	resultsTypeIncidents   resultsType = "incidents"
	resultsTypeParticiants resultsType = "participants"
)

func withResultType(resultType resultsType) QueryOption {
	return func(qo *QueryOptions) {
		qo.queryData.Add("query[results_type][select]", string(resultType))
	}
}

// WithIncidentLocation indicates the state, city, and/or county of an incident
func WithIncidentLocation(state, city, county string) QueryOption {
	return func(qo *QueryOptions) {
		criterionID := uuid.New().String()
		qo.queryData.Add(fmt.Sprintf("query[filters][%s][type]", criterionID), "IncidentLocation")
		qo.queryData.Add(fmt.Sprintf("query[filters][%s][outer_filter][weight]", criterionID), "0.001")
		qo.queryData.Add(fmt.Sprintf("query[filters][%s][outer_filter][comparator]", criterionID), "is in")
		if state != "" {
			qo.queryData.Add(fmt.Sprintf("query[filters][%s][outer_filter][filter][field][state][select]", criterionID), state)
		}
		if city != "" {
			qo.queryData.Add(fmt.Sprintf("query[filters][%s][outer_filter][filter][field][city_county][city][textfield]", criterionID), city)
		}
		if county != "" {
			qo.queryData.Add(fmt.Sprintf("query[filters][%s][outer_filter][filter][field][city_county][county][textfield]", criterionID), county)
		}
	}
}

// IncidentDateComparator indicates how a date range should be specified
type IncidentDateComparator string

const (
	// IsIn indicates the date must be between from and to
	IsIn IncidentDateComparator = "is in"
	// IsNotIn indicates the date must not be between from and to
	IsNotIn IncidentDateComparator = "is not in"
)

// WithIncidentDate indicates the date range of an incident
func WithIncidentDate(comparator IncidentDateComparator, from, to time.Time) QueryOption {
	return func(qo *QueryOptions) {
		criterionID := uuid.New().String()
		qo.queryData.Add(fmt.Sprintf("query[filters][%s][type]", criterionID), "IncidentDate")
		qo.queryData.Add(fmt.Sprintf("query[filters][%s][outer_filter][weight]", criterionID), "0.001")
		qo.queryData.Add(fmt.Sprintf("query[filters][%s][outer_filter][comparator]", criterionID), string(comparator))
		qo.queryData.Add(fmt.Sprintf("query[filters][%s][outer_filter][filter][field][date-from]", criterionID), from.Format("01/02/2006"))
		qo.queryData.Add(fmt.Sprintf("query[filters][%s][outer_filter][filter][field][date-to]", criterionID), to.Format("01/02/2006"))
	}
}
