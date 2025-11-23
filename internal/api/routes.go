package api

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"golang-backend-service/internal/database"
	"golang-backend-service/internal/ionos"
	"golang-backend-service/internal/logger"

	"github.com/gorilla/mux"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/sirupsen/logrus"
	httpSwagger "github.com/swaggo/http-swagger"
)

// Prometheus metrics
var (
	httpRequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_requests_total",
			Help: "Total number of HTTP requests",
		},
		[]string{"method", "endpoint", "status"},
	)

	httpRequestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_request_duration_seconds",
			Help:    "Duration of HTTP requests in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"method", "endpoint"},
	)
)

// CreateUserRequest represents the request body for creating a user
type CreateUserRequest struct {
	Username string `json:"username"`
	Email    string `json:"email"`
}

// ErrorResponse represents an error response
type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message"`
}

// corsMiddleware adds CORS headers to all responses
func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Allow requests from any origin (for development)
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		
		// Handle preflight requests
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}
		
		next.ServeHTTP(w, r)
	})
}

// HealthResponse represents the health check response
type HealthResponse struct {
	Status    string    `json:"status"`
	Timestamp time.Time `json:"timestamp"`
}

// SetupRoutes configures all API routes
func SetupRoutes() *mux.Router {
	return SetupRoutesWithDependencies(nil)
}

// SetupRoutesWithDependencies configures all API routes with optional dependencies
func SetupRoutesWithDependencies(ionosService *ionos.Service) *mux.Router {
	router := mux.NewRouter()

	// Add CORS middleware (must be first to handle preflight requests)
	router.Use(corsMiddleware)
	
	// Add metrics middleware
	router.Use(metricsMiddleware)
	router.Use(loggingMiddleware)

	// Health check endpoint
	router.HandleFunc("/health", healthHandler).Methods("GET")

	// User endpoints
	router.HandleFunc("/users", getUsersHandler).Methods("GET")
	router.HandleFunc("/users", createUserHandler).Methods("POST")
	router.HandleFunc("/users/{id}", getUserByIDHandler).Methods("GET")

	// IP Reputation endpoints
	router.HandleFunc("/api/webhooks/stalwart/delivery-failure", processDeliveryFailureHandler).Methods("POST")
	router.HandleFunc("/api/ips/{ip}/reputation", getIPReputationHandler).Methods("GET")
	router.HandleFunc("/api/ips/{ip}/failures", getIPFailuresHandler).Methods("GET")
	router.HandleFunc("/api/ips/{ip}/quarantine", quarantineIPHandler).Methods("POST")
	router.HandleFunc("/api/ips/{ip}/dnsbl-check", checkDNSBLHandler).Methods("POST")
	router.HandleFunc("/api/dashboard/ip-health", getIPHealthDashboardHandler).Methods("GET")
	
	// IP Reservation endpoints (IONOS)
	if ionosService != nil {
		ipHandler := NewIPReservationHandler(ionosService, logger.Log)
		router.HandleFunc("/api/v1/ips/reserve", ipHandler.HandleReserveIPs).Methods("POST")
		router.HandleFunc("/api/v1/ips/reserved", ipHandler.HandleListReservedIPs).Methods("GET")
		router.HandleFunc("/api/v1/ips/reserved/{id}", ipHandler.HandleGetReservedIP).Methods("GET")
		router.HandleFunc("/api/v1/ips/reserved/{id}/status", ipHandler.HandleUpdateIPStatus).Methods("PUT")
		router.HandleFunc("/api/v1/ips/reserved/{id}/recheck", ipHandler.HandleRecheckBlacklist).Methods("POST")
		router.HandleFunc("/api/v1/ips/reserved/{id}", ipHandler.HandleDeleteReservedIP).Methods("DELETE")
		router.HandleFunc("/api/v1/ips/quota", ipHandler.HandleCheckQuota).Methods("GET")
		router.HandleFunc("/api/v1/ips/cleanup", ipHandler.HandleCleanupBlocks).Methods("POST")
		router.HandleFunc("/api/v1/ips/statistics", ipHandler.HandleGetStatistics).Methods("GET")
	}
	
	// Testing endpoints
	router.HandleFunc("/api/testing/simulate-failures", simulateFailuresHandler).Methods("POST")
	router.HandleFunc("/api/testing/test-cases", getTestCasesHandler).Methods("GET")
	router.HandleFunc("/api/testing/test-cases/{id}/run", runTestCaseHandler).Methods("POST")
	router.HandleFunc("/api/testing/test-suite/run", runTestSuiteHandler).Methods("POST")

	// Metrics endpoint for Prometheus
	router.Handle("/metrics", promhttp.Handler())

	// Swagger documentation
	router.PathPrefix("/swagger/").Handler(httpSwagger.WrapHandler)

	return router
}

