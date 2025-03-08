package main

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/go-playground/validator/v10"

	c "github.com/quesurifn/ics-calendar-tidbyt-lambda/ics/calendar"
	t "github.com/quesurifn/ics-calendar-tidbyt-lambda/ics/types"
	"go.uber.org/zap"
)


type Response events.APIGatewayProxyResponse

func HandleRequest(ctx context.Context, request events.APIGatewayProxyRequest) (Response, error) {
	logger, err := zap.NewProduction()
	if err != nil {
		return GetErrorResponseType(err)
	}

	event := &t.IcsRequest{}
	err = json.Unmarshal([]byte(request.Body), event)
	if err != nil {
		logger.Error("Error", zap.Any("err", err))
		return GetErrorResponseType(err)
	}

	// Set default for ShowInProgress if not provided
	if event.ShowInProgress == nil {
		defaultTrue := true
		event.ShowInProgress = &defaultTrue
	}

	// Set default for IncludeAllDayEvents if not provided
	if event.IncludeAllDayEvents == nil {
		defaultTrue := true
		event.IncludeAllDayEvents = &defaultTrue
	}

	val := validator.New()
	err = val.Struct(event)
	if err != nil {
		validationErrors := err.(validator.ValidationErrors)
		logger.Error("Error", zap.Any("err", validationErrors))
		return GetErrorResponseType(validationErrors)
	}

	cal := c.Calendar{
		Logger: logger,
	}

	data, err := cal.DownloadCalendar(event.ICSUrl)
	if err != nil {
		logger.Error("Error", zap.Any("err", err))
		return GetErrorResponseType(err)
	}

	events, err := cal.ParseCalendar(data, event.TZ)
	if err != nil {
		logger.Error("Error", zap.Any("err", err))
		return GetErrorResponseType(err)
	}

	if len(events) == 0 {
		base := t.BaseResponse{
			Data: nil,
		}
		respBytes, err := json.Marshal(base)
		if err != nil {
			logger.Error("Error", zap.Any("err", err))
			return GetErrorResponseType(err)
		}

		return Response{
			StatusCode:      200,
			IsBase64Encoded: false,
			Body:            string(respBytes),
			Headers: map[string]string{
				"Content-Type": "application/json",
			},
		}, nil
	}

	nextEvent, err := cal.NextEvent(events, event.TZ, event.IncludeAllDayEvents, event.OnlyShowAllDayEvents, event.ShowInProgress)
	if err != nil {
		//the error here is events is empty, so return an empty response
			base := t.BaseResponse{
				Data: nil,
			}
			respBytes, err := json.Marshal(base)
			if err != nil {
				logger.Error("Error", zap.Any("err", err))
				return GetErrorResponseType(err)
			}
	
			return Response{
				StatusCode:      200,
				IsBase64Encoded: false,
				Body:            string(respBytes),
				Headers: map[string]string{
					"Content-Type": "application/json",
				},
			}, nil
	}

	response := t.BaseResponse{
		Data: &t.IcsResponse{
			Name:      nextEvent.Name,
			StartTime: nextEvent.StartTime,
			EndTime:   nextEvent.EndTime,
			Location:  nextEvent.Location,
			Detail: &t.EventDetail{
				IsToday:           nextEvent.Detail.IsToday,
				IsTomorrow:        nextEvent.Detail.IsTomorrow,
				IsThisWeek:        nextEvent.Detail.IsThisWeek,
				MinutesUntilStart: nextEvent.Detail.MinutesUntilStart,
				MinutesUntilEnd:   nextEvent.Detail.MinutesUntilEnd,
				HoursToEnd:        nextEvent.Detail.HoursToEnd,
				InProgress:        nextEvent.Detail.InProgress,
				IsAllDay:		   nextEvent.Detail.IsAllDay,
			},
		},
	}

	respBytes, err := json.Marshal(response)
	if err != nil {
		logger.Error("Error", zap.Any("err", err))
		return GetErrorResponseType(err)
	}

	resp := Response{
		StatusCode:      200,
		IsBase64Encoded: false,
		Body:            string(respBytes),
		Headers: map[string]string{
			"Content-Type":           "application/json",
			"X-MyCompany-Func-Reply": "hello-handler",
		},
	}

	return resp, nil
}

func main() {
	lambda.Start(HandleRequest)
}

func GetErrorResponseType(err error) (Response, error) {
	resp := &t.BaseResponse{
		Error: &t.ErrorResponse{
			Error:   true,
			Message: err.Error(),
		},
	}

	respBytes, err := json.Marshal(resp)
	if err != nil {
		return Response{
			StatusCode: 500,
		}, fmt.Errorf("failed to marshal response: %w", err)
	}

	formattedResp := Response{
		StatusCode:      400,
		IsBase64Encoded: false,
		Body:            string(respBytes),
		Headers: map[string]string{
			"Content-Type": "application/json",
		},
	}

	return formattedResp, nil
}
