package engine

import (
	"fmt"

	"github.com/nicolo-ribaudo/netex-validator/types"
	"github.com/nicolo-ribaudo/netex-validator/validation/context"
)

// NetworkConsistencyValidator validates network-level consistency using object model
type NetworkConsistencyValidator struct {
	*BaseObjectValidator
}

// NewNetworkConsistencyValidator creates a new network consistency validator
func NewNetworkConsistencyValidator() *NetworkConsistencyValidator {
	rules := []types.ValidationRule{
		{
			Code:     "NET_OBJ_1",
			Name:     "Line-Route-JourneyPattern hierarchy",
			Message:  "Routes must belong to lines and journey patterns must belong to routes",
			Severity: types.ERROR,
		},
		{
			Code:     "NET_OBJ_2",
			Name:     "Operator-Line consistency",
			Message:  "Lines must reference valid operators from ResourceFrame",
			Severity: types.ERROR,
		},
		{
			Code:     "NET_OBJ_3",
			Name:     "Authority-Network consistency",
			Message:  "Networks must reference valid authorities",
			Severity: types.ERROR,
		},
		{
			Code:     "NET_OBJ_4",
			Name:     "Stop assignment consistency",
			Message:  "Stop assignments must reference valid stop points and stop places",
			Severity: types.ERROR,
		},
		{
			Code:     "NET_OBJ_5",
			Name:     "Network completeness",
			Message:  "Network should have complete organizational structure",
			Severity: types.WARNING,
		},
		{
			Code:     "NET_OBJ_6",
			Name:     "Cross-frame reference consistency",
			Message:  "References between frames must be consistent",
			Severity: types.ERROR,
		},
	}

	base := NewBaseObjectValidator("NetworkConsistencyValidator", rules)
	return &NetworkConsistencyValidator{
		BaseObjectValidator: base,
	}
}

// Validate performs network consistency validation
func (v *NetworkConsistencyValidator) Validate(ctx *context.ObjectValidationContext) []types.ValidationIssue {
	var issues []types.ValidationIssue

	// Validate Line-Route-JourneyPattern hierarchy
	issues = append(issues, v.validateHierarchy(ctx)...)

	// Validate operator-line consistency
	issues = append(issues, v.validateOperatorLineConsistency(ctx)...)

	// Validate authority-network consistency
	issues = append(issues, v.validateAuthorityNetworkConsistency(ctx)...)

	// Validate stop assignment consistency
	issues = append(issues, v.validateStopAssignmentConsistency(ctx)...)

	// Validate network completeness
	issues = append(issues, v.validateNetworkCompleteness(ctx)...)

	// Validate cross-frame references
	issues = append(issues, v.validateCrossFrameReferences(ctx)...)

	return issues
}

// validateHierarchy validates the Line-Route-JourneyPattern hierarchy
func (v *NetworkConsistencyValidator) validateHierarchy(ctx *context.ObjectValidationContext) []types.ValidationIssue {
	var issues []types.ValidationIssue

	// Check that all routes reference valid lines
	for _, route := range ctx.Routes() {
		if route.LineRef == nil {
			issues = append(issues, types.ValidationIssue{
				Rule: v.rules[0], // NET_OBJ_1
				Location: types.DataLocation{
					FileName:  ctx.FileName,
					ElementID: route.ID,
				},
				Message: fmt.Sprintf("Route '%s' has no line reference", route.ID),
			})
			continue
		}

		// Check if referenced line exists
		lineExists := ctx.GetLine(route.LineRef.Ref) != nil || ctx.GetFlexibleLine(route.LineRef.Ref) != nil
		if !lineExists {
			issues = append(issues, types.ValidationIssue{
				Rule: v.rules[0], // NET_OBJ_1
				Location: types.DataLocation{
					FileName:  ctx.FileName,
					ElementID: route.ID,
				},
				Message: fmt.Sprintf("Route '%s' references non-existent line '%s'", route.ID, route.LineRef.Ref),
			})
		}
	}

	// Check that all journey patterns reference valid routes
	for _, jp := range ctx.JourneyPatterns() {
		if jp.RouteRef == nil {
			issues = append(issues, types.ValidationIssue{
				Rule: v.rules[0], // NET_OBJ_1
				Location: types.DataLocation{
					FileName:  ctx.FileName,
					ElementID: jp.ID,
				},
				Message: fmt.Sprintf("Journey pattern '%s' has no route reference", jp.ID),
			})
			continue
		}

		// Check if referenced route exists
		route := ctx.GetRoute(jp.RouteRef.Ref)
		if route == nil {
			issues = append(issues, types.ValidationIssue{
				Rule: v.rules[0], // NET_OBJ_1
				Location: types.DataLocation{
					FileName:  ctx.FileName,
					ElementID: jp.ID,
				},
				Message: fmt.Sprintf("Journey pattern '%s' references non-existent route '%s'", jp.ID, jp.RouteRef.Ref),
			})
		}
	}

	return issues
}

