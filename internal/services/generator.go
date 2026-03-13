package services

import (
	"context"
	"fmt"
	"github.com/mh-airlines/afs-engine/internal/config"
	"github.com/mh-airlines/afs-engine/internal/models"
	"github.com/mh-airlines/afs-engine/internal/utils"
	log "github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/options"
	"strings"
	"time"
)

// MFSMatch wraps a MasterFlight with the UTC operating day it matched for.
// When a flight's local departure crosses midnight due to timezone offset,
// BaseDate will differ from the target date by the local day offset.
type MFSMatch struct {
	MFS      models.MasterFlight
	BaseDate time.Time // The UTC operating day this MFS matched for
}

// AFSGenerator handles AFS generation from MFS
type AFSGenerator struct {
	db            *Database
	config        *config.Config
	airportCache  map[string]*models.Airport
	cacheLoadTime time.Time
	configService *ConfigService
}

// NewAFSGenerator creates a new AFS generator
func NewAFSGenerator(db *Database, cfg *config.Config) *AFSGenerator {
	return &AFSGenerator{
		db:            db,
		config:        cfg,
		airportCache:  make(map[string]*models.Airport),
		configService: NewConfigService(db),
	}
}

func formatTimeToHHMM(timeStr string) string {
	return strings.ReplaceAll(timeStr, ":", "")
}

// loadAirportCache loads all airports from iata_airports collection into memory
func (g *AFSGenerator) loadAirportCache(ctx context.Context) error {
	// Reload cache every 24 hours or if empty
	if len(g.airportCache) > 0 && time.Since(g.cacheLoadTime) < 24*time.Hour {
		return nil
	}

	collection := g.db.GetRefCollection("iata_airports")

	cursor, err := collection.Find(ctx, bson.M{"isActive": true})
	if err != nil {
		return fmt.Errorf("failed to query airports: %w", err)
	}
	defer cursor.Close(ctx)

	var airports []models.Airport
	if err := cursor.All(ctx, &airports); err != nil {
		return fmt.Errorf("failed to decode airports: %w", err)
	}

	// Build cache indexed by IATA code
	newCache := make(map[string]*models.Airport)
	for i := range airports {
		newCache[airports[i].IATAAirportCode] = &airports[i]
	}

	g.airportCache = newCache
	g.cacheLoadTime = time.Now()

	log.WithField("airportCount", len(g.airportCache)).Info("Airport cache loaded")
	return nil
}

// determineCategoryCode determines if a flight is International (I) or Domestic (D)
func (g *AFSGenerator) determineCategoryCode(depStation, arrStation string) string {
	depAirport, depExists := g.airportCache[depStation]
	arrAirport, arrExists := g.airportCache[arrStation]

	// Default to International if airport info not found
	if !depExists || !arrExists {
		log.WithFields(log.Fields{
			"departure": depStation,
			"arrival":   arrStation,
			"depFound":  depExists,
			"arrFound":  arrExists,
		}).Warn("Airport not found in cache, defaulting to International")
		return "I"
	}

	// Compare country codes
	if depAirport.CountryCode == arrAirport.CountryCode {
		return "D" // Domestic - same country
	}

	return "I" // International - different countries
}

