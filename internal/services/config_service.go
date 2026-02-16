package services

import (
	"context"
	"fmt"
	"time"

	"github.com/mh-airlines/afs-engine/internal/models"
	log "github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// ConfigService manages operational configuration loading and caching
type ConfigService struct {
	db                *Database
	defaultConfig     *models.DefaultConfig
	airlineConfigs    map[primitive.ObjectID]*models.AirlineConfig // Keyed by airline ID
	airlineByCode     map[string]*models.Airline                   // Keyed by airline code (e.g., "MH")
	cacheLoadTime     time.Time
	cacheTTL          time.Duration
}

// NewConfigService creates a new configuration service
func NewConfigService(db *Database) *ConfigService {
	return &ConfigService{
		db:             db,
		airlineConfigs: make(map[primitive.ObjectID]*models.AirlineConfig),
		airlineByCode:  make(map[string]*models.Airline),
		cacheTTL:       5 * time.Minute, // Reload configs every 5 minutes
	}
}

// LoadConfigurations loads all operational configurations into cache
func (cs *ConfigService) LoadConfigurations(ctx context.Context) error {
	// Check if cache is still valid
	if cs.defaultConfig != nil && time.Since(cs.cacheLoadTime) < cs.cacheTTL {
		return nil
	}

	log.Info("Loading operational configurations...")

	// Load default config
	if err := cs.loadDefaultConfig(ctx); err != nil {
		return fmt.Errorf("failed to load default config: %w", err)
	}

	// Load airlines
	if err := cs.loadAirlines(ctx); err != nil {
		return fmt.Errorf("failed to load airlines: %w", err)
	}

	// Load airline-specific configs
	if err := cs.loadAirlineConfigs(ctx); err != nil {
		return fmt.Errorf("failed to load airline configs: %w", err)
	}

	cs.cacheLoadTime = time.Now()
	log.WithFields(log.Fields{
		"airlines":       len(cs.airlineByCode),
		"airlineConfigs": len(cs.airlineConfigs),
	}).Info("Operational configurations loaded successfully")

	return nil
}

// loadDefaultConfig loads the active default configuration
func (cs *ConfigService) loadDefaultConfig(ctx context.Context) error {
	collection := cs.db.GetRefCollection("default_config")

	filter := bson.M{"isActive": true}
	
	var config models.DefaultConfig
	err := collection.FindOne(ctx, filter).Decode(&config)
	if err != nil {
		return fmt.Errorf("no active default configuration found: %w", err)
	}

	cs.defaultConfig = &config
	log.WithField("version", config.Version).Info("Loaded default configuration")
	return nil
}

// loadAirlines loads all active airlines
func (cs *ConfigService) loadAirlines(ctx context.Context) error {
	collection := cs.db.GetRefCollection("airlines")

	filter := bson.M{"isActive": true}
	cursor, err := collection.Find(ctx, filter)
	if err != nil {
		return err
	}
	defer cursor.Close(ctx)

	var airlines []models.Airline
	if err := cursor.All(ctx, &airlines); err != nil {
		return err
	}

	// Build airline code -> airline mapping
	newCache := make(map[string]*models.Airline)
	for i := range airlines {
		newCache[airlines[i].Code] = &airlines[i]
	}

	cs.airlineByCode = newCache
	log.WithField("count", len(airlines)).Info("Loaded airlines")
	return nil
}

// loadAirlineConfigs loads all active airline-specific configurations
func (cs *ConfigService) loadAirlineConfigs(ctx context.Context) error {
	collection := cs.db.GetRefCollection("airline_config")

	filter := bson.M{"isActive": true}
	cursor, err := collection.Find(ctx, filter)
	if err != nil {
		return err
	}
	defer cursor.Close(ctx)

	var configs []models.AirlineConfig
	if err := cursor.All(ctx, &configs); err != nil {
		return err
	}

	// Build airline ID -> config mapping
	newCache := make(map[primitive.ObjectID]*models.AirlineConfig)
	for i := range configs {
		newCache[configs[i].AirlineID] = &configs[i]
	}

	cs.airlineConfigs = newCache
	log.WithField("count", len(configs)).Info("Loaded airline configurations")
	return nil
}

// GetConfigForAirline returns the configuration for a specific airline
// Returns airline-specific config if exists, otherwise returns default config
func (cs *ConfigService) GetConfigForAirline(airlineCode string) (*models.GateConfig, *models.CheckInConfig, error) {
	// Ensure configs are loaded
	if cs.defaultConfig == nil {
		return nil, nil, fmt.Errorf("configurations not loaded")
	}

	// Look up airline by code
	airline, exists := cs.airlineByCode[airlineCode]
	if !exists {
		// Airline not found, use default config
		log.WithField("airlineCode", airlineCode).Debug("Airline not found, using default config")
		return &cs.defaultConfig.Gate, &cs.defaultConfig.CheckIn, nil
	}

	// Check if airline has specific config
	airlineConfig, hasOverride := cs.airlineConfigs[airline.ID]
	if hasOverride {
		log.WithField("airlineCode", airlineCode).Debug("Using airline-specific config")
		return &airlineConfig.Gate, &airlineConfig.CheckIn, nil
	}

	// Use default config
	log.WithField("airlineCode", airlineCode).Debug("No airline override, using default config")
	return &cs.defaultConfig.Gate, &cs.defaultConfig.CheckIn, nil
}

// CalculateOperationalTimings calculates check-in and gate timings for a flight
func (cs *ConfigService) CalculateOperationalTimings(
	flightDate time.Time,
	std string, // Format: "HHmm" or "HH:mm"
	airlineCode string,
	categoryCode string, // "I" for International, "D" for Domestic
	movementType string, // "DEPARTURE" or "ARRIVAL"
) (*models.OperationalTimings, error) {

	// Only calculate timings for departures from homeStation
	if movementType != "DEPARTURE" {
		return &models.OperationalTimings{}, nil
	}

	// Get configuration for this airline
	gateConfig, checkInConfig, err := cs.GetConfigForAirline(airlineCode)
	if err != nil {
		return nil, err
	}

	// Parse STD time
	stdTime, err := parseTimeString(std)
	if err != nil {
		return nil, fmt.Errorf("failed to parse STD time %s: %w", std, err)
	}

	// Combine flight date with STD time
	stdDateTime := time.Date(
		flightDate.Year(),
		flightDate.Month(),
		flightDate.Day(),
		stdTime.Hour(),
		stdTime.Minute(),
		0, 0,
		flightDate.Location(),
	)

	// Determine if domestic or international
	isDomestic := categoryCode == "D"

	// Calculate check-in timings
	var checkInOpenOffset, checkInCloseOffset int
	if isDomestic {
		checkInOpenOffset = checkInConfig.DomesticOpen
		checkInCloseOffset = checkInConfig.DomesticClose
	} else {
		checkInOpenOffset = checkInConfig.InternationalOpen
		checkInCloseOffset = checkInConfig.InternationalClose
	}

	schOpenTimeC := stdDateTime.Add(-time.Duration(checkInOpenOffset) * time.Minute)
	schCloseTimeC := stdDateTime.Add(-time.Duration(checkInCloseOffset) * time.Minute)

	// Calculate gate timings
	var gateOpenOffset, gateCloseOffset, boardingOffset, finalCallOffset int
	if isDomestic {
		gateOpenOffset = gateConfig.DomesticOpen
		gateCloseOffset = gateConfig.DomesticClose
		boardingOffset = gateConfig.DomesticBoarding
		finalCallOffset = gateConfig.DomesticFinalCall
	} else {
		gateOpenOffset = gateConfig.InternationalOpen
		gateCloseOffset = gateConfig.InternationalClose
		boardingOffset = gateConfig.InternationalBoarding
		finalCallOffset = gateConfig.InternationalFinalCall
	}

	schOpenTimeL := stdDateTime.Add(-time.Duration(gateOpenOffset) * time.Minute)
	schCloseTimeL := stdDateTime.Add(-time.Duration(gateCloseOffset) * time.Minute)
	schBoardTimeL := stdDateTime.Add(-time.Duration(boardingOffset) * time.Minute)
	schFCTimeL := stdDateTime.Add(-time.Duration(finalCallOffset) * time.Minute)

	// Format all times as YYYYMMDDHHmm
	timings := &models.OperationalTimings{
		SchOpenTimeC:  formatToYYYYMMDDHHmm(schOpenTimeC),
		SchCloseTimeC: formatToYYYYMMDDHHmm(schCloseTimeC),
		SchOpenTimeL:  formatToYYYYMMDDHHmm(schOpenTimeL),
		SchCloseTimeL: formatToYYYYMMDDHHmm(schCloseTimeL),
		SchBoardTimeL: formatToYYYYMMDDHHmm(schBoardTimeL),
		SchFCTimeL:    formatToYYYYMMDDHHmm(schFCTimeL),
	}

	log.WithFields(log.Fields{
		"airlineCode":  airlineCode,
		"categoryCode": categoryCode,
		"std":          std,
		"checkInOpen":  timings.SchOpenTimeC,
		"checkInClose": timings.SchCloseTimeC,
		"gateOpen":     timings.SchOpenTimeL,
		"boarding":     timings.SchBoardTimeL,
	}).Debug("Calculated operational timings")

	return timings, nil
}

// parseTimeString parses time string in formats "HHmm" or "HH:mm"
func parseTimeString(timeStr string) (time.Time, error) {
	// Remove any colons
	cleanTime := ""
	for _, char := range timeStr {
		if char >= '0' && char <= '9' {
			cleanTime += string(char)
		}
	}

	// Ensure we have exactly 4 digits
	if len(cleanTime) != 4 {
		return time.Time{}, fmt.Errorf("invalid time format: %s", timeStr)
	}

	hour := 0
	minute := 0
	_, err := fmt.Sscanf(cleanTime, "%02d%02d", &hour, &minute)
	if err != nil {
		return time.Time{}, err
	}

	// Return a time with just hour and minute set
	return time.Date(0, 1, 1, hour, minute, 0, 0, time.UTC), nil
}

// formatToYYYYMMDDHHmm formats a time to YYYYMMDDHHmm string
func formatToYYYYMMDDHHmm(t time.Time) string {
	return t.Format("200601021504")
}