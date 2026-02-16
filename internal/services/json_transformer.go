package services

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/mh-airlines/afs-engine/internal/models"
	"github.com/mh-airlines/afs-engine/internal/utils"
	log "github.com/sirupsen/logrus"
)

type JSONTransformer struct{}

func NewJSONTransformer() *JSONTransformer {
	return &JSONTransformer{}
}

type JSONFIDASM struct {
	Header  HeaderJSON    `json:"header"`
	PayLoad []PayLoadJSON `json:"payload"`
}

type HeaderJSON struct {
	MsgCode     string `json:"msgCode"`
	MsgSubtype  string `json:"msgSubtype"`
	MsgVersion  string `json:"msgVersion"`
	RefID       string `json:"refID"`
	MsgLength   int    `json:"msgLength"`
	MsgCount    int    `json:"msgCount"`
	EndOfChain  int    `json:"endOfChain"`
	MsgTimeSent string `json:"msgTimeSent"`
}

type PayLoadJSON struct {
	Header             string   `json:"header"`
	ActionCode         string   `json:"actionCode"`
	AFSkey             string   `json:"afsKey"`
	FlightNo           string   `json:"flightNo"`
	Leg                string   `json:"leg"`
	STAD               string   `json:"stad"`
	OfficialFlightDate string   `json:"officialFlightDate"`
	AircraftType       string   `json:"aircraftType"`
	ServiceClass       string   `json:"serviceClass"`
	AircraftOperator   string   `json:"aircraftOperator"`
	ServiceTypeCode    string   `json:"serviceTypeCode"`
	CodeShareFlight    []string `json:"codeShareFlight"`
	FlightMode         string   `json:"flightMode"`
	ModeSequence       string   `json:"modeSequence"`
	CategoryCode       string   `json:"categoryCode"`
	Station1           string   `json:"station1"`
	Station2           string   `json:"station2"`
	Station3           string   `json:"station3"`
	Station4           string   `json:"station4"`
	Station5           string   `json:"station5"`
	Station6           string   `json:"station6"`
	STD1               string   `json:"std1"`
	STD2               string   `json:"std2"`
	STD3               string   `json:"std3"`
	STD4               string   `json:"std4"`
	STD5               string   `json:"std5"`
	STA2               string   `json:"sta2"`
	STA3               string   `json:"sta3"`
	STA4               string   `json:"sta4"`
	STA5               string   `json:"sta5"`
	STA6               string   `json:"sta6"`
	SpFIndicator       string   `json:"spfIndicator"`
	SchOpenTimeC       string   `json:"schOpenTimeC"`
	SchCloseTimeC      string   `json:"schCloseTimeC"`
	SchOpenTimeL       string   `json:"schOpenTimeL"`
	SchCloseTimeL      string   `json:"schCloseTimeL"`
	SchBoardTimeL      string   `json:"schBoardTimeL"`
	SchFCTimeL         string   `json:"schFcTimeL"`
	StandCode          string   `json:"standCode"`
	LoungeCode         string   `json:"loungeCode"`
	AcftRegnNo         string   `json:"acftRegnNo"`
	Memo               string   `json:"memo"`
	TerminalID         string   `json:"terminalId"`
	SuffixDisp         string   `json:"suffixDisp"`
	CheckInType        string   `json:"checkInType"`
	IslandsAlloc       string   `json:"islandsAlloc"`
	DeskAlloc          string   `json:"deskAlloc"`
	IslandStatus       string   `json:"islandStatus"`
	ActIslandOpenTime  string   `json:"actIslandOpenTime"`
	ActIslandCloseTime string   `json:"actIslandCloseTime"`
}

func (t *JSONTransformer) TransformToJSON(afsRecords []models.ActiveFlight, batchID string) (string, error) {
	log.WithFields(log.Fields{
		"recordCount": len(afsRecords),
		"batchId":     batchID,
	}).Info("Transforming AFS records to FIDASM1 JSON")

	payloads := make([]PayLoadJSON, len(afsRecords))
	for i, afs := range afsRecords {
		payloads[i] = t.transformFlight(afs)
	}

	msgTimeSent := time.Now().Format("20060102150405")

	jsonDoc := JSONFIDASM{
		Header: HeaderJSON{
			MsgCode:     "FIDASM1",
			MsgSubtype:  "",
			MsgVersion:  "1",
			RefID:       "",
			MsgLength:   0,
			MsgCount:    len(afsRecords),
			EndOfChain:  0,
			MsgTimeSent: msgTimeSent,
		},
		PayLoad: payloads,
	}

	output, err := json.MarshalIndent(jsonDoc, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal JSON: %w", err)
	}

	log.WithField("batchId", batchID).Debug("FIDASM1 JSON transformation completed")
	return string(output), nil
}