// GenerateAFS generates AFS records for target date
func (g *AFSGenerator) GenerateAFS(ctx context.Context, targetDate *time.Time) (*models.GenerationStats, error) {
	flightDate := utils.GetTodayDate()
	if targetDate != nil {
		flightDate = utils.NormalizeDate(*targetDate)
	}

	stats := &models.GenerationStats{
		StartTime: time.Now(),
	}

	log.WithField("date", utils.FormatDate(flightDate)).Info("Starting AFS generation")

	// Load airport cache before processing
	if err := g.loadAirportCache(ctx); err != nil {
		log.WithError(err).Warn("Failed to load airport cache, category codes may be inaccurate")
	}

	// Load operational configurations
	if err := g.configService.LoadConfigurations(ctx); err != nil {
		log.WithError(err).Warn("Failed to load operational configurations, timings may be missing")
	}

	// Phase 1: Query valid MFS records (local-date-aware)
	mfsMatches, err := g.queryValidMFSForLocalDate(ctx, flightDate)
	if err != nil {
		return stats, fmt.Errorf("failed to query MFS: %w", err)
	}

	stats.MFSRecordsQueried = len(mfsMatches)
	log.WithField("count", len(mfsMatches)).Info("Found valid MFS records")

	// Phase 2: Query codeshares for all MFS records
	mfsMatches, err = g.attachCodeshares(ctx, mfsMatches)
	if err != nil {
		log.WithError(err).Warn("Failed to attach codeshares, continuing without them")
	}

	// Phase 3: Process each MFS record — baseDate for internal calc, flightDate for local output
	for _, match := range mfsMatches {
		afsRecords := g.expandMFSToAFS(match.MFS, match.BaseDate, flightDate)

		for _, afs := range afsRecords {
			if err := g.upsertAFS(ctx, afs); err != nil {
				log.WithError(err).WithField("flightNo", match.MFS.FlightNo).Error("Failed to upsert AFS")
				stats.Errors++
				continue
			}
			stats.AFSRecordsGenerated++
		}
	}

	stats.EndTime = time.Now()
	stats.Duration = stats.EndTime.Sub(stats.StartTime)

	log.WithFields(log.Fields{
		"mfsRecords":   stats.MFSRecordsQueried,
		"afsGenerated": stats.AFSRecordsGenerated,
		"errors":       stats.Errors,
		"duration":     stats.Duration.Seconds(),
	}).Info("AFS generation completed")

	return stats, nil
}

// queryValidMFSForLocalDate queries MFS records whose local departure date matches the target date.
// It widens the DB query by ±1 day to catch flights where UTC-to-local timezone conversion
// shifts the operating day, then filters by frequency using the computed baseDate.
func (g *AFSGenerator) queryValidMFSForLocalDate(ctx context.Context, targetDate time.Time) ([]MFSMatch, error) {
	collection := g.db.GetCollection("master_flights")

	// Widen window by ±1 day to catch cross-midnight timezone shifts
	windowStart := targetDate.AddDate(0, 0, -1)
	windowEnd := targetDate.AddDate(0, 0, 1)

	filter := bson.M{
		"startDate":      bson.M{"$lte": windowEnd},
		"endDate":        bson.M{"$gte": windowStart},
		"scheduleStatus": "ACTIVE",
		"$or": []bson.M{
			{"deletedAt": nil},
			{"deletedAt": bson.M{"$exists": false}},
		},
	}

	cursor, err := collection.Find(ctx, filter)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var mfsRecords []models.MasterFlight
	if err := cursor.All(ctx, &mfsRecords); err != nil {
		return nil, err
	}

	// For each MFS, calculate the local day offset and determine the baseDate
	var matches []MFSMatch
	for _, mfs := range mfsRecords {
		// Use the first station's STD and UTC offset to determine the day shift
		offset := 0
		if len(mfs.Stations) > 0 {
			offset = utils.CalculateLocalDateOffset(
				mfs.Stations[0].STD,
				mfs.Stations[0].UTCLocalTimeVariationDep,
			)
		}

		// baseDate is the UTC operating day that produces a local departure on targetDate
		baseDate := targetDate.AddDate(0, 0, -offset)

		// Check that baseDate falls within the MFS validity period
		if !utils.IsWithinValidityPeriod(baseDate, mfs.StartDate, mfs.EndDate) {
			continue
		}

		// Check frequency against the baseDate (the actual UTC operating day)
		if !utils.MatchesFrequency(baseDate, mfs.Frequency, mfs.StartDate) {
			continue
		}

		matches = append(matches, MFSMatch{
			MFS:      mfs,
			BaseDate: baseDate,
		})
	}

	return matches, nil
}

