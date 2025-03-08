package calendar

import (
	"sort"
	"strings"
	"time"
	"fmt"
	"errors"

	"github.com/apognu/gocal"
	t "github.com/quesurifn/ics-calendar-tidbyt-lambda/ics/types"
	"go.uber.org/zap"
	"gopkg.in/resty.v1"
)

type Calendar struct {
	Logger *zap.Logger
}

func (c Calendar) DownloadCalendar(url string) (string, error) {
	client := resty.New()

	resp, err := client.R().Get(url)
	if err != nil {
		return "", err
	}

	return resp.String(), err
}

func (c Calendar) ParseCalendar(data string, tz string) ([]t.Event, error) {
	usersLoc, err := time.LoadLocation(tz)
	if err != nil {
		c.Logger.Error("Error", zap.Any("err", err))
		return nil, err
	}

	parser := gocal.NewParser(strings.NewReader(data))

	//look back one day for all day events
	//this introduces additional complexity because now old events are showing up
	start, end := time.Now().AddDate(0, 0, -1).In(usersLoc), time.Now().AddDate(0, 0, 7).In(usersLoc)
	parser.Start, parser.End = &start, &end

	parser.Parse()

	var events []t.Event
	for _, e := range parser.Events {
		events = append(events, t.Event{
			Name:      e.Summary,
			StartTime: e.Start.Unix(),
			EndTime:   e.End.Unix(),
			Location:  &e.Location,
		})
	}

	c.Logger.Info("ParseCalendar", zap.Any("events", events))

	return events, nil
}

