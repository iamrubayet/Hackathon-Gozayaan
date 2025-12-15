package handlers

import (
	"encoding/json"
	"net/http"

	"rickshaw-app/internal/models"

	"github.com/go-chi/chi/v5"
	"gorm.io/gorm"
)

// adminBasicAuth enforces a hardcoded basic auth (admin/admin) for the minimal admin panel.
func adminBasicAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		username, password, ok := r.BasicAuth()
		if !ok || username != "admin" || password != "admin" {
			w.Header().Set("WWW-Authenticate", `Basic realm="admin"`)
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r)
	})
}

type AdminHandler struct {
	db *gorm.DB
}

func NewAdminHandler(db *gorm.DB) *AdminHandler {
	return &AdminHandler{db: db}
}

// RegisterAdminRoutes wires the admin endpoints under /admin.
func RegisterAdminRoutes(r chi.Router, handler *AdminHandler) {
	r.Group(func(r chi.Router) {
		r.Use(adminBasicAuth)

		r.Get("/", handler.AdminPage)
		r.Get("/api/rides", handler.ListRides)
		r.Get("/api/drivers", handler.ListDrivers)
		r.Get("/api/ratings", handler.ListRatings)
		r.Get("/api/users", handler.ListUsers)
		r.Get("/api/ride-history", handler.ListRideHistory)
	})
}

func (h *AdminHandler) AdminPage(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	page := `<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8" />
  <meta name="viewport" content="width=device-width, initial-scale=1.0" />
  <title>Admin Panel</title>
  <style>
    body { font-family: "Helvetica Neue", Arial, sans-serif; margin: 24px; background: #f7f7f7; color: #111; }
    h1 { margin-bottom: 16px; }
    section { margin-bottom: 32px; background: #fff; padding: 16px; border: 1px solid #e5e5e5; border-radius: 8px; box-shadow: 0 1px 2px rgba(0,0,0,0.04); }
    table { border-collapse: collapse; width: 100%; margin-top: 8px; font-size: 14px; }
    th, td { border: 1px solid #e5e5e5; padding: 8px; text-align: left; }
    th { background: #fafafa; }
    .grid { display: grid; gap: 16px; grid-template-columns: repeat(auto-fit, minmax(320px, 1fr)); }
    .badge { display: inline-block; padding: 2px 6px; border-radius: 4px; background: #eef2ff; color: #27327c; font-size: 12px; }
    .status-requested { background: #fff7e6; color: #8a5a00; }
    .status-accepted { background: #e6f7ff; color: #005b96; }
    .status-started { background: #e6fffa; color: #006d5b; }
    .status-completed { background: #f0fff4; color: #276749; }
    .status-cancelled { background: #fff5f5; color: #9b2c2c; }
  </style>
</head>
<body>
  <h1>Admin Panel</h1>
  <div class="grid">
    <section>
      <h2>Rides</h2>
      <div id="rides"></div>
    </section>
    <section>
      <h2>Drivers</h2>
      <div id="drivers"></div>
    </section>
    <section>
      <h2>Ratings</h2>
      <div id="ratings"></div>
    </section>
    <section>
      <h2>Users</h2>
      <div id="users"></div>
    </section>
    <section>
      <h2>Ride History</h2>
      <div id="history"></div>
    </section>
  </div>
  <script>
    const endpoints = [
      { id: 'rides', url: '/admin/api/rides', map: rows => rows.map(r => ({ id: r.id, rider_id: r.rider_id, driver_id: r.driver_id, status: r.status, fare: r.fare, distance: r.distance, created_at: r.created_at })) },
      { id: 'drivers', url: '/admin/api/drivers', map: rows => rows.map(d => ({ id: d.id, user_id: d.user_id, vehicle_number: d.vehicle_number, is_available: d.is_available, rating: d.rating, total_rides: d.total_rides })) },
      { id: 'ratings', url: '/admin/api/ratings', map: rows => rows.map(r => ({ id: r.id, ride_id: r.ride_id, driver_id: r.driver_id, rider_id: r.rider_id, rating: r.rating, comment: r.comment })) },
      { id: 'users', url: '/admin/api/users', map: rows => rows.map(u => ({ id: u.id, name: u.name, phone: u.phone, user_type: u.user_type })) },
      { id: 'history', url: '/admin/api/ride-history', map: rows => rows.map(h => ({ id: h.id, ride_id: h.ride_id, status: h.status, note: h.note, created_at: h.created_at })) },
    ];

	function renderTable(id, rows) {
	  const container = document.getElementById(id);
	  if (!rows.length) { container.innerHTML = '<p>No data</p>'; return; }
	  const cols = Object.keys(rows[0]);
	  const thead = '<tr>' + cols.map(c => '<th>' + c + '</th>').join('') + '</tr>';
	  const tbody = rows.map(r => '<tr>' + cols.map(c => '<td>' + (r[c] ?? '') + '</td>').join('') + '</tr>').join('');
	  container.innerHTML = '<table><thead>' + thead + '</thead><tbody>' + tbody + '</tbody></table>';
	}

    endpoints.forEach(({ id, url, map }) => {
      fetch(url).then(r => r.json()).then(data => renderTable(id, map(data))).catch(() => {
        document.getElementById(id).innerHTML = '<p style="color:red">Failed to load</p>';
      });
    });
  </script>
</body>
</html>`

	w.Write([]byte(page))
}

func (h *AdminHandler) ListRides(w http.ResponseWriter, r *http.Request) {
	var rides []models.Ride
	if err := h.db.Order("created_at DESC").Find(&rides).Error; err != nil {
		http.Error(w, "failed to fetch rides", http.StatusInternalServerError)
		return
	}
	writeJSON(w, rides)
}

func (h *AdminHandler) ListDrivers(w http.ResponseWriter, r *http.Request) {
	var drivers []models.Driver
	if err := h.db.Order("created_at DESC").Find(&drivers).Error; err != nil {
		http.Error(w, "failed to fetch drivers", http.StatusInternalServerError)
		return
	}
	writeJSON(w, drivers)
}

func (h *AdminHandler) ListRatings(w http.ResponseWriter, r *http.Request) {
	var ratings []models.Rating
	if err := h.db.Order("created_at DESC").Find(&ratings).Error; err != nil {
		http.Error(w, "failed to fetch ratings", http.StatusInternalServerError)
		return
	}
	writeJSON(w, ratings)
}

func (h *AdminHandler) ListUsers(w http.ResponseWriter, r *http.Request) {
	var users []models.User
	if err := h.db.Order("created_at DESC").Find(&users).Error; err != nil {
		http.Error(w, "failed to fetch users", http.StatusInternalServerError)
		return
	}
	writeJSON(w, users)
}

func (h *AdminHandler) ListRideHistory(w http.ResponseWriter, r *http.Request) {
	var history []models.RideHistory
	if err := h.db.Order("created_at DESC").Find(&history).Error; err != nil {
		http.Error(w, "failed to fetch ride history", http.StatusInternalServerError)
		return
	}
	writeJSON(w, history)
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(false)
	enc.Encode(v)
}