// validateOperatorLineConsistency validates operator-line relationships
func (v *NetworkConsistencyValidator) validateOperatorLineConsistency(ctx *context.ObjectValidationContext) []types.ValidationIssue {
	var issues []types.ValidationIssue

	// Check regular lines
	for _, line := range ctx.Lines() {
		if line.OperatorRef != nil {
			operatorID := line.OperatorRef.Ref
			operator := ctx.GetOperator(operatorID)

			// Check common data repository if not found locally
			if operator == nil {
				if commonRepo := ctx.GetCommonDataRepository(); commonRepo != nil {
					operator = commonRepo.GetSharedOperator(operatorID)
				}
			}

			if operator == nil {
				issues = append(issues, types.ValidationIssue{
					Rule: v.rules[1], // NET_OBJ_2
					Location: types.DataLocation{
						FileName:  ctx.FileName,
						ElementID: line.ID,
					},
					Message: fmt.Sprintf("Line '%s' references non-existent operator '%s'", line.ID, operatorID),
				})
			}
		}
	}

	// Check flexible lines
	for _, flexLine := range ctx.FlexibleLines() {
		if flexLine.OperatorRef != nil {
			operatorID := flexLine.OperatorRef.Ref
			operator := ctx.GetOperator(operatorID)

			// Check common data repository if not found locally
			if operator == nil {
				if commonRepo := ctx.GetCommonDataRepository(); commonRepo != nil {
					operator = commonRepo.GetSharedOperator(operatorID)
				}
			}

			if operator == nil {
				issues = append(issues, types.ValidationIssue{
					Rule: v.rules[1], // NET_OBJ_2
					Location: types.DataLocation{
						FileName:  ctx.FileName,
						ElementID: flexLine.ID,
					},
					Message: fmt.Sprintf("Flexible line '%s' references non-existent operator '%s'", flexLine.ID, operatorID),
				})
			}
		}
	}

	return issues
}

// validateAuthorityNetworkConsistency validates authority-network relationships
func (v *NetworkConsistencyValidator) validateAuthorityNetworkConsistency(ctx *context.ObjectValidationContext) []types.ValidationIssue {
	var issues []types.ValidationIssue

	// This would require access to networks - simplified for this example
	// In a full implementation, we would need to access the ServiceFrame networks

	return issues
}

// validateStopAssignmentConsistency validates stop assignment relationships
func (v *NetworkConsistencyValidator) validateStopAssignmentConsistency(ctx *context.ObjectValidationContext) []types.ValidationIssue {
	var issues []types.ValidationIssue

	// Access stop assignments from ServiceFrame
	if !ctx.HasFrame("ServiceFrame") {
		return issues
	}

	// This is a complex validation that would require accessing the ServiceFrame
	// and iterating through stop assignments - simplified here
	// In practice, we would validate:
	// 1. ScheduledStopPointRef references exist
	// 2. StopPlaceRef/QuayRef references exist in SiteFrame
	// 3. Geographic consistency between stop points and stop places

	return issues
}

// validateNetworkCompleteness validates network organizational completeness
func (v *NetworkConsistencyValidator) validateNetworkCompleteness(ctx *context.ObjectValidationContext) []types.ValidationIssue {
	var issues []types.ValidationIssue

	// Check if we have the basic organizational structure
	if !ctx.HasFrame("ResourceFrame") {
		issues = append(issues, types.ValidationIssue{
			Rule: v.rules[4], // NET_OBJ_5
			Location: types.DataLocation{
				FileName: ctx.FileName,
			},
			Message: "Missing ResourceFrame - no organizational data available",
		})
		return issues
	}

	// Check for operators
	hasOperators := len(ctx.ServiceJourneys()) == 0 // If no service journeys, operators not required
	for _, sj := range ctx.ServiceJourneys() {
		if sj.OperatorRef != nil || (sj.LineRef != nil &&
			(ctx.GetLine(sj.LineRef.Ref) != nil && ctx.GetLine(sj.LineRef.Ref).OperatorRef != nil) ||
			(ctx.GetFlexibleLine(sj.LineRef.Ref) != nil && ctx.GetFlexibleLine(sj.LineRef.Ref).OperatorRef != nil)) {
			hasOperators = true
			break
		}
	}

	if !hasOperators && len(ctx.ServiceJourneys()) > 0 {
		issues = append(issues, types.ValidationIssue{
			Rule: v.rules[4], // NET_OBJ_5
			Location: types.DataLocation{
				FileName: ctx.FileName,
			},
			Message: "Service journeys exist but no operator references found",
		})
	}

	return issues
}

// validateCrossFrameReferences validates references between different frames
func (v *NetworkConsistencyValidator) validateCrossFrameReferences(ctx *context.ObjectValidationContext) []types.ValidationIssue {
	var issues []types.ValidationIssue

	// Validate TimetableFrame -> ServiceFrame references
	for _, dsj := range ctx.DatedServiceJourneys() {
		if dsj.ServiceJourneyRef != nil {
			serviceJourney := ctx.GetServiceJourney(dsj.ServiceJourneyRef.Ref)
			if serviceJourney == nil {
				issues = append(issues, types.ValidationIssue{
					Rule: v.rules[5], // NET_OBJ_6
					Location: types.DataLocation{
						FileName:  ctx.FileName,
						ElementID: dsj.ID,
					},
					Message: fmt.Sprintf("Dated service journey '%s' references non-existent service journey '%s'",
						dsj.ID, dsj.ServiceJourneyRef.Ref),
				})
			}
		}

		if dsj.OperatingDayRef != nil {
			operatingDay := ctx.GetOperatingDay(dsj.OperatingDayRef.Ref)
			if operatingDay == nil {
				issues = append(issues, types.ValidationIssue{
					Rule: v.rules[5], // NET_OBJ_6
					Location: types.DataLocation{
						FileName:  ctx.FileName,
						ElementID: dsj.ID,
					},
					Message: fmt.Sprintf("Dated service journey '%s' references non-existent operating day '%s'",
						dsj.ID, dsj.OperatingDayRef.Ref),
				})
			}
		}
	}

	// Validate ServiceFrame -> ResourceFrame references (already covered in operator validation)

	return issues
}