// attachCodeshares queries and attaches codeshare information to MFS matches.
// Each match's BaseDate is used for codeshare date/frequency filtering.
func (g *AFSGenerator) attachCodeshares(ctx context.Context, matches []MFSMatch) ([]MFSMatch, error) {
	if len(matches) == 0 {
		return matches, nil
	}

	// Collect all MFS IDs
	mfsIDs := make([]primitive.ObjectID, len(matches))
	for i, m := range matches {
		mfsIDs[i] = m.MFS.ID
	}

	// Build a map from MFS ID to its baseDate for per-record filtering
	baseDateMap := make(map[primitive.ObjectID]time.Time)
	for _, m := range matches {
		baseDateMap[m.MFS.ID] = m.BaseDate
	}

	// Find the widest date range across all baseDates for the initial query
	minDate := matches[0].BaseDate
	maxDate := matches[0].BaseDate
	for _, m := range matches[1:] {
		if m.BaseDate.Before(minDate) {
			minDate = m.BaseDate
		}
		if m.BaseDate.After(maxDate) {
			maxDate = m.BaseDate
		}
	}

	// Query all codeshares for these MFS records within the date range
	collection := g.db.GetCollection("codeshares")
	filter := bson.M{
		"masterflightRef": bson.M{"$in": mfsIDs},
		"csStartDate":     bson.M{"$lte": maxDate},
		"csEndDate":       bson.M{"$gte": minDate},
	}

	cursor, err := collection.Find(ctx, filter)
	if err != nil {
		return matches, err
	}
	defer cursor.Close(ctx)

	var codeshares []models.Codeshare
	if err := cursor.All(ctx, &codeshares); err != nil {
		return matches, err
	}

	// Filter codeshares by frequency using each MFS record's baseDate
	codeshareMap := make(map[primitive.ObjectID][]models.Codeshare)
	for _, cs := range codeshares {
		baseDate, ok := baseDateMap[cs.MasterFlightRef]
		if !ok {
			continue
		}
		// Check codeshare validity and frequency against the baseDate
		if !utils.IsWithinValidityPeriod(baseDate, cs.CSStartDate, cs.CSEndDate) {
			continue
		}
		if utils.MatchesFrequency(baseDate, cs.Frequency, cs.CSStartDate) {
			codeshareMap[cs.MasterFlightRef] = append(codeshareMap[cs.MasterFlightRef], cs)
		}
	}

	// Attach codeshares to their corresponding MFS matches
	for i := range matches {
		if cs, found := codeshareMap[matches[i].MFS.ID]; found {
			matches[i].MFS.Codeshares = cs
			log.WithFields(log.Fields{
				"flightNo":       matches[i].MFS.FlightNo,
				"codeshareCount": len(cs),
			}).Debug("Attached codeshares to MFS")
		}
	}

	return matches, nil
}

// findMatchingCodeshares finds codeshares that match the given sector
func (g *AFSGenerator) findMatchingCodeshares(codeshares []models.Codeshare, sector string) []string {
	var matchingFlights []string

	for _, cs := range codeshares {
		if cs.Sector == sector {
			matchingFlights = append(matchingFlights, cs.CodeshareFlightNo...)
		}
	}

	return matchingFlights
}

