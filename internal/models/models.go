package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type MasterFlight struct {
	ID                primitive.ObjectID `bson:"_id,omitempty"`
	CreationDate      time.Time          `bson:"creationDate"`
	FileName          string             `bson:"fileName"`
	BucketPath        string             `bson:"bucketPath"`
	FlightOwner       string             `bson:"flightOwner"`
	OperationalSuffix string             `bson:"operationalSuffix"`
	FlightNo          string             `bson:"flightNo"`
	SeasonID          string             `bson:"seasonId"`
	ItineraryVarID    int                `bson:"itineraryVarId"`
	StartDate         time.Time          `bson:"startDate"`
	EndDate           time.Time          `bson:"endDate"`
	Frequency         string             `bson:"frequency"`
	FreqRate          string             `bson:"freqRate"`
	IATAServiceType   string             `bson:"iataServiceTypeCode"`
	ScheduleStatus    string             `bson:"scheduleStatus"`
	Stations          []Station          `bson:"stations"`
	AdditionalFields  AdditionalFields   `bson:"additionalFields"`
	MessageType       string             `bson:"MessageType"`
	IsAdhoc           bool               `bson:"isAdhoc"`
	HomeStation       string             `bson:"homeStation"`
	SourceTracking    SourceTracking     `bson:"sourceTracking"`
	Codeshares        []Codeshare        `bson:"-"`
}

type Codeshare struct {
	ID                primitive.ObjectID `bson:"_id,omitempty"`
	MasterFlightRef   primitive.ObjectID `bson:"masterflightRef"`
	CSFSKey           string             `bson:"csfsKey"`
	SeasonID          string             `bson:"seasonId"`
	FlightOwner       string             `bson:"flightOwner"`
	FlightNo          string             `bson:"flightNo"`
	CSStartDate       time.Time          `bson:"csStartDate"`
	CSEndDate         time.Time          `bson:"csEndDate"`
	Frequency         string             `bson:"frequency"`
	FreqRate          string             `bson:"freqRate"`
	CodeshareFlightNo []string           `bson:"codeshareFlightNo"`
	Sector            string             `bson:"sector"`
	SourceTracking    SourceTracking     `bson:"sourceTracking"`
	CreatedAt         time.Time          `bson:"createdAt"`
	UpdatedAt         time.Time          `bson:"updatedAt"`
}

// Station represents a flight leg
type Station struct {
	DepartureStation         string    `bson:"departureStation"`
	PassengerTerminalDep     string    `bson:"passengerTerminalDep"`
	STD                      string    `bson:"std"`
	UTCLocalTimeVariationDep string    `bson:"utcLocalTimeVariationDep"`
	CD                       int       `bson:"cd"`
	ArrivalStation           string    `bson:"arrivalStation"`
	PassengerTerminalArr     string    `bson:"passengerTerminalArr"`
	STA                      string    `bson:"sta"`
	CA                       int       `bson:"ca"` // Day change arrival
	UTCLocalTimeVariationArr string    `bson:"utcLocalTimeVariationArr"`
	IATASubTypeCode          string    `bson:"iataSubTypeCode"`
	AircraftOwner            string    `bson:"aircraftOwner"`
	TailNo                   string    `bson:"tailNo"`
	AircraftConfiguration    string    `bson:"aircraftConfiguration"`
	OnwardFlight             string    `bson:"onwardFlight"`
	CreatedAt                time.Time `bson:"createdAt"`
	UpdatedAt                time.Time `bson:"updatedAt"`
}

// AdditionalFields contains extra MFS metadata
type AdditionalFields struct {
	BilateralInformation string `bson:"bilateralInformation"`
	RecordSerialNumber   string `bson:"recordSerialNumber"`
}

// SourceTracking tracks data provenance
type SourceTracking struct {
	DataSource     string    `bson:"dataSource"`
	Version        int       `bson:"version"`
	CreatedBy      Timestamp `bson:"createdBy"`
	LastModifiedBy Timestamp `bson:"lastModifiedBy,omitempty"`
	AllowedSources []string  `bson:"allowedSources"`
}

// Timestamp represents a timestamp with source
type Timestamp struct {
	Source    string    `bson:"source"`
	Timestamp time.Time `bson:"timestamp"`
}

