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
  <title>Rickshaw Admin Dashboard</title>
  <style>
    * { margin: 0; padding: 0; box-sizing: border-box; }
    
    body {
      font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, "Helvetica Neue", Arial, sans-serif;
      background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
      min-height: 100vh;
      padding: 20px;
      color: #333;
    }
    
    .container {
      max-width: 1400px;
      margin: 0 auto;
    }
    
    header {
      background: white;
      border-radius: 12px;
      padding: 24px 32px;
      margin-bottom: 24px;
      box-shadow: 0 4px 6px rgba(0, 0, 0, 0.1);
    }
    
    h1 {
      font-size: 28px;
      color: #667eea;
      font-weight: 600;
      margin-bottom: 8px;
    }
    
    .subtitle {
      color: #6b7280;
      font-size: 14px;
    }
    
    .tabs {
      display: flex;
      gap: 8px;
      margin-bottom: 24px;
      flex-wrap: wrap;
    }
    
    .tab {
      background: white;
      border: none;
      padding: 12px 24px;
      border-radius: 8px;
      cursor: pointer;
      font-size: 14px;
      font-weight: 500;
      color: #6b7280;
      transition: all 0.2s;
      box-shadow: 0 2px 4px rgba(0, 0, 0, 0.1);
    }
    
    .tab:hover {
      transform: translateY(-2px);
      box-shadow: 0 4px 8px rgba(0, 0, 0, 0.15);
    }
    
    .tab.active {
      background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
      color: white;
    }
    
    .content-section {
      display: none;
      background: white;
      border-radius: 12px;
      padding: 24px;
      box-shadow: 0 4px 6px rgba(0, 0, 0, 0.1);
      animation: fadeIn 0.3s;
    }
    
    .content-section.active {
      display: block;
    }
    
    @keyframes fadeIn {
      from { opacity: 0; transform: translateY(10px); }
      to { opacity: 1; transform: translateY(0); }
    }
    
    h2 {
      font-size: 20px;
      margin-bottom: 16px;
      color: #1f2937;
      border-bottom: 2px solid #667eea;
      padding-bottom: 8px;
    }
    
    .loading {
      text-align: center;
      padding: 40px;
      color: #6b7280;
      font-size: 14px;
    }
    
    .error {
      background: #fee;
      border: 1px solid #fcc;
      border-radius: 8px;
      padding: 16px;
      color: #c00;
      font-size: 14px;
    }
    
    .table-container {
      overflow-x: auto;
      border-radius: 8px;
      border: 1px solid #e5e7eb;
    }
    
    table {
      width: 100%;
      border-collapse: collapse;
      font-size: 14px;
    }
    
    thead {
      background: #f9fafb;
      position: sticky;
      top: 0;
    }
    
    th {
      padding: 12px 16px;
      text-align: left;
      font-weight: 600;
      color: #374151;
      border-bottom: 2px solid #e5e7eb;
      white-space: nowrap;
    }
    
    td {
      padding: 12px 16px;
      border-bottom: 1px solid #f3f4f6;
    }
    
    tr:hover {
      background: #f9fafb;
    }
    
    tr:last-child td {
      border-bottom: none;
    }
    
    .badge {
      display: inline-block;
      padding: 4px 12px;
      border-radius: 12px;
      font-size: 12px;
      font-weight: 500;
      text-transform: capitalize;
    }
    
    .status-requested { background: #fef3c7; color: #92400e; }
    .status-accepted { background: #dbeafe; color: #1e40af; }
    .status-started { background: #d1fae5; color: #065f46; }
    .status-completed { background: #dcfce7; color: #166534; }
    .status-cancelled { background: #fee2e2; color: #991b1b; }
    
    .bool-true { color: #059669; font-weight: 500; }
    .bool-false { color: #dc2626; font-weight: 500; }
    
    .rating-stars {
      color: #fbbf24;
      font-weight: 600;
    }
    
    .empty-state {
      text-align: center;
      padding: 60px 20px;
      color: #9ca3af;
    }
    
    .empty-state svg {
      width: 64px;
      height: 64px;
      margin-bottom: 16px;
      opacity: 0.5;
    }
    
    .stats-grid {
      display: grid;
      grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
      gap: 16px;
      margin-bottom: 24px;
    }
    
    .stat-card {
      background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
      color: white;
      padding: 20px;
      border-radius: 8px;
      box-shadow: 0 2px 4px rgba(0, 0, 0, 0.1);
    }
    
    .stat-label {
      font-size: 13px;
      opacity: 0.9;
      margin-bottom: 4px;
    }
    
    .stat-value {
      font-size: 28px;
      font-weight: 600;
    }
    
    @media (max-width: 768px) {
      body { padding: 12px; }
      header { padding: 16px 20px; }
      h1 { font-size: 22px; }
      .tab { padding: 10px 16px; font-size: 13px; }
      .content-section { padding: 16px; }
      th, td { padding: 8px 12px; font-size: 13px; }
      .stats-grid { grid-template-columns: 1fr; }
    }
  </style>
</head>
<body>
  <div class="container">
    <header>
      <h1>🚖 Rickshaw Admin Dashboard</h1>
      <div class="subtitle">Real-time monitoring and management</div>
    </header>
    
    <div class="stats-grid" id="stats"></div>
    
    <div class="tabs">
      <button class="tab active" data-tab="rides">Rides</button>
      <button class="tab" data-tab="drivers">Drivers</button>
      <button class="tab" data-tab="users">Users</button>
      <button class="tab" data-tab="ratings">Ratings</button>
      <button class="tab" data-tab="history">Ride History</button>
    </div>
    
    <div class="content-section active" id="rides-section">
      <h2>Recent Rides</h2>
      <div id="rides-content" class="loading">Loading...</div>
    </div>
    
    <div class="content-section" id="drivers-section">
      <h2>Active Drivers</h2>
      <div id="drivers-content" class="loading">Loading...</div>
    </div>
    
    <div class="content-section" id="users-section">
      <h2>Registered Users</h2>
      <div id="users-content" class="loading">Loading...</div>
    </div>
    
    <div class="content-section" id="ratings-section">
      <h2>Recent Ratings</h2>
      <div id="ratings-content" class="loading">Loading...</div>
    </div>
    
    <div class="content-section" id="history-section">
      <h2>Ride Status History</h2>
      <div id="history-content" class="loading">Loading...</div>
    </div>
  </div>
  
  <script>
    const state = {
      rides: [],
      drivers: [],
      users: [],
      ratings: [],
      history: []
    };
    
    // Tab switching
    document.querySelectorAll('.tab').forEach(tab => {
      tab.addEventListener('click', () => {
        document.querySelectorAll('.tab').forEach(t => t.classList.remove('active'));
        document.querySelectorAll('.content-section').forEach(s => s.classList.remove('active'));
        
        tab.classList.add('active');
        document.getElementById(tab.dataset.tab + '-section').classList.add('active');
      });
    });
    
    // Fetch data
    async function fetchData() {
      try {
        const [rides, drivers, users, ratings, history] = await Promise.all([
          fetch('/admin/api/rides').then(r => r.json()),
          fetch('/admin/api/drivers').then(r => r.json()),
          fetch('/admin/api/users').then(r => r.json()),
          fetch('/admin/api/ratings').then(r => r.json()),
          fetch('/admin/api/ride-history').then(r => r.json())
        ]);
        
        state.rides = rides;
        state.drivers = drivers;
        state.users = users;
        state.ratings = ratings;
        state.history = history;
        
        renderStats();
        renderRides();
        renderDrivers();
        renderUsers();
        renderRatings();
        renderHistory();
      } catch (err) {
        console.error('Failed to fetch data:', err);
        document.querySelectorAll('[id$="-content"]').forEach(el => {
          el.innerHTML = '<div class="error">Failed to load data. Please refresh the page.</div>';
        });
      }
    }
    
    function renderStats() {
      const totalRides = state.rides.length;
      const activeDrivers = state.drivers.filter(d => d.is_available).length;
      const completedRides = state.rides.filter(r => r.status === 'completed').length;
      const avgRating = state.ratings.length ? 
        (state.ratings.reduce((sum, r) => sum + r.rating, 0) / state.ratings.length).toFixed(1) : 'N/A';
      
      document.getElementById('stats').innerHTML = ` + "`" + `
        <div class="stat-card">
          <div class="stat-label">Total Rides</div>
          <div class="stat-value">${totalRides}</div>
        </div>
        <div class="stat-card">
          <div class="stat-label">Active Drivers</div>
          <div class="stat-value">${activeDrivers}</div>
        </div>
        <div class="stat-card">
          <div class="stat-label">Completed Rides</div>
          <div class="stat-value">${completedRides}</div>
        </div>
        <div class="stat-card">
          <div class="stat-label">Avg Rating</div>
          <div class="stat-value">${avgRating} ★</div>
        </div>
      ` + "`" + `;
    }
    
    function renderTable(containerId, columns, data, formatters = {}) {
      const container = document.getElementById(containerId);
      
      if (!data.length) {
        container.innerHTML = ` + "`" + `
          <div class="empty-state">
            <svg fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M20 13V6a2 2 0 00-2-2H6a2 2 0 00-2 2v7m16 0v5a2 2 0 01-2 2H6a2 2 0 01-2-2v-5m16 0h-2.586a1 1 0 00-.707.293l-2.414 2.414a1 1 0 01-.707.293h-3.172a1 1 0 01-.707-.293l-2.414-2.414A1 1 0 006.586 13H4" />
            </svg>
            <div>No data available</div>
          </div>
        ` + "`" + `;
        return;
      }
      
      const thead = columns.map(c => ` + "`<th>${c.label}</th>`" + `).join('');
      const tbody = data.map(row => {
        const cells = columns.map(c => {
          let value = row[c.key];
          if (formatters[c.key]) value = formatters[c.key](value, row);
          return ` + "`<td>${value ?? ''}</td>`" + `;
        }).join('');
        return ` + "`<tr>${cells}</tr>`" + `;
      }).join('');
      
      container.innerHTML = ` + "`" + `
        <div class="table-container">
          <table>
            <thead><tr>${thead}</tr></thead>
            <tbody>${tbody}</tbody>
          </table>
        </div>
      ` + "`" + `;
    }
    
    function formatStatus(status) {
      return ` + "`<span class=\"badge status-${status}\">${status}</span>`" + `;
    }
    
    function formatBool(val) {
      return ` + "`<span class=\"bool-${val}\">${val ? '✓ Yes' : '✗ No'}</span>`" + `;
    }
    
    function formatRating(val) {
      return ` + "`<span class=\"rating-stars\">${val} ★</span>`" + `;
    }
    
    function formatDate(val) {
      if (!val) return '';
      return new Date(val).toLocaleString('en-US', { 
        month: 'short', day: 'numeric', hour: '2-digit', minute: '2-digit' 
      });
    }
    
    function truncate(str, len = 30) {
      if (!str) return '';
      return str.length > len ? str.substring(0, len) + '...' : str;
    }
    
    function renderRides() {
      renderTable('rides-content', [
        { key: 'id', label: 'ID' },
        { key: 'rider_id', label: 'Rider' },
        { key: 'driver_id', label: 'Driver' },
        { key: 'status', label: 'Status' },
        { key: 'fare', label: 'Fare' },
        { key: 'distance', label: 'Distance' },
        { key: 'created_at', label: 'Created' }
      ], state.rides, {
        id: v => truncate(v, 8),
        rider_id: v => truncate(v, 8),
        driver_id: v => v ? truncate(v, 8) : '-',
        status: formatStatus,
        fare: v => v ? ` + "`${v.toFixed(2)}`" + ` : '-',
        distance: v => v ? ` + "`${v.toFixed(1)} km`" + ` : '-',
        created_at: formatDate
      });
    }
    
    function renderDrivers() {
      renderTable('drivers-content', [
        { key: 'id', label: 'ID' },
        { key: 'user_id', label: 'User' },
        { key: 'vehicle_number', label: 'Vehicle' },
        { key: 'is_available', label: 'Available' },
        { key: 'rating', label: 'Rating' },
        { key: 'total_rides', label: 'Total Rides' }
      ], state.drivers, {
        id: v => truncate(v, 8),
        user_id: v => truncate(v, 8),
        is_available: formatBool,
        rating: formatRating
      });
    }
    
    function renderUsers() {
      renderTable('users-content', [
        { key: 'id', label: 'ID' },
        { key: 'name', label: 'Name' },
        { key: 'phone', label: 'Phone' },
        { key: 'user_type', label: 'Type' },
        { key: 'created_at', label: 'Joined' }
      ], state.users, {
        id: v => truncate(v, 8),
        user_type: v => ` + "`<span class=\"badge\">${v}</span>`" + `,
        created_at: formatDate
      });
    }
    
    function renderRatings() {
      renderTable('ratings-content', [
        { key: 'id', label: 'ID' },
        { key: 'ride_id', label: 'Ride' },
        { key: 'driver_id', label: 'Driver' },
        { key: 'rating', label: 'Rating' },
        { key: 'comment', label: 'Comment' },
        { key: 'created_at', label: 'Date' }
      ], state.ratings, {
        id: v => truncate(v, 8),
        ride_id: v => truncate(v, 8),
        driver_id: v => truncate(v, 8),
        rating: formatRating,
        comment: v => truncate(v, 40),
        created_at: formatDate
      });
    }
    
    function renderHistory() {
      renderTable('history-content', [
        { key: 'id', label: 'ID' },
        { key: 'ride_id', label: 'Ride' },
        { key: 'status', label: 'Status' },
        { key: 'note', label: 'Note' },
        { key: 'created_at', label: 'Timestamp' }
      ], state.history, {
        id: v => truncate(v, 8),
        ride_id: v => truncate(v, 8),
        status: formatStatus,
        note: v => truncate(v, 50),
        created_at: formatDate
      });
    }
    
    // Initialize
    fetchData();
    
    // Auto-refresh every 30 seconds
    setInterval(fetchData, 30000);
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