// expandMFSToAFS expands MFS record into AFS records (one per leg that touches homeStation).
// baseDate is the UTC operating day used as flightDate; times remain in UTC for downstream compatibility.
func (g *AFSGenerator) expandMFSToAFS(mfs models.MasterFlight, baseDate time.Time, localDate time.Time) []models.ActiveFlight {
	flightDate := baseDate
	var afsRecords []models.ActiveFlight

	showSuffix := false
	airline, exists := g.configService.airlineByCode[mfs.FlightOwner]
	if exists {
		showSuffix = airline.ShowSuffix
		log.WithFields(log.Fields{
			"flightOwner": mfs.FlightOwner,
			"showSuffix":  showSuffix,
			"airlineID":   airline.ID.Hex(),
			"airlineName": airline.Name,
		}).Debug("Retrieved airline showSuffix setting")
	} else {
		log.WithField("flightOwner", mfs.FlightOwner).Warn("Airline not found in cache")
	}

	for i, station := range mfs.Stations {
		// Only create AFS if this leg touches the homeStation
		// Either departing from homeStation OR arriving at homeStation
		if mfs.HomeStation == "" ||
			(station.DepartureStation != mfs.HomeStation && station.ArrivalStation != mfs.HomeStation) {
			log.WithFields(log.Fields{
				"flightNo":    mfs.FlightNo,
				"homeStation": mfs.HomeStation,
				"departure":   station.DepartureStation,
				"arrival":     station.ArrivalStation,
				"legSeq":      i + 1,
			}).Debug("Skipping leg - does not touch homeStation")
			continue
		}

		afsObjectID := primitive.NewObjectID()

		expiryDate := utils.CalculateExpiryDate(flightDate, g.config.Storage.AFSTTLDays)
		now := time.Now()

		// Build sector string for this leg
		sector := fmt.Sprintf("%s %s", station.DepartureStation, station.ArrivalStation)

		// Find matching codeshare flights for this leg
		codeshareFlights := g.findMatchingCodeshares(mfs.Codeshares, sector)

		// Determine movement type based on homeStation
		movementType := ""
		if station.DepartureStation == mfs.HomeStation {
			movementType = "DEPARTURE"
		} else if station.ArrivalStation == mfs.HomeStation {
			movementType = "ARRIVAL"
		}

		// Determine category code (International/Domestic)
		categoryCode := g.determineCategoryCode(station.DepartureStation, station.ArrivalStation)

		// Calculate operational timings (only for departures) using UTC STD
		var opTimings models.OperationalTimings
		if movementType == "DEPARTURE" {
			timings, err := g.configService.CalculateOperationalTimings(
				flightDate,
				station.STD,
				mfs.FlightOwner,
				categoryCode,
				movementType,
			)
			if err != nil {
				log.WithError(err).WithFields(log.Fields{
					"flightNo":   mfs.FlightNo,
					"flightDate": flightDate,
					"std":        station.STD,
				}).Warn("Failed to calculate operational timings")
			} else if timings != nil {
				opTimings = *timings
			}
		}

		afs := models.ActiveFlight{
			ID:                       afsObjectID,
			FlightNo:                 mfs.FlightNo,
			FlightOwner:              mfs.FlightOwner,
			OperationalSuffix:        mfs.OperationalSuffix,
			ShowSuffix:               showSuffix,
			FlightDate:               flightDate,
			LegSequence:              i + 1,
			DepartureStation:         station.DepartureStation,
			ArrivalStation:           station.ArrivalStation,
			PassengerTerminalDep:     station.PassengerTerminalDep,
			PassengerTerminalArr:     station.PassengerTerminalArr,
			STD:                      station.STD,
			STA:                      station.STA,
			UTCLocalTimeVariationDep: station.UTCLocalTimeVariationDep,
			UTCLocalTimeVariationArr: station.UTCLocalTimeVariationArr,
			DayChangeDeparture:       station.CD,
			DayChangeArrival:         station.CA,
			AircraftType:             station.IATASubTypeCode,
			AircraftOwner:            station.AircraftOwner,
			TailNo:                   station.TailNo,
			AircraftConfiguration:    station.AircraftConfiguration,
			ServiceType:              mfs.IATAServiceType,
			OnwardFlight:             station.OnwardFlight,
			CodeshareFlights:         codeshareFlights,
			HomeStation:              mfs.HomeStation,
			MovementType:             movementType,
			CategoryCode:             categoryCode,
			OperationalTimings:       opTimings,
			SourceMFSID:              mfs.ID,
			SeasonID:                 mfs.SeasonID,
			ItineraryVarID:           mfs.ItineraryVarID,
			DeliveryStatus:           "PENDING",
			DeliveryAttempts:         0,
			ExpiresAt:                expiryDate,
			CreatedAt:                now,
			UpdatedAt:                now,
		}

		log.WithFields(log.Fields{
			"flightNo":     mfs.FlightNo,
			"homeStation":  mfs.HomeStation,
			"departure":    station.DepartureStation,
			"arrival":      station.ArrivalStation,
			"movementType": movementType,
			"categoryCode": categoryCode,
			"showSuffix":   showSuffix, // ← ADD THIS
			"hasTimings":   opTimings.SchOpenTimeC != "",
			"legSeq":       i + 1,
		}).Debug("Created AFS record for homeStation leg")

		afsRecords = append(afsRecords, afs)
	}

	return afsRecords
}

