package handlers

import (
	"crypto/subtle"
	"encoding/base64"
	"fmt"
	"net/http"
	"strings"
	"time"

	"quadis-backend-go/internal/config"
	"quadis-backend-go/internal/domain"
	"quadis-backend-go/internal/repository"
	"quadis-backend-go/internal/service/ota"
)

type OTAHandler struct {
	repo       repository.Repository
	otaService *ota.OTAService
	cfg        *config.Config
}

func NewOTAHandler(repo repository.Repository, otaService *ota.OTAService, cfg *config.Config) *OTAHandler {
	return &OTAHandler{
		repo:       repo,
		otaService: otaService,
		cfg:        cfg,
	}
}

type posCredentials struct {
	user      string
	password  string
	idContext string
}

func (h *OTAHandler) extractCredentials(r *http.Request, message map[string]interface{}) *posCredentials {
	// 1. From message POS
	if pos, ok := message["POS"].(map[string]interface{}); ok {
		requestor, _ := pos["RequestorID"].(map[string]interface{})
		var user, pass, idContext string
		if requestor != nil {
			user, _ = requestor["User"].(string)
			pass, _ = requestor["Password"].(string)
			idContext, _ = requestor["ID_Context"].(string)
		}
		if user == "" {
			user, _ = pos["Username"].(string)
		}
		if user == "" {
			user, _ = pos["User"].(string)
		}
		if pass == "" {
			pass, _ = pos["Password"].(string)
		}
		if idContext == "" {
			idContext, _ = pos["ID_Context"].(string)
		}
		if user != "" && pass != "" {
			return &posCredentials{user: user, password: pass, idContext: idContext}
		}
	}

	// 2. From HTTP Basic Auth
	authHeader := r.Header.Get("Authorization")
	if strings.HasPrefix(authHeader, "Basic ") {
		decoded, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(authHeader, "Basic "))
		if err == nil {
			parts := strings.SplitN(string(decoded), ":", 2)
			if len(parts) == 2 {
				return &posCredentials{user: parts[0], password: parts[1]}
			}
		}
	}

	return nil
}

func (h *OTAHandler) authenticate(r *http.Request, message map[string]interface{}) (int, string) {
	expectedUser := h.cfg.ResAvenueUsername
	expectedPass := h.cfg.ResAvenuePassword
	expectedContext := h.cfg.ResAvenueIDContext

	if expectedUser == "" || expectedPass == "" {
		return http.StatusServiceUnavailable, "Channel manager access is not configured on this server"
	}

	creds := h.extractCredentials(r, message)
	if creds == nil {
		return http.StatusUnauthorized, "Missing credentials: send POS.Username/Password or HTTP Basic auth"
	}

	userOk := subtle.ConstantTimeCompare([]byte(creds.user), []byte(expectedUser)) == 1
	passOk := subtle.ConstantTimeCompare([]byte(creds.password), []byte(expectedPass)) == 1
	contextOk := expectedContext == "" || subtle.ConstantTimeCompare([]byte(creds.idContext), []byte(expectedContext)) == 1

	if !userOk || !passOk || !contextOk {
		return http.StatusUnauthorized, "Authentication failed"
	}

	return http.StatusOK, ""
}

func (h *OTAHandler) statusResponse(root string, status string, remark string) map[string]interface{} {
	return map[string]interface{}{
		root: map[string]interface{}{
			"TimeStamp": time.Now().UTC().Format(time.RFC3339),
			"Target":    h.cfg.ResAvenueTarget,
			"Version":   "2.0",
			"Status":    status,
			"Remark":    remark,
		},
	}
}

