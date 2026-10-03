package engine

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/nicolo-ribaudo/netex-validator/types"
	"github.com/nicolo-ribaudo/netex-validator/validation/context"
)

// ServiceJourneyObjectValidator validates service journey business logic using object model
type ServiceJourneyObjectValidator struct {
	*BaseObjectValidator
}

// NewServiceJourneyObjectValidator creates a new service journey object validator
func NewServiceJourneyObjectValidator() *ServiceJourneyObjectValidator {
	rules := []types.ValidationRule{
		{
			Code:     "SJ_OBJ_1",
			Name:     "Service journey pattern consistency",
			Message:  "Service journey must have consistent number of passing times with journey pattern",
			Severity: types.ERROR,
		},
		{
			Code:     "SJ_OBJ_2",
			Name:     "Service journey timing progression",
			Message:  "Service journey passing times must progress logically",
			Severity: types.ERROR,
		},
		{
			Code:     "SJ_OBJ_3",
			Name:     "Service journey transport mode consistency",
			Message:  "Service journey transport mode must be compatible with line",
			Severity: types.ERROR,
		},
		{
			Code:     "SJ_OBJ_4",
			Name:     "Service journey operator consistency",
			Message:  "Service journey must reference valid operator",
			Severity: types.ERROR,
		},
		{
			Code:     "SJ_OBJ_5",
			Name:     "Service journey calendar consistency",
			Message:  "Service journey must have valid calendar information",
			Severity: types.ERROR,
		},
		{
			Code:     "SJ_OBJ_6",
			Name:     "Service journey realistic timing",
			Message:  "Service journey timing should be realistic",
			Severity: types.WARNING,
		},
	}

	base := NewBaseObjectValidator("ServiceJourneyObjectValidator", rules)
	return &ServiceJourneyObjectValidator{
		BaseObjectValidator: base,
	}
}

// Validate performs service journey object validation
func (v *ServiceJourneyObjectValidator) Validate(ctx *context.ObjectValidationContext) []types.ValidationIssue {
	var issues []types.ValidationIssue

	serviceJourneys := ctx.ServiceJourneys()

	for _, sj := range serviceJourneys {
		// Validate pattern consistency
		issues = append(issues, v.validatePatternConsistency(ctx, sj)...)

		// Validate timing progression
		issues = append(issues, v.validateTimingProgression(ctx, sj)...)

		// Validate transport mode consistency
		issues = append(issues, v.validateTransportModeConsistency(ctx, sj)...)

		// Validate operator reference
		issues = append(issues, v.validateOperatorReference(ctx, sj)...)

		// Validate calendar consistency
		issues = append(issues, v.validateCalendarConsistency(ctx, sj)...)

		// Validate realistic timing
		issues = append(issues, v.validateRealisticTiming(ctx, sj)...)
	}

	return issues
}

// validatePatternConsistency checks if service journey has consistent passing times with journey pattern
func (v *ServiceJourneyObjectValidator) validatePatternConsistency(ctx *context.ObjectValidationContext, sj *context.ServiceJourney) []types.ValidationIssue {
	var issues []types.ValidationIssue

	if sj.JourneyPatternRef == nil {
		return issues // This is handled by XPath rules
	}

	journeyPattern := ctx.GetJourneyPattern(sj.JourneyPatternRef.Ref)
	if journeyPattern == nil {
		// Reference not found - this will be caught by reference validation
		return issues
	}

	// Check if passing times count matches journey pattern stops
	if sj.PassingTimes != nil && journeyPattern.PointsInSequence != nil {
		passingTimesCount := len(sj.PassingTimes.TimetabledPassingTimes)
		stopPointsCount := len(journeyPattern.PointsInSequence.StopPointInJourneyPatterns)

		if passingTimesCount != stopPointsCount {
			issues = append(issues, types.ValidationIssue{
				Rule: v.rules[0], // SJ_OBJ_1
				Location: types.DataLocation{
					FileName:  ctx.FileName,
					ElementID: sj.ID,
				},
				Message: fmt.Sprintf("Service journey '%s' has %d passing times but journey pattern '%s' has %d stop points",
					sj.ID, passingTimesCount, journeyPattern.ID, stopPointsCount),
			})
		}

		// Validate order consistency
		for i, tpt := range sj.PassingTimes.TimetabledPassingTimes {
			if i < len(journeyPattern.PointsInSequence.StopPointInJourneyPatterns) {
				expectedStopRef := journeyPattern.PointsInSequence.StopPointInJourneyPatterns[i].ScheduledStopPointRef
				if tpt.StopPointInJourneyPatternRef != nil && expectedStopRef != nil {
					// The timetabled passing time should reference the corresponding stop point in journey pattern
					// This is a complex validation that requires deep object model navigation
					jpStopPoint := journeyPattern.PointsInSequence.StopPointInJourneyPatterns[i]
					if tpt.StopPointInJourneyPatternRef.Ref != jpStopPoint.ID {
						issues = append(issues, types.ValidationIssue{
							Rule: v.rules[0], // SJ_OBJ_1
							Location: types.DataLocation{
								FileName:  ctx.FileName,
								ElementID: tpt.ID,
							},
							Message: fmt.Sprintf("Timetabled passing time order mismatch in service journey '%s'", sj.ID),
						})
					}
				}
			}
		}
	}

	return issues
}

