package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"net/http"

	"rickshaw-app/internal/config"
	"rickshaw-app/internal/middleware"
	"rickshaw-app/internal/models"

	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

type DriverHandler struct {
	db  *gorm.DB
	rdb *redis.Client
	cfg *config.Config
}

func NewDriverHandler(db *gorm.DB, rdb *redis.Client, cfg *config.Config) *DriverHandler {
	return &DriverHandler{db: db, rdb: rdb, cfg: cfg}
}

type CreateDriverRequest struct {
	VehicleNumber string `json:"vehicle_number"`
	VehicleModel  string `json:"vehicle_model"`
	LicenseNumber string `json:"license_number"`
}

type UpdateLocationRequest struct {
	Lat float64 `json:"lat"`
	Lng float64 `json:"lng"`
}

type UpdateAvailabilityRequest struct {
	IsAvailable bool `json:"is_available"`
}

func (h *DriverHandler) CreateProfile(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	userType := middleware.GetUserType(r.Context())

	if userType != "driver" {
		respondJSON(w, http.StatusForbidden, map[string]string{"error": "only drivers can create driver profile"})
		return
	}

	var req CreateDriverRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request"})
		return
	}

	driver := &models.Driver{
		UserID:        userID,
		VehicleNumber: req.VehicleNumber,
		VehicleModel:  req.VehicleModel,
		LicenseNumber: req.LicenseNumber,
		IsAvailable:   true,
	}

	if err := h.db.Create(driver).Error; err != nil {
		respondJSON(w, http.StatusConflict, map[string]string{"error": "driver profile already exists"})
		return
	}

	respondJSON(w, http.StatusCreated, driver)
}

func (h *DriverHandler) GetProfile(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())

	var driver models.Driver
	if err := h.db.Where("user_id = ?", userID).First(&driver).Error; err != nil {
		respondJSON(w, http.StatusNotFound, map[string]string{"error": "driver profile not found"})
		return
	}

	respondJSON(w, http.StatusOK, driver)
}

func (h *DriverHandler) UpdateLocation(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())

	var req UpdateLocationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request"})
		return
	}

	var driver models.Driver
	if err := h.db.Where("user_id = ?", userID).First(&driver).Error; err != nil {
		respondJSON(w, http.StatusNotFound, map[string]string{"error": "driver profile not found"})
		return
	}

	driver.CurrentLat = req.Lat
	driver.CurrentLng = req.Lng

	if err := h.db.Save(&driver).Error; err != nil {
		respondJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to update location"})
		return
	}

	ctx := context.Background()
	h.rdb.GeoAdd(ctx, "drivers:locations", &redis.GeoLocation{
		Name:      driver.ID,
		Longitude: req.Lng,
		Latitude:  req.Lat,
	})

	respondJSON(w, http.StatusOK, driver)
}

func (h *DriverHandler) UpdateAvailability(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())

	var req UpdateAvailabilityRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request"})
		return
	}

	var driver models.Driver
	if err := h.db.Where("user_id = ?", userID).First(&driver).Error; err != nil {
		respondJSON(w, http.StatusNotFound, map[string]string{"error": "driver profile not found"})
		return
	}

	driver.IsAvailable = req.IsAvailable

	if err := h.db.Save(&driver).Error; err != nil {
		respondJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to update availability"})
		return
	}

	respondJSON(w, http.StatusOK, driver)
}

func (h *DriverHandler) GetNearbyDrivers(w http.ResponseWriter, r *http.Request) {
	lat := r.URL.Query().Get("lat")
	lng := r.URL.Query().Get("lng")

	if lat == "" || lng == "" {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "lat and lng are required"})
		return
	}

	var latF, lngF float64
	fmt.Sscanf(lat, "%f", &latF)
	fmt.Sscanf(lng, "%f", &lngF)

	var drivers []models.Driver
	if err := h.db.Where("is_available = ?", true).Find(&drivers).Error; err != nil {
		respondJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to fetch drivers"})
		return
	}

	type DriverWithDistance struct {
		models.Driver
		Distance float64 `json:"distance"`
	}

	nearbyDrivers := []DriverWithDistance{}
	for _, driver := range drivers {
		if driver.CurrentLat == 0 && driver.CurrentLng == 0 {
			continue
		}

		distance := haversine(latF, lngF, driver.CurrentLat, driver.CurrentLng)
		if distance <= 10 {
			nearbyDrivers = append(nearbyDrivers, DriverWithDistance{
				Driver:   driver,
				Distance: distance,
			})
		}
	}

	respondJSON(w, http.StatusOK, nearbyDrivers)
}

func haversine(lat1, lon1, lat2, lon2 float64) float64 {
	const R = 6371
	dLat := (lat2 - lat1) * math.Pi / 180
	dLon := (lon2 - lon1) * math.Pi / 180

	a := math.Sin(dLat/2)*math.Sin(dLat/2) +
		math.Cos(lat1*math.Pi/180)*math.Cos(lat2*math.Pi/180)*
			math.Sin(dLon/2)*math.Sin(dLon/2)

	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))
	return R * c
}