// transformFlight transforms single AFS record to PayLoadJSON element
func (t *JSONTransformer) transformFlight(afs models.ActiveFlight) PayLoadJSON {
	// Format timestamps as YYYYMMDDHHmm
	formatTimestamp := func(dateStr, timeStr string) string {
		if dateStr == "" || timeStr == "" {
			return ""
		}
		combined := dateStr + timeStr
		formatted := ""
		for _, char := range combined {
			if char >= '0' && char <= '9' {
				formatted += string(char)
			}
		}
		return formatted
	}

	legValue := ""
	switch afs.MovementType {
	case "DEPARTURE":
		legValue = "D"
	case "ARRIVAL":
		legValue = "A"
	default:
		legValue = string(afs.LegSequence + 64)
	}

	var stad string
	if afs.MovementType == "ARRIVAL" {
		stad = formatTimestamp(utils.FormatDate(afs.FlightDate), afs.STA)
	} else {
		stad = formatTimestamp(utils.FormatDate(afs.FlightDate), afs.STD)
	}

	officialFlightDate := formatTimestamp(utils.FormatDate(afs.FlightDate), afs.STD)
	std1 := formatTimestamp(utils.FormatDate(afs.FlightDate), afs.STD)
	sta2 := formatTimestamp(utils.FormatDate(afs.FlightDate), afs.STA)

	codeshareFlights := []string{}
	if len(afs.CodeshareFlights) > 0 {
		codeshareFlights = afs.CodeshareFlights
	}

	categoryCode := afs.CategoryCode
	timings := afs.OperationalTimings

	suffixDisp := "N"
	if afs.OperationalSuffix != "" && afs.ShowSuffix {
		suffixDisp = "Y"
	}

	return PayLoadJSON{
		Header:             "AFS",
		ActionCode:         "NEW",
		AFSkey:             afs.ID.Hex(),
		FlightNo:           afs.FlightNo,
		Leg:                legValue,
		STAD:               stad,
		OfficialFlightDate: officialFlightDate,
		AircraftType:       afs.AircraftType,
		ServiceClass:       "",
		AircraftOperator:   afs.FlightOwner,
		ServiceTypeCode:    afs.ServiceType,
		CodeShareFlight:    codeshareFlights,
		FlightMode:         "0",
		ModeSequence:       "0",
		CategoryCode:       categoryCode,
		Station1:           afs.DepartureStation,
		Station2:           afs.ArrivalStation,
		Station3:           "",
		Station4:           "",
		Station5:           "",
		Station6:           "",
		STD1:               std1,
		STD2:               "",
		STD3:               "",
		STD4:               "",
		STD5:               "",
		STA2:               sta2,
		STA3:               "",
		STA4:               "",
		STA5:               "",
		STA6:               "",
		SpFIndicator:       "N",
		SchOpenTimeC:       timings.SchOpenTimeC,
		SchCloseTimeC:      timings.SchCloseTimeC,
		SchOpenTimeL:       timings.SchOpenTimeL,
		SchCloseTimeL:      timings.SchCloseTimeL,
		SchBoardTimeL:      timings.SchBoardTimeL,
		SchFCTimeL:         timings.SchFCTimeL,
		StandCode:          "",
		LoungeCode:         "",
		AcftRegnNo:         afs.TailNo,
		Memo:               "",
		TerminalID:         afs.PassengerTerminalDep,
		SuffixDisp:         suffixDisp,
		CheckInType:        "C",
		IslandsAlloc:       "",
		DeskAlloc:          "",
		IslandStatus:       "",
		ActIslandOpenTime:  "",
		ActIslandCloseTime: "",
	}
}

func (t *JSONTransformer) CreateManifest(batchID string, afsRecords []models.ActiveFlight, apiResponse *models.APIResponse) map[string]interface{} {
	flightIDs := make([]string, len(afsRecords))
	for i, afs := range afsRecords {
		flightIDs[i] = afs.ID.Hex()
	}

	manifest := map[string]interface{}{
		"batchId":     batchID,
		"timestamp":   time.Now().Format(time.RFC3339),
		"flightCount": len(afsRecords),
		"flightIds":   flightIDs,
		"apiStatus":   "pending",
		"format":      "FIDASM1",
	}

	if apiResponse != nil {
		manifest["apiStatus"] = "completed"
		manifest["apiResponse"] = map[string]interface{}{
			"statusCode": apiResponse.StatusCode,
			"message":    apiResponse.Message,
			"timestamp":  apiResponse.Timestamp.Format(time.RFC3339),
			"accepted":   apiResponse.Accepted,
			"rejected":   apiResponse.Rejected,
		}
	}

	return manifest
}