// PropertyDetails handles OTA_HotelDetailsRQ -> /property-details
func (h *OTAHandler) PropertyDetails(w http.ResponseWriter, r *http.Request) {
	var body map[string]interface{}
	if err := ParseJSON(r, &body); err != nil {
		JSON(w, http.StatusOK, h.statusResponse("OTA_HotelDetailsRS", "Failure", "Request body must be a JSON object"))
		return
	}

	msg, _ := body["OTA_HotelDetailsRQ"].(map[string]interface{})
	if msg == nil {
		msg = body
	}

	if code, remark := h.authenticate(r, msg); code != http.StatusOK {
		JSON(w, code, h.statusResponse("OTA_HotelDetailsRS", "Failure", remark))
		return
	}

	hotelCode := 0
	if hc, ok := msg["HotelCode"].(float64); ok {
		hotelCode = int(hc)
	} else if pID, ok := msg["PropertyId"].(string); ok {
		var n int
		if _, err := fmt.Sscanf(pID, "prop-%d", &n); err == nil {
			hotelCode = n
		}
	}

	details, err := h.otaService.GetPropertyDetails(r.Context(), hotelCode)
	if err != nil {
		JSON(w, http.StatusOK, h.statusResponse("OTA_HotelDetailsRS", "Failure", err.Error()))
		return
	}

	JSON(w, http.StatusOK, map[string]interface{}{
		"OTA_HotelDetailsRS": map[string]interface{}{
			"TimeStamp": time.Now().UTC().Format(time.RFC3339),
			"Status":    "Success",
			"Target":    h.cfg.ResAvenueTarget,
			"Version":   "2.0",
			"Hotel":     details,
		},
	})
}

func (h *OTAHandler) posEcho(msg map[string]interface{}) map[string]interface{} {
	pos, _ := msg["POS"].(map[string]interface{})
	if pos == nil {
		return map[string]interface{}{
			"Username":   "",
			"Password":   "",
			"ID_Context": "",
		}
	}
	user, _ := pos["Username"].(string)
	if user == "" {
		if req, ok := pos["RequestorID"].(map[string]interface{}); ok {
			user, _ = req["User"].(string)
		}
	}
	idCtx, _ := pos["ID_Context"].(string)
	if idCtx == "" {
		if req, ok := pos["RequestorID"].(map[string]interface{}); ok {
			idCtx, _ = req["ID_Context"].(string)
		}
	}
	return map[string]interface{}{
		"Username":   user,
		"Password":   "",
		"ID_Context": idCtx,
	}
}

// InventoryFetch handles OTA_HotelInventoryRQ -> /inventory/fetch
func (h *OTAHandler) InventoryFetch(w http.ResponseWriter, r *http.Request) {
	var body map[string]interface{}
	if err := ParseJSON(r, &body); err != nil {
		JSON(w, http.StatusOK, h.statusResponse("OTA_HotelInventoryRS", "Failure", "Invalid JSON"))
		return
	}

	msg, _ := body["OTA_HotelInventoryRQ"].(map[string]interface{})
	if msg == nil {
		msg = body
	}

	if code, remark := h.authenticate(r, msg); code != http.StatusOK {
		JSON(w, code, h.statusResponse("OTA_HotelInventoryRS", "Failure", remark))
		return
	}

	hotelCode := 0
	if hc, ok := msg["HotelCode"].(float64); ok {
		hotelCode = int(hc)
	}

	var roomCodes []int
	if rcs, ok := msg["InvCodes"].([]interface{}); ok {
		for _, v := range rcs {
			if n, ok := v.(float64); ok {
				roomCodes = append(roomCodes, int(n))
			}
		}
	} else if rc, ok := msg["InvCode"].(float64); ok {
		roomCodes = append(roomCodes, int(rc))
	}

	startDate, _ := msg["Start"].(string)
	if startDate == "" {
		startDate, _ = msg["StartDate"].(string)
	}
	endDate, _ := msg["End"].(string)
	if endDate == "" {
		endDate, _ = msg["EndDate"].(string)
	}
	if endDate == "" {
		endDate = startDate
	}

	data, err := h.otaService.FetchInventory(r.Context(), hotelCode, roomCodes, startDate, endDate)
	if err != nil {
		JSON(w, http.StatusOK, h.statusResponse("OTA_HotelInventoryRS", "Failure", err.Error()))
		return
	}

	data["POS"] = h.posEcho(msg)
	data["TimeStamp"] = time.Now().UTC().Format(time.RFC3339)
	data["EchoToken"] = msg["EchoToken"]

	JSON(w, http.StatusOK, map[string]interface{}{
		"OTA_HotelInventoryRS": data,
	})
}