// metricsMiddleware tracks HTTP request metrics
func metricsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Create a response writer wrapper to capture status code
		wrapped := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}

		next.ServeHTTP(wrapped, r)

		duration := time.Since(start).Seconds()
		
		// Record metrics
		httpRequestsTotal.WithLabelValues(
			r.Method,
			r.URL.Path,
			strconv.Itoa(wrapped.statusCode),
		).Inc()

		httpRequestDuration.WithLabelValues(
			r.Method,
			r.URL.Path,
		).Observe(duration)
	})
}

// loggingMiddleware logs all HTTP requests
func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		wrapped := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}
		next.ServeHTTP(wrapped, r)

		logger.WithFields(logrus.Fields{
			"method":      r.Method,
			"path":        r.URL.Path,
			"status":      wrapped.statusCode,
			"duration_ms": time.Since(start).Milliseconds(),
			"remote_addr": r.RemoteAddr,
		}).Info("HTTP request processed")
	})
}

// responseWriter wraps http.ResponseWriter to capture status code
type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

// @Summary Health check
// @Description Check if the service is healthy
// @Tags health
// @Produce json
// @Success 200 {object} HealthResponse
// @Router /health [get]
func healthHandler(w http.ResponseWriter, r *http.Request) {
	response := HealthResponse{
		Status:    "healthy",
		Timestamp: time.Now(),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// @Summary Get all users
// @Description Retrieve all users from the database
// @Tags users
// @Produce json
// @Success 200 {array} database.User
// @Failure 500 {object} ErrorResponse
// @Router /users [get]
func getUsersHandler(w http.ResponseWriter, r *http.Request) {
	users, err := database.GetAllUsers()
	if err != nil {
		logger.WithFields(logrus.Fields{
			"action": "get_users",
			"error":  err.Error(),
		}).Error("Failed to retrieve users")

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(ErrorResponse{
			Error:   "database_error",
			Message: "Failed to retrieve users",
		})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(users)
}

// @Summary Create a new user
// @Description Create a new user with username and email
// @Tags users
// @Accept json
// @Produce json
// @Param user body CreateUserRequest true "User information"
// @Success 201 {object} database.User
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /users [post]
func createUserHandler(w http.ResponseWriter, r *http.Request) {
	var req CreateUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		logger.WithFields(logrus.Fields{
			"action": "create_user",
			"error":  err.Error(),
		}).Warn("Invalid request body")

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(ErrorResponse{
			Error:   "invalid_request",
			Message: "Invalid request body",
		})
		return
	}

	if req.Username == "" || req.Email == "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(ErrorResponse{
			Error:   "validation_error",
			Message: "Username and email are required",
		})
		return
	}

	user, err := database.CreateUser(req.Username, req.Email)
	if err != nil {
		logger.WithFields(logrus.Fields{
			"action":   "create_user",
			"username": req.Username,
			"error":    err.Error(),
		}).Error("Failed to create user")

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(ErrorResponse{
			Error:   "database_error",
			Message: "Failed to create user",
		})
		return
	}

	logger.WithFields(logrus.Fields{
		"action":   "create_user",
		"user_id":  user.ID,
		"username": user.Username,
	}).Info("User created successfully")

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(user)
}

// @Summary Get user by ID
// @Description Retrieve a specific user by their ID
// @Tags users
// @Produce json
// @Param id path int true "User ID"
// @Success 200 {object} database.User
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /users/{id} [get]
func getUserByIDHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	idStr := vars["id"]

	id, err := strconv.Atoi(idStr)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(ErrorResponse{
			Error:   "invalid_id",
			Message: "User ID must be a number",
		})
		return
	}

	user, err := database.GetUserByID(id)
	if err != nil {
		if err.Error() == "user not found" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusNotFound)
			json.NewEncoder(w).Encode(ErrorResponse{
				Error:   "not_found",
				Message: "User not found",
			})
			return
		}

		logger.WithFields(logrus.Fields{
			"action":  "get_user_by_id",
			"user_id": id,
			"error":   err.Error(),
		}).Error("Failed to retrieve user")

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(ErrorResponse{
			Error:   "database_error",
			Message: "Failed to retrieve user",
		})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(user)
}

