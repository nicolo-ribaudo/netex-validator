package engine

import (
	"fmt"
	"sync"

	"github.com/nicolo-ribaudo/netex-validator/validation/context"
)

// DataCollector interface for collecting data during validation
type DataCollector interface {
	// CollectFromFile collects data from a single file validation context
	CollectFromFile(ctx *context.ObjectValidationContext) error

	// GetCollectedData returns the collected data for cross-file validation
	GetCollectedData() interface{}

	// GetName returns the name of this data collector
	GetName() string

	// Reset clears all collected data
	Reset()
}

// DataCollectorRegistry manages multiple data collectors
type DataCollectorRegistry struct {
	collectors map[string]DataCollector
	mutex      sync.RWMutex
}

// NewDataCollectorRegistry creates a new data collector registry
func NewDataCollectorRegistry() *DataCollectorRegistry {
	return &DataCollectorRegistry{
		collectors: make(map[string]DataCollector),
	}
}

// RegisterCollector adds a data collector to the registry
func (r *DataCollectorRegistry) RegisterCollector(collector DataCollector) {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	r.collectors[collector.GetName()] = collector
}

// CollectFromAllFiles runs all collectors on a validation context
func (r *DataCollectorRegistry) CollectFromAllFiles(ctx *context.ObjectValidationContext) error {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	for name, collector := range r.collectors {
		if err := collector.CollectFromFile(ctx); err != nil {
			return fmt.Errorf("data collector '%s' failed: %w", name, err)
		}
	}

	return nil
}

// GetCollector returns a specific data collector
func (r *DataCollectorRegistry) GetCollector(name string) DataCollector {
	r.mutex.RLock()
	defer r.mutex.RUnlock()
	return r.collectors[name]
}

// ResetAll resets all data collectors
func (r *DataCollectorRegistry) ResetAll() {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	for _, collector := range r.collectors {
		collector.Reset()
	}
}

// CommonDataCollector collects shared data from common files
type CommonDataCollector struct {
	name           string
	commonRepo     *context.CommonDataRepository
	collectedFiles map[string]bool
	mutex          sync.RWMutex
}

// NewCommonDataCollector creates a new common data collector
func NewCommonDataCollector() *CommonDataCollector {
	return &CommonDataCollector{
		name:           "CommonDataCollector",
		commonRepo:     context.NewCommonDataRepository(),
		collectedFiles: make(map[string]bool),
	}
}

// GetName returns the collector name
func (c *CommonDataCollector) GetName() string {
	return c.name
}

// CollectFromFile collects shared data from common files
func (c *CommonDataCollector) CollectFromFile(ctx *context.ObjectValidationContext) error {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	// Only collect from common files (files starting with '_')
	if !ctx.IsCommonFile {
		return nil
	}

	// Avoid collecting from the same file multiple times
	if c.collectedFiles[ctx.FileName] {
		return nil
	}

	// Collect operators
	for _, operator := range ctx.ServiceJourneys() {
		if operator.OperatorRef != nil {
			if op := ctx.GetOperator(operator.OperatorRef.Ref); op != nil {
				c.commonRepo.AddSharedOperator(op)
			}
		}
	}

	// Collect stop places
	for _, stopPlace := range ctx.StopPlaces() {
		c.commonRepo.AddSharedStopPlace(stopPlace)
	}

	// Mark file as processed
	c.collectedFiles[ctx.FileName] = true

	return nil
}

// GetCollectedData returns the common data repository
func (c *CommonDataCollector) GetCollectedData() interface{} {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	return c.commonRepo
}

// GetCommonDataRepository returns the typed common data repository
func (c *CommonDataCollector) GetCommonDataRepository() *context.CommonDataRepository {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	return c.commonRepo
}

// Reset clears all collected data
func (c *CommonDataCollector) Reset() {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.commonRepo = context.NewCommonDataRepository()
	c.collectedFiles = make(map[string]bool)
}

// NetworkTopologyCollector collects network topology information
type NetworkTopologyCollector struct {
	name            string
	lineRouteMap    map[string][]string // line ID -> route IDs
	routePatternMap map[string][]string // route ID -> journey pattern IDs
	patternStopMap  map[string][]string // journey pattern ID -> stop point IDs
	mutex           sync.RWMutex
}

// NewNetworkTopologyCollector creates a new network topology collector
func NewNetworkTopologyCollector() *NetworkTopologyCollector {
	return &NetworkTopologyCollector{
		name:            "NetworkTopologyCollector",
		lineRouteMap:    make(map[string][]string),
		routePatternMap: make(map[string][]string),
		patternStopMap:  make(map[string][]string),
	}
}

// GetName returns the collector name
func (n *NetworkTopologyCollector) GetName() string {
	return n.name
}

// CollectFromFile collects network topology from a file
func (n *NetworkTopologyCollector) CollectFromFile(ctx *context.ObjectValidationContext) error {
	n.mutex.Lock()
	defer n.mutex.Unlock()

	// Collect line-route relationships
	for _, route := range ctx.Routes() {
		if route.LineRef != nil {
			lineID := route.LineRef.Ref
			n.lineRouteMap[lineID] = append(n.lineRouteMap[lineID], route.ID)
		}
	}

	// Collect route-journey pattern relationships
	for _, jp := range ctx.JourneyPatterns() {
		if jp.RouteRef != nil {
			routeID := jp.RouteRef.Ref
			n.routePatternMap[routeID] = append(n.routePatternMap[routeID], jp.ID)
		}

		// Collect journey pattern-stop relationships
		if jp.PointsInSequence != nil {
			var stopIDs []string
			for _, stopPoint := range jp.PointsInSequence.StopPointInJourneyPatterns {
				if stopPoint.ScheduledStopPointRef != nil {
					stopIDs = append(stopIDs, stopPoint.ScheduledStopPointRef.Ref)
				}
			}
			n.patternStopMap[jp.ID] = stopIDs
		}
	}

	return nil
}

