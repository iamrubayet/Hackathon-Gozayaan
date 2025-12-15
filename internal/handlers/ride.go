package handlers

import (
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"time"

	"rickshaw-app/internal/config"
	"rickshaw-app/internal/middleware"
	"rickshaw-app/internal/models"

	"github.com/go-chi/chi/v5"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

type RideHandler struct {
	db  *gorm.DB
	rdb *redis.Client
	cfg *config.Config
}

func NewRideHandler(db *gorm.DB, rdb *redis.Client, cfg *config.Config) *RideHandler {
	return &RideHandler{db: db, rdb: rdb, cfg: cfg}
}

type CreateRideRequest struct {
	PickupLat      float64 `json:"pickup_lat"`
	PickupLng      float64 `json:"pickup_lng"`
	PickupAddress  string  `json:"pickup_address"`
	DropoffLat     float64 `json:"dropoff_lat"`
	DropoffLng     float64 `json:"dropoff_lng"`
	DropoffAddress string  `json:"dropoff_address"`
}

type CreateFareRequest struct {
	PickupLat      float64 `json:"pickup_lat"`
	PickupLng      float64 `json:"pickup_lng"`
	PickupAddress  string  `json:"pickup_address"`
	DropoffLat     float64 `json:"dropoff_lat"`
	DropoffLng     float64 `json:"dropoff_lng"`
	DropoffAddress string  `json:"dropoff_address"`
}

type RateRideRequest struct {
	Rating  int    `json:"rating"`
	Comment string `json:"comment"`
}

func (h *RideHandler) CreateRide(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	userType := middleware.GetUserType(r.Context())

	if userType != "rider" {
		respondJSON(w, http.StatusForbidden, map[string]string{"error": "only riders can create rides"})
		return
	}

	var req CreateRideRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request"})
		return
	}

	distance := haversine(req.PickupLat, req.PickupLng, req.DropoffLat, req.DropoffLng)
	duration := int(distance / 0.5)
	fare := calculateFare(distance)

	ride := &models.Ride{
		RiderID:        userID,
		PickupLat:      req.PickupLat,
		PickupLng:      req.PickupLng,
		PickupAddress:  req.PickupAddress,
		DropoffLat:     req.DropoffLat,
		DropoffLng:     req.DropoffLng,
		DropoffAddress: req.DropoffAddress,
		Status:         "requested",
		Fare:           fare,
		Distance:       distance,
		Duration:       duration,
	}

	if err := h.db.Create(ride).Error; err != nil {
		respondJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to create ride"})
		return
	}

	h.logHistory(ride.ID, "requested", fmt.Sprintf("created by rider %s", userID))

	respondJSON(w, http.StatusCreated, ride)
}

func (h *RideHandler) GetRides(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	userType := middleware.GetUserType(r.Context())

	var rides []models.Ride
	query := h.db.Order("created_at DESC")

	if userType == "rider" {
		query = query.Where("rider_id = ?", userID)
	} else if userType == "driver" {
		var driver models.Driver
		if err := h.db.Where("user_id = ?", userID).First(&driver).Error; err != nil {
			respondJSON(w, http.StatusNotFound, map[string]string{"error": "driver profile not found"})
			return
		}

		// Get rides assigned to this driver OR available requested rides
		var assignedRides []models.Ride
		if err := h.db.Order("created_at DESC").Where("driver_id = ?", driver.ID).Find(&assignedRides).Error; err != nil {
			respondJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to fetch rides"})
			return
		}

		var availableRides []models.Ride
		if err := h.db.Order("created_at DESC").Where("status = ? AND driver_id IS NULL", "requested").Find(&availableRides).Error; err != nil {
			respondJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to fetch rides"})
			return
		}

		// Filter available rides based on distance from driver's current location
		filteredRides := assignedRides // Always include assigned rides
		if driver.CurrentLat != 0 && driver.CurrentLng != 0 {
			for _, ride := range availableRides {
				distance := haversine(driver.CurrentLat, driver.CurrentLng, ride.PickupLat, ride.PickupLng)
				if distance <= 10 {
					filteredRides = append(filteredRides, ride)
				}
			}
		}

		respondJSON(w, http.StatusOK, filteredRides)
		return
	}

	if err := query.Find(&rides).Error; err != nil {
		respondJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to fetch rides"})
		return
	}

	respondJSON(w, http.StatusOK, rides)
}