// ActiveFlight represents the Active Flight Schedule record
type ActiveFlight struct {
	ID                       primitive.ObjectID `bson:"_id"`
	FlightNo                 string             `bson:"flightNo"`
	FlightOwner              string             `bson:"flightOwner"`
	OperationalSuffix        string             `bson:"operationalSuffix"`
	ShowSuffix				bool                `bson:"showSuffix"`
	FlightDate               time.Time          `bson:"flightDate"`
	LegSequence              int                `bson:"legSequence"`
	DepartureStation         string             `bson:"departureStation"`
	ArrivalStation           string             `bson:"arrivalStation"`
	PassengerTerminalDep     string             `bson:"passengerTerminalDep"`
	PassengerTerminalArr     string             `bson:"passengerTerminalArr"`
	STD                      string             `bson:"std"`
	STA                      string             `bson:"sta"`
	UTCLocalTimeVariationDep string             `bson:"utcLocalTimeVariationDep"`
	UTCLocalTimeVariationArr string             `bson:"utcLocalTimeVariationArr"`
	DayChangeDeparture       int                `bson:"dayChangeDeparture"`
	DayChangeArrival         int                `bson:"dayChangeArrival"`
	AircraftType             string             `bson:"aircraftType"`
	AircraftOwner            string             `bson:"aircraftOwner"`
	TailNo                   string             `bson:"tailNo"`
	AircraftConfiguration    string             `bson:"aircraftConfiguration"`
	ServiceType              string             `bson:"serviceType"`
	OnwardFlight             string             `bson:"onwardFlight"`
	CodeshareFlights         []string           `bson:"codeshareFlights,omitempty"`
	HomeStation              string             `bson:"homeStation"`
	MovementType             string             `bson:"movementType"`
	CategoryCode             string             `bson:"categoryCode"`
	OperationalTimings       OperationalTimings `bson:"operationalTimings,omitempty"`
	SourceMFSID              primitive.ObjectID `bson:"sourceMFSId"`
	SeasonID                 string             `bson:"seasonId"`
	ItineraryVarID           int                `bson:"itineraryVarId"`
	DeliveryStatus           string             `bson:"deliveryStatus"`
	DeliveryAttempts         int                `bson:"deliveryAttempts"`
	DeliveredAt              *time.Time         `bson:"deliveredAt,omitempty"`
	SentXMLBatchID           string             `bson:"sentXMLBatchId,omitempty"`
	APIResponse              *APIResponse       `bson:"apiResponse,omitempty"`
	LastErrorMessage         string             `bson:"lastErrorMessage,omitempty"`
	LastErrorAt              *time.Time         `bson:"lastErrorAt,omitempty"`
	ExpiresAt                time.Time          `bson:"expiresAt"`
	CreatedAt                time.Time          `bson:"createdAt"`
	UpdatedAt                time.Time          `bson:"updatedAt"`
}

type OperationalTimings struct {
	// Check-in timings
	SchOpenTimeC  string `bson:"schOpenTimeC,omitempty" json:"schOpenTimeC,omitempty"`
	SchCloseTimeC string `bson:"schCloseTimeC,omitempty" json:"schCloseTimeC,omitempty"`
	
	// Gate timings
	SchOpenTimeL  string `bson:"schOpenTimeL,omitempty" json:"schOpenTimeL,omitempty"`
	SchCloseTimeL string `bson:"schCloseTimeL,omitempty" json:"schCloseTimeL,omitempty"`
	SchBoardTimeL string `bson:"schBoardTimeL,omitempty" json:"schBoardTimeL,omitempty"`
	SchFCTimeL    string `bson:"schFcTimeL,omitempty" json:"schFcTimeL,omitempty"`
}

type APIResponse struct {
	StatusCode int       `bson:"statusCode"`
	Message    string    `bson:"message"`
	Timestamp  time.Time `bson:"timestamp"`
	Accepted   int       `bson:"accepted"`
	Rejected   int       `bson:"rejected"`
	Errors     []string  `bson:"errors,omitempty"`
}

type GenerationStats struct {
	MFSRecordsQueried   int           `json:"mfsRecordsQueried"`
	AFSRecordsGenerated int           `json:"afsRecordsGenerated"`
	Errors              int           `json:"errors"`
	StartTime           time.Time     `json:"startTime"`
	EndTime             time.Time     `json:"endTime"`
	Duration            time.Duration `json:"duration"`
}

// DeliveryStats tracks API delivery statistics
type DeliveryStats struct {
	TotalBatches      int `json:"totalBatches"`
	SuccessfulBatches int `json:"successfulBatches"`
	FailedBatches     int `json:"failedBatches"`
	TotalFlights      int `json:"totalFlights"`
	DeliveredFlights  int `json:"deliveredFlights"`
	FailedFlights     int `json:"failedFlights"`
}

// DeliveryResult represents the result of a batch delivery
type DeliveryResult struct {
	BatchID         string       `json:"batchId"`
	Success         bool         `json:"success"`
	TotalRecords    int          `json:"totalRecords"`
	AcceptedRecords int          `json:"acceptedRecords"`
	RejectedRecords int          `json:"rejectedRecords"`
	Errors          []string     `json:"errors"`
	APIResponse     *APIResponse `json:"apiResponse,omitempty"`
}

// Airport represents the IATA airport information
type Airport struct {
	ID               primitive.ObjectID `bson:"_id,omitempty"`
	CityName         string             `bson:"cityName"`
	IATAAirportCode  string             `bson:"iataAirportCode"`
	ICAOAirportCode  string             `bson:"icaoAirportCode"`
	AirportShortName string             `bson:"airportShortName"`
	AirportName      string             `bson:"airportName"`
	Language         []string           `bson:"language"`
	IsActive         bool               `bson:"isActive"`
	CityTranslation  []interface{}      `bson:"cityTranslation"`
	CountryCode      string             `bson:"countryCode"`
	UTCVariation     string             `bson:"utcVariation"`
}