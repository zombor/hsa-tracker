package receipt

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"path/filepath"
	"strings"
)

// corsError writes an error response with CORS headers set
func corsError(w http.ResponseWriter, message string, code int) {
	setCORSHeaders(w)
	http.Error(w, message, code)
}

// setCORSHeaders sets CORS headers on a response
func setCORSHeaders(w http.ResponseWriter) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, DELETE, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
	w.Header().Set("Access-Control-Max-Age", "3600")
}

// handleIndex serves the HTML interface
func (s *Server) handleIndex(w http.ResponseWriter, r *http.Request) {
	setCORSHeaders(w)
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write(indexHTML)
}

// handleListReceipts returns a list of all receipts
func (s *Server) handleListReceipts(w http.ResponseWriter, r *http.Request) {
	receipts, err := s.service.ListReceipts()
	if err != nil {
		slog.Error("Error listing receipts", "error", err)
		corsError(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(receipts); err != nil {
		slog.Error("Error encoding response", "error", err)
	}
}

// handleUploadReceipt handles receipt upload
func (s *Server) handleUploadReceipt(w http.ResponseWriter, r *http.Request) {
	// Parse multipart form (max 50MB to handle high-resolution phone photos)
	// Increase from 10MB to 50MB for better mobile support
	maxFormSize := int64(50 << 20) // 50MB
	if err := r.ParseMultipartForm(maxFormSize); err != nil {
		slog.Error("Error parsing multipart form", "error", err)
		setCORSHeaders(w)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		errorMsg := "Error parsing form"
		if err.Error() == "http: request body too large" {
			errorMsg = "File is too large. Maximum size is 50MB. Please compress or resize your image."
		}
		json.NewEncoder(w).Encode(map[string]string{
			"error": errorMsg,
		})
		return
	}

	f, header, err := r.FormFile("file")
	if err != nil {
		slog.Error("Error getting file from form", "error", err)
		setCORSHeaders(w)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		errorMsg := "No file provided"
		if err.Error() == "http: no such file" {
			errorMsg = "No file was selected. Please choose a file to upload."
		}
		json.NewEncoder(w).Encode(map[string]string{
			"error": errorMsg,
		})
		return
	}
	defer f.Close()

	// Check file size before reading
	if header.Size > maxFormSize {
		setCORSHeaders(w)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{
			"error": "File is too large. Maximum size is 50MB. Please compress or resize your image.",
		})
		return
	}

	// Read file data
	data, err := io.ReadAll(f)
	if err != nil {
		slog.Error("Error reading file data", "error", err, "filename", header.Filename)
		setCORSHeaders(w)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{
			"error": "Error reading file. Please try again.",
		})
		return
	}

	// Determine content type
	contentType := header.Header.Get("Content-Type")
	if contentType == "" {
		ext := strings.ToLower(filepath.Ext(header.Filename))
		switch ext {
		case ".jpg", ".jpeg":
			contentType = "image/jpeg"
		case ".png":
			contentType = "image/png"
		case ".pdf":
			contentType = "application/pdf"
		case ".heic":
			contentType = "image/heic"
		case ".heif":
			contentType = "image/heif"
		default:
			contentType = "application/octet-stream"
		}
	}
	
	// Normalize content type for common phone formats
	contentType = strings.ToLower(strings.TrimSpace(contentType))
	// Preserve HEIC/HEIF MIME types so conversion logic can detect them
	// The conversion logic will handle converting HEIC to PNG

	// Process receipt
	receipt, err := s.service.ProcessReceipt(header.Filename, data, contentType)
	if err != nil {
		slog.Error("Error processing receipt", "filename", header.Filename, "error", err)
		setCORSHeaders(w)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{
			"error": err.Error(),
		})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	if err := json.NewEncoder(w).Encode(receipt); err != nil {
		slog.Error("Error encoding response", "error", err)
	}
}

// handleGetReceipt returns a single receipt
func (s *Server) handleGetReceipt(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		corsError(w, "Receipt ID required", http.StatusBadRequest)
		return
	}
	receipt, err := s.service.GetReceipt(id)
	if err != nil {
		corsError(w, "Receipt not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(receipt); err != nil {
		slog.Error("Error encoding response", "error", err)
	}
}

// handleGetReceiptFile returns the file for a receipt
func (s *Server) handleGetReceiptFile(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		corsError(w, "Receipt ID required", http.StatusBadRequest)
		return
	}
	data, contentType, err := s.service.GetReceiptFile(id)
	if err != nil {
		corsError(w, "File not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", contentType)
	w.Write(data)
}

// handleDeleteReceipt deletes a receipt
func (s *Server) handleDeleteReceipt(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		corsError(w, "Receipt ID required", http.StatusBadRequest)
		return
	}
	if err := s.service.DeleteReceipt(id); err != nil {
		corsError(w, "Error deleting receipt", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// handleListReimbursements returns a list of all reimbursements
func (s *Server) handleListReimbursements(w http.ResponseWriter, r *http.Request) {
	reimbursements, err := s.service.ListReimbursements()
	if err != nil {
		slog.Error("Error listing reimbursements", "error", err)
		corsError(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Ensure we always return an array, not nil
	if reimbursements == nil {
		reimbursements = []*Reimbursement{}
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(reimbursements); err != nil {
		slog.Error("Error encoding response", "error", err)
	}
}

// handleCreateReimbursement handles reimbursement creation
func (s *Server) handleCreateReimbursement(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ReceiptIDs []string `json:"receipt_ids"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		corsError(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	reimbursement, err := s.service.CreateReimbursement(req.ReceiptIDs)
	if err != nil {
		slog.Error("Error creating reimbursement", "error", err)
		setCORSHeaders(w)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{
			"error": err.Error(),
		})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	if err := json.NewEncoder(w).Encode(reimbursement); err != nil {
		slog.Error("Error encoding response", "error", err)
	}
}

// handleGetReimbursement returns a reimbursement with its receipts
func (s *Server) handleGetReimbursement(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		corsError(w, "Reimbursement ID required", http.StatusBadRequest)
		return
	}
	reimbursement, receipts, err := s.service.GetReimbursementWithReceipts(id)
	if err != nil {
		corsError(w, "Reimbursement not found", http.StatusNotFound)
		return
	}

	response := map[string]interface{}{
		"reimbursement": reimbursement,
		"receipts":      receipts,
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		slog.Error("Error encoding response", "error", err)
	}
}

// handleStaticCSS serves the CSS file
func (s *Server) handleStaticCSS(w http.ResponseWriter, r *http.Request) {
	setCORSHeaders(w)
	w.Header().Set("Content-Type", "text/css")
	w.Write(appCSS)
}

// handleStaticJS serves the JavaScript file
func (s *Server) handleStaticJS(w http.ResponseWriter, r *http.Request) {
	setCORSHeaders(w)
	// Use module MIME type for ES6 modules
	w.Header().Set("Content-Type", "application/javascript; charset=utf-8")
	w.Write(appJS)
}
