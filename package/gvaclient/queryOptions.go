package gvaclient

import (
	"fmt"
	"net/url"
	"time"

	"github.com/google/uuid"
)

type QueryOptions struct {
	queryData url.Values
}
type QueryID string

type QueryOption func(*QueryOptions)

func WithAllCriteriaMatching(all bool) QueryOption {
	return func(qo *QueryOptions) {
		value := "Or"
		if all {
			value = "And"
		}
		qo.queryData.Add("query[base_group][base_group_select]", value)
	}
}

type ResultsType string

const (
	Incidents   ResultsType = "incidents"
	Particiants ResultsType = "participants"
)

func WithResultType(resultType ResultsType) QueryOption {
	return func(qo *QueryOptions) {
		qo.queryData.Add("query[results_type][select]", string(resultType))
	}
}

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

type IncidentDateComparator string

const (
	IsIn           IncidentDateComparator = "is in"
	IsNotIn        IncidentDateComparator = "is not in"
	IsInTheLast    IncidentDateComparator = "is in the last"
	IsNotInTheLast IncidentDateComparator = "is not in the last"
	IsCurrentYear  IncidentDateComparator = "is current year"
	IsYear         IncidentDateComparator = "is year"
	IsNotYear      IncidentDateComparator = "is not year"
)

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