// upsertAFS inserts or updates AFS record (idempotent)
func (g *AFSGenerator) upsertAFS(ctx context.Context, afs models.ActiveFlight) error {
	collection := g.db.GetCollection("active_flights")

	filter := bson.M{"_id": afs.ID}

	stdFormatted := formatTimeToHHMM(afs.STD)
	staFormatted := formatTimeToHHMM(afs.STA)

	update := bson.M{
		"$set": bson.M{
			"flightNo":                 afs.FlightNo,
			"flightOwner":              afs.FlightOwner,
			"operationalSuffix":        afs.OperationalSuffix,
			"showSuffix":               afs.ShowSuffix, // ← ADD THIS
			"flightDate":               afs.FlightDate,
			"legSequence":              afs.LegSequence,
			"departureStation":         afs.DepartureStation,
			"arrivalStation":           afs.ArrivalStation,
			"passengerTerminalDep":     afs.PassengerTerminalDep,
			"passengerTerminalArr":     afs.PassengerTerminalArr,
			"std":                      stdFormatted,
			"sta":                      staFormatted,
			"utcLocalTimeVariationDep": afs.UTCLocalTimeVariationDep,
			"utcLocalTimeVariationArr": afs.UTCLocalTimeVariationArr,
			"dayChangeDeparture":       afs.DayChangeDeparture,
			"dayChangeArrival":         afs.DayChangeArrival,
			"aircraftType":             afs.AircraftType,
			"aircraftOwner":            afs.AircraftOwner,
			"tailNo":                   afs.TailNo,
			"aircraftConfiguration":    afs.AircraftConfiguration,
			"serviceType":              afs.ServiceType,
			"onwardFlight":             afs.OnwardFlight,
			"codeshareFlights":         afs.CodeshareFlights,
			"homeStation":              afs.HomeStation,
			"movementType":             afs.MovementType,
			"categoryCode":             afs.CategoryCode,
			"operationalTimings":       afs.OperationalTimings,
			"sourceMFSId":              afs.SourceMFSID,
			"seasonId":                 afs.SeasonID,
			"itineraryVarId":           afs.ItineraryVarID,
			"deliveryStatus":           afs.DeliveryStatus,
			"deliveryAttempts":         afs.DeliveryAttempts,
			"expiresAt":                afs.ExpiresAt,
			"updatedAt":                time.Now(),
		},
		"$setOnInsert": bson.M{
			"createdAt": time.Now(),
		},
	}

	opts := options.Update().SetUpsert(true)
	_, err := collection.UpdateOne(ctx, filter, update, opts)
	return err
}

// GetAFSForDelivery retrieves AFS records ready for delivery.
// Uses a date range [targetDate-1, targetDate] to include AFS records where
// flightDate is the previous UTC day due to local-date-aware generation.
func (g *AFSGenerator) GetAFSForDelivery(ctx context.Context, flightDate time.Time, status string) ([]models.ActiveFlight, error) {
	collection := g.db.GetCollection("active_flights")

	filter := bson.M{
		"flightDate": bson.M{
			"$gte": flightDate.AddDate(0, 0, -1),
			"$lte": flightDate,
		},
		"deliveryStatus": status,
	}

	cursor, err := collection.Find(ctx, filter)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var records []models.ActiveFlight
	if err := cursor.All(ctx, &records); err != nil {
		return nil, err
	}

	return records, nil
}

func (g *AFSGenerator) UpdateDeliveryStatus(ctx context.Context, afsIDs []primitive.ObjectID, status string, additionalData bson.M) error {
	collection := g.db.GetCollection("active_flights")

	filter := bson.M{"_id": bson.M{"$in": afsIDs}}

	update := bson.M{
		"$set": bson.M{
			"deliveryStatus": status,
			"updatedAt":      time.Now(),
		},
		"$inc": bson.M{"deliveryAttempts": 1},
	}

	if additionalData != nil {
		for k, v := range additionalData {
			update["$set"].(bson.M)[k] = v
		}
	}

	_, err := collection.UpdateMany(ctx, filter, update)
	return err
}
