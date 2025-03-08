package types

type Event struct {
	Name      string
	StartTime int64
	EndTime   int64
	Location  *string
	Detail    *EventDetail
	IsAllDay  bool
}

type EventDetail struct {
	IsToday           bool `json:"isToday"`
	IsTomorrow        bool `json:"isTomorrow"`
	IsThisWeek        bool `json:"isThisWeek"`
	MinutesUntilStart int  `json:"minutesUntilStart"`
	MinutesUntilEnd   int  `json:"minutesUntilEnd"`
	HoursToEnd        int  `json:"hoursToEnd"`
	InProgress        bool `json:"inProgress"`
	IsAllDay          bool `json:"isAllDay"`
}

type BaseResponse struct {
	Data  *IcsResponse   `json:"data"`
	Error *ErrorResponse `json:"message"`
}


//ShowInProgress and IncludeAllDayEvents are set as pointers so that they can default to true if they are unset
type IcsRequest struct {
	ICSUrl         string `json:"icsUrl" validate:"required,url"`
	TZ             string `json:"tz" validate:"required,timezone"`
	ShowInProgress *bool   `json:"showInProgress" validate:"boolean"`
	IncludeAllDayEvents *bool `json:"includeAllDayEvents" validate:"boolean"`
	OnlyShowAllDayEvents bool `json:"onlyShowAllDayEvents" validate:"boolean"`
}

type ErrorResponse struct {
	Error   bool   `json:"error"`
	Message string `json:"message"`
}

type IcsResponse struct {
	Name      string       `json:"name"`
	StartTime int64        `json:"start"`
	EndTime   int64        `json:"end"`
	Location  *string      `json:"location"`
	Detail    *EventDetail `json:"detail"`
}
