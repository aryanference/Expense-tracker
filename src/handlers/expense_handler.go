package handlers

import (
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"strings"

	"github.com/yourname/expense-tracker/src/models"
	"github.com/yourname/expense-tracker/src/service"
)

type ExpenseHandler struct{ service *service.ExpenseService }

func NewExpenseHandler(s *service.ExpenseService) *ExpenseHandler { return &ExpenseHandler{service: s} }

func (h *ExpenseHandler) Create(w http.ResponseWriter, r *http.Request) {
	var request models.CreateExpenseRequest
	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(&request); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	expense, err := h.service.AddExpense(request)
	if err != nil {
		if errors.Is(err, service.ErrPersistFailed) {
			log.Printf("saving expense: %v", err)
			writeError(w, http.StatusInternalServerError, "internal server error")
			return
		}
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	w.Header().Set("Location", "/expenses/"+expense.ID)
	writeJSON(w, http.StatusCreated, expense)
}

func (h *ExpenseHandler) List(w http.ResponseWriter, r *http.Request) {
	// Category comparisons are case-insensitive so callers need not match stored casing.
	writeJSON(w, http.StatusOK, h.service.ListExpenses(strings.TrimSpace(r.URL.Query().Get("category"))))
}

func (h *ExpenseHandler) Total(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]float64{"total": h.service.GetTotal(strings.TrimSpace(r.URL.Query().Get("category")))})
}

func (h *ExpenseHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		writeError(w, http.StatusBadRequest, "expense id is required")
		return
	}
	if err := h.service.DeleteExpense(id); err != nil {
		if errors.Is(err, service.ErrExpenseNotFound) {
			writeError(w, http.StatusNotFound, "expense not found")
			return
		}
		log.Printf("deleting expense: %v", err)
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusNoContent)
}

func MethodNotAllowed(w http.ResponseWriter, r *http.Request) {
	writeError(w, http.StatusMethodNotAllowed, "method not allowed")
}
func NotFound(w http.ResponseWriter, r *http.Request) {
	writeError(w, http.StatusNotFound, "not found")
}
func MissingID(w http.ResponseWriter, r *http.Request) {
	writeError(w, http.StatusBadRequest, "expense id is required")
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}
func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}
