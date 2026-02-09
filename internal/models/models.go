package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// MasterFlight represents the Master Flight Schedule record
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
	SourceTracking    SourceTracking     `bson:"sourceTracking"`
	Codeshares        []Codeshare        `bson:"-"` // Not stored in MFS, populated from codeshares collection
}

// Codeshare represents a codeshare flight record
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
	CodeshareFlights         []string           `bson:"codeshareFlights,omitempty"` // NEW: Array of codeshare flight numbers
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
	ExpiresAt                time.Time          `bson:"expiresAt"` // TTL
	CreatedAt                time.Time          `bson:"createdAt"`
	UpdatedAt                time.Time          `bson:"updatedAt"`
}

// APIResponse stores the API delivery response
type APIResponse struct {
	StatusCode int       `bson:"statusCode"`
	Message    string    `bson:"message"`
	Timestamp  time.Time `bson:"timestamp"`
	Accepted   int       `bson:"accepted"`
	Rejected   int       `bson:"rejected"`
	Errors     []string  `bson:"errors,omitempty"`
}

// GenerationStats tracks AFS generation statistics
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