// validateTimingProgression checks if passing times progress logically
func (v *ServiceJourneyObjectValidator) validateTimingProgression(ctx *context.ObjectValidationContext, sj *context.ServiceJourney) []types.ValidationIssue {
	var issues []types.ValidationIssue

	if sj.PassingTimes == nil || len(sj.PassingTimes.TimetabledPassingTimes) < 2 {
		return issues
	}

	var previousTime time.Time
	var hasValidPreviousTime bool

	for i, tpt := range sj.PassingTimes.TimetabledPassingTimes {
		var currentTime time.Time
		var hasValidCurrentTime bool

		// Use departure time for progression, fall back to arrival time
		timeStr := tpt.DepartureTime
		if timeStr == "" {
			timeStr = tpt.ArrivalTime
		}

		if timeStr != "" {
			if parsedTime, err := parseTimeString(timeStr); err == nil {
				currentTime = parsedTime
				hasValidCurrentTime = true
			}
		}

		// Check progression
		if hasValidPreviousTime && hasValidCurrentTime {
			if !currentTime.After(previousTime) {
				issues = append(issues, types.ValidationIssue{
					Rule: v.rules[1], // SJ_OBJ_2
					Location: types.DataLocation{
						FileName:  ctx.FileName,
						ElementID: tpt.ID,
					},
					Message: fmt.Sprintf("Timing regression in service journey '%s' at stop %d", sj.ID, i+1),
				})
			}
		}

		// Check arrival/departure consistency at same stop
		if tpt.ArrivalTime != "" && tpt.DepartureTime != "" {
			if arrivalTime, err1 := parseTimeString(tpt.ArrivalTime); err1 == nil {
				if departureTime, err2 := parseTimeString(tpt.DepartureTime); err2 == nil {
					if arrivalTime.After(departureTime) {
						issues = append(issues, types.ValidationIssue{
							Rule: v.rules[1], // SJ_OBJ_2
							Location: types.DataLocation{
								FileName:  ctx.FileName,
								ElementID: tpt.ID,
							},
							Message: fmt.Sprintf("Arrival time after departure time in service journey '%s'", sj.ID),
						})
					}
				}
			}
		}

		if hasValidCurrentTime {
			previousTime = currentTime
			hasValidPreviousTime = true
		}
	}

	return issues
}

// validateTransportModeConsistency checks transport mode compatibility with line
func (v *ServiceJourneyObjectValidator) validateTransportModeConsistency(ctx *context.ObjectValidationContext, sj *context.ServiceJourney) []types.ValidationIssue {
	var issues []types.ValidationIssue

	if sj.TransportMode == "" || sj.LineRef == nil {
		return issues // No override or no line reference
	}

	// Get the referenced line
	line := ctx.GetLine(sj.LineRef.Ref)
	if line == nil {
		// Check flexible line
		flexLine := ctx.GetFlexibleLine(sj.LineRef.Ref)
		if flexLine != nil && flexLine.TransportMode != "" {
			if sj.TransportMode != flexLine.TransportMode {
				issues = append(issues, types.ValidationIssue{
					Rule: v.rules[2], // SJ_OBJ_3
					Location: types.DataLocation{
						FileName:  ctx.FileName,
						ElementID: sj.ID,
					},
					Message: fmt.Sprintf("Service journey '%s' transport mode '%s' incompatible with flexible line '%s' mode '%s'",
						sj.ID, sj.TransportMode, flexLine.ID, flexLine.TransportMode),
				})
			}
		}
		return issues
	}

	// Check consistency with line transport mode
	if line.TransportMode != "" && sj.TransportMode != line.TransportMode {
		issues = append(issues, types.ValidationIssue{
			Rule: v.rules[2], // SJ_OBJ_3
			Location: types.DataLocation{
				FileName:  ctx.FileName,
				ElementID: sj.ID,
			},
			Message: fmt.Sprintf("Service journey '%s' transport mode '%s' incompatible with line '%s' mode '%s'",
				sj.ID, sj.TransportMode, line.ID, line.TransportMode),
		})
	}

	return issues
}