// RateFetch handles OTA_HotelRateRQ -> /rates/fetch
func (h *OTAHandler) RateFetch(w http.ResponseWriter, r *http.Request) {
	var body map[string]interface{}
	if err := ParseJSON(r, &body); err != nil {
		JSON(w, http.StatusOK, h.statusResponse("OTA_HotelRateRS", "Failure", "Invalid JSON"))
		return
	}

	msg, _ := body["OTA_HotelRateRQ"].(map[string]interface{})
	if msg == nil {
		msg = body
	}

	if code, remark := h.authenticate(r, msg); code != http.StatusOK {
		JSON(w, code, h.statusResponse("OTA_HotelRateRS", "Failure", remark))
		return
	}

	hotelCode := 0
	if hc, ok := msg["HotelCode"].(float64); ok {
		hotelCode = int(hc)
	}

	var rateCodes []int
	if rcs, ok := msg["RateCodes"].([]interface{}); ok {
		for _, v := range rcs {
			if n, ok := v.(float64); ok {
				rateCodes = append(rateCodes, int(n))
			}
		}
	} else if rc, ok := msg["RateCode"].(float64); ok {
		rateCodes = append(rateCodes, int(rc))
	}

	startDate, _ := msg["Start"].(string)
	if startDate == "" {
		startDate, _ = msg["StartDate"].(string)
	}
	endDate, _ := msg["End"].(string)
	if endDate == "" {
		endDate, _ = msg["EndDate"].(string)
	}
	if endDate == "" {
		endDate = startDate
	}

	data, err := h.otaService.FetchRates(r.Context(), hotelCode, rateCodes, startDate, endDate)
	if err != nil {
		JSON(w, http.StatusOK, h.statusResponse("OTA_HotelRateRS", "Failure", err.Error()))
		return
	}

	data["POS"] = h.posEcho(msg)
	data["TimeStamp"] = time.Now().UTC().Format(time.RFC3339)
	data["EchoToken"] = msg["EchoToken"]

	JSON(w, http.StatusOK, map[string]interface{}{
		"OTA_HotelRateRS": data,
	})
}

// InventoryUpdate handles OTA_HotelInvCountNotifRQ -> /inventory/update
func (h *OTAHandler) InventoryUpdate(w http.ResponseWriter, r *http.Request) {
	var body map[string]interface{}
	if err := ParseJSON(r, &body); err != nil {
		JSON(w, http.StatusOK, h.statusResponse("OTA_HotelInvCountNotifRS", "Failure", "Invalid JSON"))
		return
	}

	msg, _ := body["OTA_HotelInvCountNotifRQ"].(map[string]interface{})
	if msg == nil {
		msg = body
	}

	if code, remark := h.authenticate(r, msg); code != http.StatusOK {
		JSON(w, code, h.statusResponse("OTA_HotelInvCountNotifRS", "Failure", remark))
		return
	}

	hotelCode := 0
	if hc, ok := msg["HotelCode"].(float64); ok {
		hotelCode = int(hc)
	}

	roomCode := 0
	if rc, ok := msg["InvTypeCode"].(float64); ok {
		roomCode = int(rc)
	}

	startDate, _ := msg["StartDate"].(string)
	endDate, _ := msg["EndDate"].(string)
	if endDate == "" {
		endDate = startDate
	}

	var count *int
	if c, ok := msg["InvCount"].(float64); ok {
		ci := int(c)
		count = &ci
	}

	stopSell, _ := msg["StopSell"].(bool)
	closeArr, _ := msg["CloseOnArrival"].(bool)
	closeDep, _ := msg["CloseOnDeparture"].(bool)
	cutOff := 0
	if co, ok := msg["CutOff"].(float64); ok {
		cutOff = int(co)
	}

	_, err := h.otaService.UpdateInventory(r.Context(), hotelCode, roomCode, startDate, endDate, count, stopSell, closeArr, closeDep, cutOff)
	if err != nil {
		JSON(w, http.StatusOK, h.statusResponse("OTA_HotelInvCountNotifRS", "Failure", err.Error()))
		return
	}

	JSON(w, http.StatusOK, h.statusResponse("OTA_HotelInvCountNotifRS", "Success", "Inventory updated successfully"))
}