func (c Calendar) NextEvent(events []t.Event, tz string, incAllDayPtr *bool, onlyAllDay bool, showInProgressEventsPtr *bool) (*t.Event, error) {
	//dereference the pointers
	showInProgressEvents := *showInProgressEventsPtr
	incAllDay := *incAllDayPtr

	if len(events) == 0 {
		return nil, nil
	}

	location, err := time.LoadLocation(tz)
	if err != nil {
		c.Logger.Error("Error", zap.Any("err", err))
		return nil, err
	}

	now := time.Now().In(location).Unix()


	offset, err := GetOffset(tz)
	if err != nil {
		fmt.Println("Error:", err)

		//set offset to UTC if error
		offset = 0
	}

	// do not include All Day events
	if(!incAllDay){
		//look through all events
		var notAllDayEvents []t.Event
		for i := range events {
			//look for events that are not all day, rebuild events slice
			if (!isAllDayEvent(events[i])){
				notAllDayEvents = append(notAllDayEvents, events[i])
			}		
		}
		events = notAllDayEvents

		if len(events) == 0 {
			return nil, errors.New("Events is empty")
		}
	}else{
	//specifically show all day events
		//loop through all events in slice
		for i := range events {
			//look for all day events
			if (isAllDayEvent(events[i])){
				//if event starts and ends at midnight UTC, then it is an all day event
				//convert start and end times from UTC to the correct timezone
				events[i].StartTime = events[i].StartTime - int64(offset)
				events[i].EndTime = events[i].EndTime - int64(offset)
				//IsAllDay gets set here because it can attach to the array
				events[i].IsAllDay = true

			}		
		}
	}

	if(onlyAllDay){
		//rebuild events to only have all day events
		var allDayEvents []t.Event
		for _, e := range events {
			if e.IsAllDay {
				allDayEvents = append(allDayEvents, e)
			}
		}
		events = allDayEvents

		if len(events) == 0 {
			return nil, errors.New("Events is empty")
		}
	}


	//build a list of events that are in progress
	eventsInProgress := FilterInProgress(events)

    //if there are events in progress and we are showing in progress events,  use the filtered list
	if len(eventsInProgress) > 0 && showInProgressEvents{
		events = eventsInProgress
		//if there are multiple events in progress, sort by end time
		//this will put timed events that are in progress ahead of all day events
		sort.Slice(events, func(i, j int) bool {
			return events[i].EndTime < events[j].EndTime
		})
	}else{
		//there are no events in progress (or we are ignoring them), so sort by events that are starting first
		//this will put all day events (starting at midnight) ahead of timed events
		sort.Slice(events, func(i, j int) bool {
			return events[i].StartTime < events[j].StartTime
		})
	}


	//use the first event as the next event
	next := events[0]
	//need this to reference i outside of below for loop
	slicePointer := 0
	//but sometimes events that have already ended get pulled because of the 1 day look back, so look for events that have an EndTime after now
	for i := 0; next.EndTime < now; i++ {
		if i >= len(events){
			//return error because we got to the end of the array and didn't find any events that weren't in the past
			return nil, errors.New("Events is empty")
		}
		next = events[i]
		slicePointer = i
	}

	//reset events to only be the events identified above by reindexing
	events = events[slicePointer:]
	slicePointer = 0

	//if showInProgressEvents is true, display all events
	//if showInProgressEvents is false, find an event that isn't in progress, starting at index 0
	//need to add error handling here to make sure we don't go out of bounds
	if (!showInProgressEvents){
		for j := 0; next.StartTime <= now; j++ {
			if j >= len(events){
				//return error because we got to the end of the array and didn't find any events that weren't in progress
				return nil, errors.New("Events is empty")
			}
			next = events[j]
		}
	}

	next.Detail = &t.EventDetail{}
	next.Detail.InProgress = now >= next.StartTime

	//this isn't really needed because all events are in the next week thanks to what we passed to the parser in ParseCalendar, but could eventually be useful
	next.Detail.IsThisWeek = now < next.StartTime+7*24*60*60
	//adding In(location) to time.Unix which will convert the time to the correct timezone
	next.Detail.IsToday = time.Unix(now, 0).In(location).Day() == time.Unix(next.StartTime, 0).In(location).Day()
	next.Detail.IsTomorrow = time.Unix(now, 0).In(location).Day() == time.Unix(next.StartTime, 0).In(location).Day()-1
	next.Detail.MinutesUntilStart = int(next.StartTime-now) / 60
	next.Detail.MinutesUntilEnd = int(next.EndTime-now) / 60
	next.Detail.HoursToEnd = int(next.EndTime-now) / 60 / 60
	//IsAllDay is set above, but it belongs in the details
	next.Detail.IsAllDay = next.IsAllDay

	c.Logger.Info("NextEvent", zap.Any("nextEvent", next))
	c.Logger.Info("Now", zap.Any("now", now))

	return &next, nil
}

//isAllDayEvent function verifies that both the Start is at midnight UTC 
//and End is at one second until midnight UTC.
//all day events can span multiple days, but they always start and end at midnight
func isAllDayEvent(e t.Event) bool {
	//look for divisible by 86400, so starts at UTC midnight
	startMidnightUTC := e.StartTime % 86400 == 0
	//look for to see if it ends at 11:59:59 so add a second and divide by 86400
	endMidnightUTC := (e.EndTime +1) % 86400 == 0	

	return startMidnightUTC && endMidnightUTC
}


//FilterInProgress filters events that are currently in progress
func FilterInProgress(events []t.Event) []t.Event {
	now := time.Now().Unix()
	// Slice to store events with InProgress = true
	var filteredEvents []t.Event

	// Iterate through the input slice
	for _, event := range events {
		//look to see if we are in between start time and end time
		if event.StartTime <= now && event.EndTime >= now {
			filteredEvents = append(filteredEvents, event)
		}
	}

	// Return the filtered slice (empty if no matches)
	return filteredEvents
}


// GetOffset returns the offset in seconds for a given timezone
func GetOffset(timezone string) (int, error) {
	// Load the location
	loc, err := time.LoadLocation(timezone)
	if err != nil {
		return 0, fmt.Errorf("failed to load location: %w", err)
	}

	// Get the current time in the specified location
	now := time.Now().In(loc)

	// Get the offset in seconds from UTC
	_, offset := now.Zone()

	return offset, nil
}
