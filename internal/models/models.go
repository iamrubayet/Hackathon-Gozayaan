package models

import (
	"time"
)

type User struct {
	ID        string    `gorm:"primaryKey;type:uuid;default:gen_random_uuid()" json:"id"`
	Name      string    `gorm:"not null" json:"name"`
	Phone     string    `gorm:"uniqueIndex;not null" json:"phone"`
	Password  string    `gorm:"not null" json:"-"`
	UserType  string    `gorm:"not null" json:"user_type"` // "rider" or "driver"
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type Driver struct {
	ID            string    `gorm:"primaryKey;type:uuid;default:gen_random_uuid()" json:"id"`
	UserID        string    `gorm:"not null;index" json:"user_id"`
	VehicleNumber string    `gorm:"not null" json:"vehicle_number"`
	VehicleModel  string    `json:"vehicle_model"`
	LicenseNumber string    `gorm:"not null" json:"license_number"`
	IsAvailable   bool      `gorm:"default:true" json:"is_available"`
	CurrentLat    float64   `json:"current_lat"`
	CurrentLng    float64   `json:"current_lng"`
	Rating        float64   `gorm:"default:5.0" json:"rating"`
	TotalRides    int       `gorm:"default:0" json:"total_rides"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

type Ride struct {
	ID             string     `gorm:"primaryKey;type:uuid;default:gen_random_uuid()" json:"id"`
	RiderID        string     `gorm:"not null;index" json:"rider_id"`
	DriverID       string     `gorm:"index" json:"driver_id,omitempty"`
	PickupLat      float64    `gorm:"not null" json:"pickup_lat"`
	PickupLng      float64    `gorm:"not null" json:"pickup_lng"`
	PickupAddress  string     `json:"pickup_address"`
	DropoffLat     float64    `gorm:"not null" json:"dropoff_lat"`
	DropoffLng     float64    `gorm:"not null" json:"dropoff_lng"`
	DropoffAddress string     `json:"dropoff_address"`
	Status         string     `gorm:"not null;default:'requested'" json:"status"` // requested, accepted, started, completed, cancelled
	Fare           float64    `json:"fare"`
	Distance       float64    `json:"distance"` // in km
	Duration       int        `json:"duration"` // in minutes
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
	CompletedAt    *time.Time `json:"completed_at,omitempty"`
}

type Rating struct {
	ID        string    `gorm:"primaryKey;type:uuid;default:gen_random_uuid()" json:"id"`
	RideID    string    `gorm:"not null;uniqueIndex" json:"ride_id"`
	RiderID   string    `gorm:"not null;index" json:"rider_id"`
	DriverID  string    `gorm:"not null;index" json:"driver_id"`
	Rating    int       `gorm:"not null" json:"rating"` // 1-5
	Comment   string    `json:"comment"`
	CreatedAt time.Time `json:"created_at"`
}