// validateOperatorReference checks if operator reference is valid
func (v *ServiceJourneyObjectValidator) validateOperatorReference(ctx *context.ObjectValidationContext, sj *context.ServiceJourney) []types.ValidationIssue {
	var issues []types.ValidationIssue

	var operatorRef string

	// Service journey can have direct operator reference or inherit from line
	if sj.OperatorRef != nil {
		operatorRef = sj.OperatorRef.Ref
	} else if sj.LineRef != nil {
		// Check line for operator reference
		line := ctx.GetLine(sj.LineRef.Ref)
		if line != nil && line.OperatorRef != nil {
			operatorRef = line.OperatorRef.Ref
		} else {
			// Check flexible line
			flexLine := ctx.GetFlexibleLine(sj.LineRef.Ref)
			if flexLine != nil && flexLine.OperatorRef != nil {
				operatorRef = flexLine.OperatorRef.Ref
			}
		}
	}

	if operatorRef == "" {
		issues = append(issues, types.ValidationIssue{
			Rule: v.rules[3], // SJ_OBJ_4
			Location: types.DataLocation{
				FileName:  ctx.FileName,
				ElementID: sj.ID,
			},
			Message: fmt.Sprintf("Service journey '%s' has no operator reference", sj.ID),
		})
		return issues
	}

	// Check if operator exists locally
	operator := ctx.GetOperator(operatorRef)
	if operator == nil {
		// Check common data repository if available
		if commonRepo := ctx.GetCommonDataRepository(); commonRepo != nil {
			operator = commonRepo.GetSharedOperator(operatorRef)
		}

		if operator == nil {
			issues = append(issues, types.ValidationIssue{
				Rule: v.rules[3], // SJ_OBJ_4
				Location: types.DataLocation{
					FileName:  ctx.FileName,
					ElementID: sj.ID,
				},
				Message: fmt.Sprintf("Service journey '%s' references non-existent operator '%s'", sj.ID, operatorRef),
			})
		}
	}

	return issues
}

// validateCalendarConsistency checks calendar information consistency
func (v *ServiceJourneyObjectValidator) validateCalendarConsistency(ctx *context.ObjectValidationContext, sj *context.ServiceJourney) []types.ValidationIssue {
	var issues []types.ValidationIssue

	// Service journey should have either day types or be referenced by dated service journey
	hasDayTypes := sj.DayTypes != nil && len(sj.DayTypes.DayTypeRefs) > 0

	// Check if referenced by any dated service journey
	hasDateServiceJourneyRef := false
	for _, dsj := range ctx.DatedServiceJourneys() {
		if dsj.ServiceJourneyRef != nil && dsj.ServiceJourneyRef.Ref == sj.ID {
			hasDateServiceJourneyRef = true
			break
		}
	}

	if !hasDayTypes && !hasDateServiceJourneyRef {
		issues = append(issues, types.ValidationIssue{
			Rule: v.rules[4], // SJ_OBJ_5
			Location: types.DataLocation{
				FileName:  ctx.FileName,
				ElementID: sj.ID,
			},
			Message: fmt.Sprintf("Service journey '%s' has no calendar information (day types or dated service journey reference)", sj.ID),
		})
	}

	// Validate day type references if present
	if hasDayTypes {
		for _, dayTypeRef := range sj.DayTypes.DayTypeRefs {
			if dayTypeRef.Ref != "" {
				dayType := ctx.GetDayType(dayTypeRef.Ref)
				if dayType == nil {
					issues = append(issues, types.ValidationIssue{
						Rule: v.rules[4], // SJ_OBJ_5
						Location: types.DataLocation{
							FileName:  ctx.FileName,
							ElementID: sj.ID,
						},
						Message: fmt.Sprintf("Service journey '%s' references non-existent day type '%s'", sj.ID, dayTypeRef.Ref),
					})
				}
			}
		}
	}

	return issues
}