// GetCollectedData returns the collected topology data
func (n *NetworkTopologyCollector) GetCollectedData() interface{} {
	n.mutex.RLock()
	defer n.mutex.RUnlock()

	return struct {
		LineRouteMap    map[string][]string
		RoutePatternMap map[string][]string
		PatternStopMap  map[string][]string
	}{
		LineRouteMap:    n.lineRouteMap,
		RoutePatternMap: n.routePatternMap,
		PatternStopMap:  n.patternStopMap,
	}
}

// Reset clears all collected data
func (n *NetworkTopologyCollector) Reset() {
	n.mutex.Lock()
	defer n.mutex.Unlock()
	n.lineRouteMap = make(map[string][]string)
	n.routePatternMap = make(map[string][]string)
	n.patternStopMap = make(map[string][]string)
}

// GetRoutesForLine returns route IDs for a given line
func (n *NetworkTopologyCollector) GetRoutesForLine(lineID string) []string {
	n.mutex.RLock()
	defer n.mutex.RUnlock()
	return n.lineRouteMap[lineID]
}

// GetPatternsForRoute returns journey pattern IDs for a given route
func (n *NetworkTopologyCollector) GetPatternsForRoute(routeID string) []string {
	n.mutex.RLock()
	defer n.mutex.RUnlock()
	return n.routePatternMap[routeID]
}

// GetStopsForPattern returns stop point IDs for a given journey pattern
func (n *NetworkTopologyCollector) GetStopsForPattern(patternID string) []string {
	n.mutex.RLock()
	defer n.mutex.RUnlock()
	return n.patternStopMap[patternID]
}

// ServiceFrequencyCollector collects service frequency information for analysis
type ServiceFrequencyCollector struct {
	name              string
	lineServiceCounts map[string]int            // line ID -> number of service journeys
	routeServiceMap   map[string][]string       // route ID -> service journey IDs
	dailyServices     map[string]map[string]int // date -> line ID -> service count
	mutex             sync.RWMutex
}

// NewServiceFrequencyCollector creates a new service frequency collector
func NewServiceFrequencyCollector() *ServiceFrequencyCollector {
	return &ServiceFrequencyCollector{
		name:              "ServiceFrequencyCollector",
		lineServiceCounts: make(map[string]int),
		routeServiceMap:   make(map[string][]string),
		dailyServices:     make(map[string]map[string]int),
	}
}

// GetName returns the collector name
func (s *ServiceFrequencyCollector) GetName() string {
	return s.name
}

// CollectFromFile collects service frequency data from a file
func (s *ServiceFrequencyCollector) CollectFromFile(ctx *context.ObjectValidationContext) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	// Count services per line
	for _, sj := range ctx.ServiceJourneys() {
		if sj.LineRef != nil {
			lineID := sj.LineRef.Ref
			s.lineServiceCounts[lineID]++
		}

		if sj.JourneyPatternRef != nil {
			jp := ctx.GetJourneyPattern(sj.JourneyPatternRef.Ref)
			if jp != nil && jp.RouteRef != nil {
				routeID := jp.RouteRef.Ref
				s.routeServiceMap[routeID] = append(s.routeServiceMap[routeID], sj.ID)
			}
		}
	}

	// Collect dated service information
	for _, dsj := range ctx.DatedServiceJourneys() {
		if dsj.ServiceJourneyRef != nil && dsj.OperatingDayRef != nil {
			operatingDay := ctx.GetOperatingDay(dsj.OperatingDayRef.Ref)
			if operatingDay != nil {
				date := operatingDay.CalendarDate
				if s.dailyServices[date] == nil {
					s.dailyServices[date] = make(map[string]int)
				}

				// Find the line for this service journey
				sj := ctx.GetServiceJourney(dsj.ServiceJourneyRef.Ref)
				if sj != nil && sj.LineRef != nil {
					lineID := sj.LineRef.Ref
					s.dailyServices[date][lineID]++
				}
			}
		}
	}

	return nil
}

// GetCollectedData returns the collected frequency data
func (s *ServiceFrequencyCollector) GetCollectedData() interface{} {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	return struct {
		LineServiceCounts map[string]int
		RouteServiceMap   map[string][]string
		DailyServices     map[string]map[string]int
	}{
		LineServiceCounts: s.lineServiceCounts,
		RouteServiceMap:   s.routeServiceMap,
		DailyServices:     s.dailyServices,
	}
}

// Reset clears all collected data
func (s *ServiceFrequencyCollector) Reset() {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.lineServiceCounts = make(map[string]int)
	s.routeServiceMap = make(map[string][]string)
	s.dailyServices = make(map[string]map[string]int)
}

// GetServiceCountForLine returns the number of services for a line
func (s *ServiceFrequencyCollector) GetServiceCountForLine(lineID string) int {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	return s.lineServiceCounts[lineID]
}

// GetServicesForRoute returns service journey IDs for a route
func (s *ServiceFrequencyCollector) GetServicesForRoute(routeID string) []string {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	return s.routeServiceMap[routeID]
}

// GetDailyServiceCount returns service count for a line on a specific date
func (s *ServiceFrequencyCollector) GetDailyServiceCount(date, lineID string) int {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	if dateMap, exists := s.dailyServices[date]; exists {
		return dateMap[lineID]
	}
	return 0
}