// RateUpdate handles OTA_HotelRateAmountNotifRQ -> /rates/update
func (h *OTAHandler) RateUpdate(w http.ResponseWriter, r *http.Request) {
	var body map[string]interface{}
	if err := ParseJSON(r, &body); err != nil {
		JSON(w, http.StatusOK, h.statusResponse("OTA_HotelRateAmountNotifRS", "Failure", "Invalid JSON"))
		return
	}

	msg, _ := body["OTA_HotelRateAmountNotifRQ"].(map[string]interface{})
	if msg == nil {
		msg = body
	}

	if code, remark := h.authenticate(r, msg); code != http.StatusOK {
		JSON(w, code, h.statusResponse("OTA_HotelRateAmountNotifRS", "Failure", remark))
		return
	}

	ratePlanCode := 0
	if rpc, ok := msg["RatePlanCode"].(float64); ok {
		ratePlanCode = int(rpc)
	}

	startDate, _ := msg["StartDate"].(string)
	endDate, _ := msg["EndDate"].(string)
	if endDate == "" {
		endDate = startDate
	}

	var single, double, triple, quad, extraAdult, extraChild *float64
	if s, ok := msg["Single"].(float64); ok {
		single = &s
	}
	if d, ok := msg["Double"].(float64); ok {
		double = &d
	}
	if t, ok := msg["Triple"].(float64); ok {
		triple = &t
	}
	if q, ok := msg["Quad"].(float64); ok {
		quad = &q
	}
	if ea, ok := msg["ExtraAdult"].(float64); ok {
		extraAdult = &ea
	}
	if ec, ok := msg["ExtraChild"].(float64); ok {
		extraChild = &ec
	}

	var minStay, maxStay *int
	if ms, ok := msg["MinStay"].(float64); ok {
		msi := int(ms)
		minStay = &msi
	}
	if ms, ok := msg["MaxStay"].(float64); ok {
		msi := int(ms)
		maxStay = &msi
	}
	stopSell, _ := msg["StopSell"].(bool)

	_, err := h.otaService.UpdateRates(r.Context(), ratePlanCode, startDate, endDate, single, double, triple, quad, extraAdult, extraChild, minStay, maxStay, stopSell)
	if err != nil {
		JSON(w, http.StatusOK, h.statusResponse("OTA_HotelRateAmountNotifRS", "Failure", err.Error()))
		return
	}

	JSON(w, http.StatusOK, h.statusResponse("OTA_HotelRateAmountNotifRS", "Success", "Rates updated successfully"))
}

// BookingPull handles OTA_HotelResNotifRQ -> /bookings/pull
func (h *OTAHandler) BookingPull(w http.ResponseWriter, r *http.Request) {
	var body map[string]interface{}
	if err := ParseJSON(r, &body); err != nil {
		JSON(w, http.StatusOK, h.statusResponse("OTA_HotelResNotifRS", "Failure", "Invalid JSON"))
		return
	}

	msg, _ := body["OTA_HotelResNotifRQ"].(map[string]interface{})
	if msg == nil {
		msg = body
	}

	if code, remark := h.authenticate(r, msg); code != http.StatusOK {
		JSON(w, code, h.statusResponse("OTA_HotelResNotifRS", "Failure", remark))
		return
	}

	hotelCode := 0
	if hc, ok := msg["HotelCode"].(float64); ok {
		hotelCode = int(hc)
	}

	fromDate, _ := msg["FromDate"].(string)
	toDate, _ := msg["ToDate"].(string)

	bookings, err := h.otaService.PullBookings(r.Context(), hotelCode, fromDate, toDate)
	if err != nil {
		JSON(w, http.StatusOK, h.statusResponse("OTA_HotelResNotifRS", "Failure", err.Error()))
		return
	}

	var resList []map[string]interface{}
	for _, b := range bookings {
		resList = append(resList, map[string]interface{}{
			"BookingCode": b.BookingCode,
			"Status":      b.CMResStatus,
			"CheckIn":     b.CheckIn,
			"CheckOut":    b.CheckOut,
			"GuestName":   b.GuestName,
			"GuestPhone":  b.GuestPhone,
			"TotalAmount": b.TotalAmount,
		})
	}

	JSON(w, http.StatusOK, map[string]interface{}{
		"OTA_HotelResNotifRS": map[string]interface{}{
			"TimeStamp":         time.Now().UTC().Format(time.RFC3339),
			"Status":            "Success",
			"Target":            h.cfg.ResAvenueTarget,
			"HotelReservations": resList,
		},
	})
}