// validateRealisticTiming checks if journey timing is realistic
func (v *ServiceJourneyObjectValidator) validateRealisticTiming(ctx *context.ObjectValidationContext, sj *context.ServiceJourney) []types.ValidationIssue {
	var issues []types.ValidationIssue

	if sj.PassingTimes == nil || len(sj.PassingTimes.TimetabledPassingTimes) < 2 {
		return issues
	}

	// Calculate total journey time
	firstStop := sj.PassingTimes.TimetabledPassingTimes[0]
	lastStop := sj.PassingTimes.TimetabledPassingTimes[len(sj.PassingTimes.TimetabledPassingTimes)-1]

	startTimeStr := firstStop.DepartureTime
	if startTimeStr == "" {
		startTimeStr = firstStop.ArrivalTime
	}

	endTimeStr := lastStop.ArrivalTime
	if endTimeStr == "" {
		endTimeStr = lastStop.DepartureTime
	}

	if startTimeStr != "" && endTimeStr != "" {
		if startTime, err1 := parseTimeString(startTimeStr); err1 == nil {
			if endTime, err2 := parseTimeString(endTimeStr); err2 == nil {
				duration := endTime.Sub(startTime)

				// Check for unreasonably long journeys (over 12 hours)
				if duration > 12*time.Hour {
					issues = append(issues, types.ValidationIssue{
						Rule: v.rules[5], // SJ_OBJ_6
						Location: types.DataLocation{
							FileName:  ctx.FileName,
							ElementID: sj.ID,
						},
						Message: fmt.Sprintf("Service journey '%s' has unusually long duration: %v", sj.ID, duration),
					})
				}

				// Check for unreasonably short journeys (under 1 minute for multi-stop)
				if duration < time.Minute && len(sj.PassingTimes.TimetabledPassingTimes) > 2 {
					issues = append(issues, types.ValidationIssue{
						Rule: v.rules[5], // SJ_OBJ_6
						Location: types.DataLocation{
							FileName:  ctx.FileName,
							ElementID: sj.ID,
						},
						Message: fmt.Sprintf("Service journey '%s' has unusually short duration: %v", sj.ID, duration),
					})
				}
			}
		}
	}

	// Check individual stop times
	for _, tpt := range sj.PassingTimes.TimetabledPassingTimes {
		if tpt.ArrivalTime != "" && tpt.DepartureTime != "" {
			if arrTime, err1 := parseTimeString(tpt.ArrivalTime); err1 == nil {
				if depTime, err2 := parseTimeString(tpt.DepartureTime); err2 == nil {
					stopTime := depTime.Sub(arrTime)

					// Check for unreasonably long stop times (over 30 minutes)
					if stopTime > 30*time.Minute {
						issues = append(issues, types.ValidationIssue{
							Rule: v.rules[5], // SJ_OBJ_6
							Location: types.DataLocation{
								FileName:  ctx.FileName,
								ElementID: tpt.ID,
							},
							Message: fmt.Sprintf("Unusually long stop time in service journey '%s': %v", sj.ID, stopTime),
						})
					}
				}
			}
		}
	}

	return issues
}

// parseTimeString parses NetEX time string format (HH:MM:SS)
func parseTimeString(timeStr string) (time.Time, error) {
	// Handle various time formats
	timeStr = strings.TrimSpace(timeStr)

	// Simple HH:MM:SS format
	if len(timeStr) == 8 && strings.Count(timeStr, ":") == 2 {
		parts := strings.Split(timeStr, ":")
		if len(parts) == 3 {
			hour, err1 := strconv.Atoi(parts[0])
			minute, err2 := strconv.Atoi(parts[1])
			second, err3 := strconv.Atoi(parts[2])

			if err1 == nil && err2 == nil && err3 == nil {
				// Use a fixed date for time comparison
				return time.Date(2023, 1, 1, hour, minute, second, 0, time.UTC), nil
			}
		}
	}

	return time.Time{}, fmt.Errorf("invalid time format: %s", timeStr)
}
