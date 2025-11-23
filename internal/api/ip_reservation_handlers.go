package api

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"
	"golang-backend-service/internal/database"
	"golang-backend-service/internal/ionos"
)

// IPReservationHandler handles IP reservation API requests
type IPReservationHandler struct {
	service *ionos.Service
	logger  *logrus.Logger
}

// NewIPReservationHandler creates a new IP reservation handler
func NewIPReservationHandler(service *ionos.Service, logger *logrus.Logger) *IPReservationHandler {
	return &IPReservationHandler{
		service: service,
		logger:  logger,
	}
}

// ReserveIPsRequest represents the request to reserve IPs
type ReserveIPsRequest struct {
	Count    int    `json:"count"`
	Location string `json:"location,omitempty"`
}

// HandleReserveIPs handles POST /api/v1/ips/reserve
func (h *IPReservationHandler) HandleReserveIPs(w http.ResponseWriter, r *http.Request) {
	var req ReserveIPsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.logger.WithError(err).Error("Failed to decode request")
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.Count <= 0 || req.Count > 50 {
		http.Error(w, "Count must be between 1 and 50", http.StatusBadRequest)
		return
	}

	h.logger.WithFields(logrus.Fields{
		"action":   "reserve_ips",
		"count":    req.Count,
		"location": req.Location,
	}).Info("Received IP reservation request")

	response, err := h.service.ReserveCleanIPs(r.Context(), req.Count, req.Location)
	if err != nil {
		h.logger.WithError(err).Error("Failed to reserve IPs")
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(response)
}

// HandleListReservedIPs handles GET /api/v1/ips/reserved
func (h *IPReservationHandler) HandleListReservedIPs(w http.ResponseWriter, r *http.Request) {
	// Parse query parameters
	var status *string
	var isBlacklisted *bool
	var location *string

	if s := r.URL.Query().Get("status"); s != "" {
		status = &s
	}

	if bl := r.URL.Query().Get("blacklisted"); bl != "" {
		b := bl == "true"
		isBlacklisted = &b
	}

	if loc := r.URL.Query().Get("location"); loc != "" {
		location = &loc
	}

	h.logger.WithFields(logrus.Fields{
		"action":        "list_reserved_ips",
		"status":        status,
		"is_blacklisted": isBlacklisted,
		"location":      location,
	}).Info("Listing reserved IPs")

	ips, err := database.ListReservedIPs(status, isBlacklisted, location)
	if err != nil {
		h.logger.WithError(err).Error("Failed to list reserved IPs")
		http.Error(w, "Failed to retrieve reserved IPs", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"count": len(ips),
		"ips":   ips,
	})
}

// HandleGetReservedIP handles GET /api/v1/ips/reserved/{id}
func (h *IPReservationHandler) HandleGetReservedIP(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	idStr := vars["id"]

	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "Invalid IP ID", http.StatusBadRequest)
		return
	}

	h.logger.WithField("ip_id", id).Info("Retrieving reserved IP")

	ip, err := database.GetReservedIPByID(id)
	if err != nil {
		h.logger.WithError(err).Error("Failed to get reserved IP")
		http.Error(w, "Reserved IP not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(ip)
}

// UpdateIPStatusRequest represents the request to update IP status
type UpdateIPStatusRequest struct {
	Status     string  `json:"status"`
	AssignedTo *string `json:"assigned_to,omitempty"`
}

// HandleUpdateIPStatus handles PUT /api/v1/ips/reserved/{id}/status
func (h *IPReservationHandler) HandleUpdateIPStatus(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	idStr := vars["id"]

	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "Invalid IP ID", http.StatusBadRequest)
		return
	}

	var req UpdateIPStatusRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.logger.WithError(err).Error("Failed to decode request")
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	validStatuses := map[string]bool{
		"reserved":    true,
		"in_use":      true,
		"released":    true,
		"quarantined": true,
	}

	if !validStatuses[req.Status] {
		http.Error(w, "Invalid status. Must be: reserved, in_use, released, or quarantined", http.StatusBadRequest)
		return
	}

	h.logger.WithFields(logrus.Fields{
		"ip_id":       id,
		"new_status":  req.Status,
		"assigned_to": req.AssignedTo,
	}).Info("Updating IP status")

	if err := database.UpdateReservedIPStatus(id, req.Status, req.AssignedTo); err != nil {
		h.logger.WithError(err).Error("Failed to update IP status")
		http.Error(w, "Failed to update IP status", http.StatusInternalServerError)
		return
	}

	// Retrieve updated IP
	ip, err := database.GetReservedIPByID(id)
	if err != nil {
		h.logger.WithError(err).Error("Failed to get updated IP")
		http.Error(w, "Failed to retrieve updated IP", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(ip)
}

// HandleRecheckBlacklist handles POST /api/v1/ips/reserved/{id}/recheck
func (h *IPReservationHandler) HandleRecheckBlacklist(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	idStr := vars["id"]

	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "Invalid IP ID", http.StatusBadRequest)
		return
	}

	h.logger.WithField("ip_id", id).Info("Rechecking blacklist status")

	if err := h.service.RecheckBlacklist(r.Context(), id); err != nil {
		h.logger.WithError(err).Error("Failed to recheck blacklist")
		http.Error(w, "Failed to recheck blacklist", http.StatusInternalServerError)
		return
	}

	// Retrieve updated IP
	ip, err := database.GetReservedIPByID(id)
	if err != nil {
		h.logger.WithError(err).Error("Failed to get updated IP")
		http.Error(w, "Failed to retrieve updated IP", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(ip)
}

// HandleDeleteReservedIP handles DELETE /api/v1/ips/reserved/{id}
func (h *IPReservationHandler) HandleDeleteReservedIP(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	idStr := vars["id"]

	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "Invalid IP ID", http.StatusBadRequest)
		return
	}

	h.logger.WithField("ip_id", id).Info("Deleting reserved IP")

	// Get IP info before deleting
	ip, err := database.GetReservedIPByID(id)
	if err != nil {
		h.logger.WithError(err).Error("Failed to get reserved IP")
		http.Error(w, "Reserved IP not found", http.StatusNotFound)
		return
	}

	// Delete from IONOS if block still exists
	if ip.ReservationBlockID != "" {
		h.logger.WithField("block_id", ip.ReservationBlockID).Info("Deleting IONOS block")
		if _, err := h.service.CleanupSingleIPBlocks(r.Context()); err != nil {
			h.logger.WithError(err).Warn("Failed to delete IONOS block, continuing with database deletion")
		}
	}

	// Delete from database
	if err := database.DeleteReservedIP(id); err != nil {
		h.logger.WithError(err).Error("Failed to delete reserved IP from database")
		http.Error(w, "Failed to delete reserved IP", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// HandleCheckQuota handles GET /api/v1/ips/quota
func (h *IPReservationHandler) HandleCheckQuota(w http.ResponseWriter, r *http.Request) {
	h.logger.Info("Checking IONOS quota")

	quota, err := h.service.CheckQuota(r.Context())
	if err != nil {
		h.logger.WithError(err).Error("Failed to check quota")
		http.Error(w, "Failed to check quota", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(quota)
}

// HandleCleanupBlocks handles POST /api/v1/ips/cleanup
func (h *IPReservationHandler) HandleCleanupBlocks(w http.ResponseWriter, r *http.Request) {
	h.logger.Info("Starting cleanup of single-IP blocks")

	deletedCount, err := h.service.CleanupSingleIPBlocks(r.Context())
	if err != nil {
		h.logger.WithError(err).Error("Failed to cleanup blocks")
		http.Error(w, "Failed to cleanup blocks", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"deleted_count": deletedCount,
		"message":       "Cleanup completed successfully",
	})
}

// HandleGetStatistics handles GET /api/v1/ips/statistics
func (h *IPReservationHandler) HandleGetStatistics(w http.ResponseWriter, r *http.Request) {
	h.logger.Info("Retrieving IP reservation statistics")

	stats, err := database.GetReservationStatistics()
	if err != nil {
		h.logger.WithError(err).Error("Failed to get statistics")
		http.Error(w, "Failed to retrieve statistics", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(stats)
}