func (h *RideHandler) GetRide(w http.ResponseWriter, r *http.Request) {
	rideID := chi.URLParam(r, "id")

	var ride models.Ride
	if err := h.db.First(&ride, "id = ?", rideID).Error; err != nil {
		respondJSON(w, http.StatusNotFound, map[string]string{"error": "ride not found"})
		return
	}

	respondJSON(w, http.StatusOK, ride)
}

func (h *RideHandler) AcceptRide(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	userType := middleware.GetUserType(r.Context())
	rideID := chi.URLParam(r, "id")

	if userType != "driver" {
		respondJSON(w, http.StatusForbidden, map[string]string{"error": "only drivers can accept rides"})
		return
	}

	var driver models.Driver
	if err := h.db.Where("user_id = ?", userID).First(&driver).Error; err != nil {
		respondJSON(w, http.StatusNotFound, map[string]string{"error": "driver profile not found"})
		return
	}

	var ride models.Ride
	if err := h.db.First(&ride, "id = ?", rideID).Error; err != nil {
		respondJSON(w, http.StatusNotFound, map[string]string{"error": "ride not found"})
		return
	}

	if ride.Status != "requested" {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "ride already accepted or completed"})
		return
	}

	ride.DriverID = &driver.ID
	ride.Status = "accepted"

	if err := h.db.Save(&ride).Error; err != nil {
		respondJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to accept ride"})
		return
	}

	h.logHistory(ride.ID, "accepted", fmt.Sprintf("accepted by driver %s", driver.ID))

	driver.IsAvailable = false
	h.db.Save(&driver)

	respondJSON(w, http.StatusOK, ride)
}

func (h *RideHandler) StartRide(w http.ResponseWriter, r *http.Request) {
	rideID := chi.URLParam(r, "id")
	userID := middleware.GetUserID(r.Context())

	var driver models.Driver
	if err := h.db.Where("user_id = ?", userID).First(&driver).Error; err != nil {
		respondJSON(w, http.StatusNotFound, map[string]string{"error": "driver profile not found"})
		return
	}

	var ride models.Ride
	if err := h.db.First(&ride, "id = ?", rideID).Error; err != nil {
		respondJSON(w, http.StatusNotFound, map[string]string{"error": "ride not found"})
		return
	}

	if ride.DriverID == nil || *ride.DriverID != driver.ID {
		respondJSON(w, http.StatusForbidden, map[string]string{"error": "not authorized"})
		return
	}

	if ride.Status != "accepted" {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "ride must be accepted first"})
		return
	}

	ride.Status = "started"

	if err := h.db.Save(&ride).Error; err != nil {
		respondJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to start ride"})
		return
	}

	h.logHistory(ride.ID, "started", fmt.Sprintf("started by driver %s", driver.ID))

	respondJSON(w, http.StatusOK, ride)
}

func (h *RideHandler) CompleteRide(w http.ResponseWriter, r *http.Request) {
	rideID := chi.URLParam(r, "id")
	userID := middleware.GetUserID(r.Context())

	var driver models.Driver
	if err := h.db.Where("user_id = ?", userID).First(&driver).Error; err != nil {
		respondJSON(w, http.StatusNotFound, map[string]string{"error": "driver profile not found"})
		return
	}

	var ride models.Ride
	if err := h.db.First(&ride, "id = ?", rideID).Error; err != nil {
		respondJSON(w, http.StatusNotFound, map[string]string{"error": "ride not found"})
		return
	}

	if ride.DriverID == nil || *ride.DriverID != driver.ID {
		respondJSON(w, http.StatusForbidden, map[string]string{"error": "not authorized"})
		return
	}

	if ride.Status != "started" {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "ride must be started first"})
		return
	}

	now := time.Now()
	ride.Status = "completed"
	ride.CompletedAt = &now

	if err := h.db.Save(&ride).Error; err != nil {
		respondJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to complete ride"})
		return
	}

	h.logHistory(ride.ID, "completed", fmt.Sprintf("completed by driver %s", driver.ID))

	driver.IsAvailable = true
	driver.TotalRides += 1
	h.db.Save(&driver)

	respondJSON(w, http.StatusOK, ride)
}