// Codes handles GET /api/ota/codes
func (h *OTAHandler) Codes(w http.ResponseWriter, r *http.Request) {
	if code, remark := h.authenticate(r, map[string]interface{}{}); code != http.StatusOK {
		JSON(w, code, map[string]interface{}{"success": false, "error": remark})
		return
	}

	props, err := h.repo.GetProperties(r.Context())
	if err != nil {
		ErrorJSON(w, http.StatusInternalServerError, err.Error())
		return
	}

	var out []map[string]interface{}
	for _, p := range props {
		rooms, _ := h.repo.GetRoomTypesByPropertyID(r.Context(), p.ID)
		var roomList []map[string]interface{}
		for _, rm := range rooms {
			var ratePlans []map[string]interface{}
			for _, plan := range []domain.MealPlan{domain.MealPlanRoomOnly, domain.MealPlanWithBreakfast, domain.MealPlanAllMealsIncluded} {
				ratePlans = append(ratePlans, map[string]interface{}{
					"RatePlanCode": ota.RatePlanOTACode(p.ID, rm.Slug, plan),
					"RatePlanName": string(plan),
				})
			}
			roomList = append(roomList, map[string]interface{}{
				"InvTypeCode": ota.RoomOTACode(p.ID, rm.Slug),
				"RoomName":    rm.Name,
				"TotalUnits":  rm.TotalUnits,
				"RatePlans":   ratePlans,
			})
		}

		var hotelCode int
		_, _ = fmt.Sscanf(p.ID, "prop-%d", &hotelCode)

		out = append(out, map[string]interface{}{
			"HotelCode": hotelCode,
			"HotelName": p.Name,
			"Slug":      p.Slug,
			"RoomTypes": roomList,
		})
	}

	JSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"data":    out,
	})
}

// Dispatch handles /resavenue and / generic message routing
func (h *OTAHandler) Dispatch(w http.ResponseWriter, r *http.Request) {
	var body map[string]interface{}
	if err := ParseJSON(r, &body); err != nil {
		JSON(w, http.StatusOK, map[string]interface{}{
			"OTA_ErrorRS": map[string]interface{}{
				"TimeStamp": time.Now().UTC().Format(time.RFC3339),
				"Status":    "Failure",
				"Remark":    "Invalid JSON request body",
			},
		})
		return
	}

	for k := range body {
		switch k {
		case "OTA_HotelDetailsRQ":
			h.PropertyDetails(w, r)
			return
		case "OTA_HotelInventoryRQ":
			h.InventoryFetch(w, r)
			return
		case "OTA_HotelInvCountNotifRQ":
			h.InventoryUpdate(w, r)
			return
		case "OTA_HotelRateRQ":
			h.RateFetch(w, r)
			return
		case "OTA_HotelRateAmountNotifRQ":
			h.RateUpdate(w, r)
			return
		case "OTA_HotelResNotifRQ":
			h.BookingPull(w, r)
			return
		}
	}

	JSON(w, http.StatusOK, map[string]interface{}{
		"OTA_ErrorRS": map[string]interface{}{
			"TimeStamp": time.Now().UTC().Format(time.RFC3339),
			"Status":    "Failure",
			"Remark":    "Unrecognised message. Expected one of: OTA_HotelDetailsRQ, OTA_HotelInventoryRQ, OTA_HotelInvCountNotifRQ, OTA_HotelRateRQ, OTA_HotelRateAmountNotifRQ, OTA_HotelResNotifRQ",
		},
	})
}
