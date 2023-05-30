// Package flightpath implements flight path logic.
package flightpath

import "sort"

// A Flight represents a single flight between 2 airports.
type Flight struct {
	From, To string
}

func (f Flight) String() string {
	return f.From + "->" + f.To
}

// Calculate summarises a list of flights to a single flight between the list
// source and destination airports.
func Calculate(flights []*Flight) *Flight {
	return calculateSort(flights)
}

func calculateSort(flights []*Flight) *Flight {
	switch len(flights) {
	case 1:
		return flights[0]
	case 2:
		if flights[0].From == flights[1].To {
			return &Flight{flights[1].From, flights[0].To}
		}
		return &Flight{flights[0].From, flights[1].To}
	}

	// Given [A->B, B->C, C->D]:
	// - B and C will be both present in the From and To positions
	// - A will never be in From position
	// - D will never be in To position
	//
	// Package sort uses less function to determine element order:
	// - left is less if left's To is equal to right's From
	// - left is less if right's To is not in list of Froms

	froms := make(map[string]bool, len(flights))
	for _, f := range flights {
		froms[f.From] = true
	}

	sort.Slice(flights, func(i, j int) bool {
		return flights[i].To == flights[j].From || !froms[flights[j].To]
	})

	return &Flight{
		From: flights[0].From,
		To:   flights[len(flights)-1].To,
	}
}

func calculateReduce(flights []*Flight) *Flight {
	if len(flights) == 1 {
		return flights[0]
	}

	// "In order to determine the flight path of a person, we must sort through
	// all of their flight records."
	//
	// "to find the total flight paths starting and ending airports"
	//
	// Following these 2 statements it is not required to have the full list
	// available, implement algorithm using sorting for now though.

	route := *flights[0]
	unprocessed := flights[1:]
	var unmatched []*Flight

loop:
	for _, f := range unprocessed {
		switch {
		case route.From == f.To:
			route.From = f.From
		case route.To == f.From:
			route.To = f.To
		default:
			unmatched = append(unmatched, f)
		}
	}
	if len(unmatched) > 0 {
		unprocessed = unmatched
		unmatched = nil
		goto loop
	}

	return &route
}