func (h *RideHandler) CancelRide(w http.ResponseWriter, r *http.Request) {
	rideID := chi.URLParam(r, "id")
	userID := middleware.GetUserID(r.Context())

	var ride models.Ride
	if err := h.db.First(&ride, "id = ?", rideID).Error; err != nil {
		respondJSON(w, http.StatusNotFound, map[string]string{"error": "ride not found"})
		return
	}

	if ride.RiderID != userID {
		var driver models.Driver
		if err := h.db.Where("user_id = ?", userID).First(&driver).Error; err == nil {
			if ride.DriverID == nil || *ride.DriverID != driver.ID {
				respondJSON(w, http.StatusForbidden, map[string]string{"error": "not authorized"})
				return
			}
		} else {
			respondJSON(w, http.StatusForbidden, map[string]string{"error": "not authorized"})
			return
		}
	}

	if ride.Status == "completed" || ride.Status == "cancelled" {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "cannot cancel completed or already cancelled ride"})
		return
	}

	ride.Status = "cancelled"

	if err := h.db.Save(&ride).Error; err != nil {
		respondJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to cancel ride"})
		return
	}

	note := fmt.Sprintf("cancelled by user %s", userID)
	if ride.DriverID != nil {
		note = fmt.Sprintf("cancelled; driver %s released", *ride.DriverID)
	}
	h.logHistory(ride.ID, "cancelled", note)

	if ride.DriverID != nil {
		var driver models.Driver
		if err := h.db.First(&driver, "id = ?", *ride.DriverID).Error; err == nil {
			driver.IsAvailable = true
			h.db.Save(&driver)
		}
	}

	respondJSON(w, http.StatusOK, ride)
}

func (h *RideHandler) RateRide(w http.ResponseWriter, r *http.Request) {
	rideID := chi.URLParam(r, "id")
	userID := middleware.GetUserID(r.Context())
	userType := middleware.GetUserType(r.Context())

	if userType != "rider" {
		respondJSON(w, http.StatusForbidden, map[string]string{"error": "only riders can rate rides"})
		return
	}

	var req RateRideRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request"})
		return
	}

	if req.Rating < 1 || req.Rating > 5 {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "rating must be between 1 and 5"})
		return
	}

	var ride models.Ride
	if err := h.db.First(&ride, "id = ?", rideID).Error; err != nil {
		respondJSON(w, http.StatusNotFound, map[string]string{"error": "ride not found"})
		return
	}

	if ride.RiderID != userID {
		respondJSON(w, http.StatusForbidden, map[string]string{"error": "not authorized"})
		return
	}

	if ride.Status != "completed" {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "can only rate completed rides"})
		return
	}

	if ride.DriverID == nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "ride not assigned to a driver"})
		return
	}

	rating := &models.Rating{
		RideID:   ride.ID,
		RiderID:  ride.RiderID,
		DriverID: *ride.DriverID,
		Rating:   req.Rating,
		Comment:  req.Comment,
	}

	if err := h.db.Create(rating).Error; err != nil {
		respondJSON(w, http.StatusConflict, map[string]string{"error": "ride already rated"})
		return
	}

	var driver models.Driver
	if err := h.db.First(&driver, "id = ?", *ride.DriverID).Error; err == nil {
		var avgRating float64
		h.db.Model(&models.Rating{}).Where("driver_id = ?", driver.ID).Select("AVG(rating)").Scan(&avgRating)
		driver.Rating = math.Round(avgRating*100) / 100
		h.db.Save(&driver)
	}

	respondJSON(w, http.StatusCreated, rating)
}

// logHistory records ride state transitions for admin visibility; best-effort (errors ignored).
func (h *RideHandler) logHistory(rideID, status, note string) {
	_ = h.db.Create(&models.RideHistory{RideID: rideID, Status: status, Note: note}).Error
}

func calculateFare(distance float64) float64 {
	baseFare := 20.0
	perKmRate := 30.0
	return math.Round((baseFare+distance*perKmRate)*100) / 100
}

func (h *RideHandler) CreateFare(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	userType := middleware.GetUserType(r.Context())

	if userType != "rider" {
		respondJSON(w, http.StatusForbidden, map[string]string{"error": "only riders can create rides"})
		return
	}

	var req CreateFareRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request"})
		return
	}

	distance := haversine(req.PickupLat, req.PickupLng, req.DropoffLat, req.DropoffLng)
	duration := int(distance / 0.5)
	fare := calculateFare(distance)

	ride := &models.Ride{
		RiderID:        userID,
		PickupLat:      req.PickupLat,
		PickupLng:      req.PickupLng,
		PickupAddress:  req.PickupAddress,
		DropoffLat:     req.DropoffLat,
		DropoffLng:     req.DropoffLng,
		DropoffAddress: req.DropoffAddress,
		Status:         "FARE_ESTIMATED",
		Fare:           fare,
		Distance:       distance,
		Duration:       duration,
	}

	respondJSON(w, http.StatusCreated, ride)
}
