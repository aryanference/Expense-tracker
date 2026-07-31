package tests

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/yourname/expense-tracker/src/handlers"
	"github.com/yourname/expense-tracker/src/models"
)

func testRouter() http.Handler {
	h := handlers.NewExpenseHandler(newService())
	mux := http.NewServeMux()
	mux.HandleFunc("POST /expenses", h.Create)
	mux.HandleFunc("GET /expenses", h.List)
	mux.HandleFunc("GET /expenses/total", h.Total)
	mux.HandleFunc("DELETE /expenses/{id}", h.Delete)
	mux.HandleFunc("DELETE /expenses/", handlers.MissingID)
	mux.HandleFunc("/expenses", handlers.MethodNotAllowed)
	mux.HandleFunc("/expenses/", handlers.MethodNotAllowed)
	mux.HandleFunc("/", handlers.NotFound)
	return mux
}
func request(r http.Handler, method, path, body string) *httptest.ResponseRecorder {
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(method, path, bytes.NewBufferString(body)))
	return w
}
func add(t *testing.T, r http.Handler, title, category string, amount float64) models.Expense {
	t.Helper()
	b, _ := json.Marshal(models.CreateExpenseRequest{Title: title, Amount: amount, Category: category, Date: "2026-07-31"})
	w := request(r, "POST", "/expenses", string(b))
	if w.Code != 201 {
		t.Fatal(w.Code, w.Body.String())
	}
	var e models.Expense
	_ = json.NewDecoder(w.Body).Decode(&e)
	return e
}
func TestPostExpense_Valid_Returns201(t *testing.T) {
	w := request(testRouter(), "POST", "/expenses", `{"title":"Coffee","amount":4.5,"category":"Food","date":"2026-07-31"}`)
	var e models.Expense
	_ = json.NewDecoder(w.Body).Decode(&e)
	if w.Code != 201 || e.ID == "" || w.Header().Get("Location") != "/expenses/"+e.ID {
		t.Fatal(w.Code, e, w.Header())
	}
}
func TestPostExpense_MissingTitle_Returns400(t *testing.T) {
	assertError(t, request(testRouter(), "POST", "/expenses", `{"amount":1,"category":"x","date":"2026-01-01"}`), 400, "title is required")
}
func TestPostExpense_NonPositiveAmount_Returns400(t *testing.T) {
	assertError(t, request(testRouter(), "POST", "/expenses", `{"title":"x","amount":0,"category":"x","date":"2026-01-01"}`), 400, "amount must be greater than 0")
}
func TestPostExpense_MissingCategory_Returns400(t *testing.T) {
	assertError(t, request(testRouter(), "POST", "/expenses", `{"title":"x","amount":1,"date":"2026-01-01"}`), 400, "category is required")
}
func TestPostExpense_InvalidDateFormat_Returns400(t *testing.T) {
	assertError(t, request(testRouter(), "POST", "/expenses", `{"title":"x","amount":1,"category":"x","date":"01-01-2026"}`), 400, "date must be in YYYY-MM-DD format")
}
func TestPostExpense_MalformedJSON_Returns400(t *testing.T) {
	assertError(t, request(testRouter(), "POST", "/expenses", "{"), 400, "invalid request body")
}
func TestGetExpenses_EmptyStore_ReturnsEmptyArray(t *testing.T) {
	w := request(testRouter(), "GET", "/expenses", "")
	var got []models.Expense
	_ = json.NewDecoder(w.Body).Decode(&got)
	if w.Code != 200 || got == nil || len(got) != 0 {
		t.Fatal(w.Code, got)
	}
}
func TestGetExpenses_ReturnsInsertionOrder(t *testing.T) {
	r := testRouter()
	add(t, r, "first", "x", 1)
	add(t, r, "second", "x", 1)
	w := request(r, "GET", "/expenses", "")
	var got []models.Expense
	_ = json.NewDecoder(w.Body).Decode(&got)
	if len(got) != 2 || got[0].Title != "first" || got[1].Title != "second" {
		t.Fatal(got)
	}
}
func TestGetExpenses_FilterByCategory_CaseInsensitive(t *testing.T) {
	r := testRouter()
	add(t, r, "a", "Food", 1)
	w := request(r, "GET", "/expenses?category=%20food%20", "")
	var got []models.Expense
	_ = json.NewDecoder(w.Body).Decode(&got)
	if len(got) != 1 {
		t.Fatal(got)
	}
}
func TestGetExpenses_FilterByCategory_NoMatches_ReturnsEmptyArray(t *testing.T) {
	w := request(testRouter(), "GET", "/expenses?category=no", "")
	var got []models.Expense
	_ = json.NewDecoder(w.Body).Decode(&got)
	if got == nil || len(got) != 0 {
		t.Fatal(got)
	}
}
func TestGetTotal_EmptyStore_ReturnsZero(t *testing.T) {
	assertTotal(t, request(testRouter(), "GET", "/expenses/total", ""), 0)
}
func TestGetTotal_Overall_CorrectSum(t *testing.T) {
	r := testRouter()
	add(t, r, "a", "x", 3.1)
	add(t, r, "b", "x", 2.2)
	assertTotal(t, request(r, "GET", "/expenses/total", ""), 5.3)
}
func TestGetTotal_ByCategory_ExcludesOtherCategories(t *testing.T) {
	r := testRouter()
	add(t, r, "a", "food", 3)
	add(t, r, "b", "other", 9)
	assertTotal(t, request(r, "GET", "/expenses/total?category=FOOD", ""), 3)
}
func TestDeleteExpense_Existing_Returns204(t *testing.T) {
	r := testRouter()
	e := add(t, r, "a", "x", 1)
	w := request(r, "DELETE", "/expenses/"+e.ID, "")
	if w.Code != 204 || w.Body.Len() != 0 {
		t.Fatal(w.Code, w.Body.String())
	}
}
func TestDeleteExpense_Nonexistent_Returns404(t *testing.T) {
	assertError(t, request(testRouter(), "DELETE", "/expenses/no", ""), 404, "expense not found")
}
func TestDeleteExpense_ThenGet_ConfirmsRemoval(t *testing.T) {
	r := testRouter()
	e := add(t, r, "a", "x", 1)
	_ = request(r, "DELETE", "/expenses/"+e.ID, "")
	w := request(r, "GET", "/expenses", "")
	var got []models.Expense
	_ = json.NewDecoder(w.Body).Decode(&got)
	if len(got) != 0 {
		t.Fatal(got)
	}
}
func TestMethodNotAllowed_Returns405(t *testing.T) {
	assertError(t, request(testRouter(), "PUT", "/expenses", ""), 405, "method not allowed")
}
func TestUnknownRoute_Returns404WithJSONBody(t *testing.T) {
	assertError(t, request(testRouter(), "GET", "/nonexistent", ""), 404, "not found")
}
func assertError(t *testing.T, w *httptest.ResponseRecorder, status int, message string) {
	t.Helper()
	var got map[string]string
	_ = json.NewDecoder(w.Body).Decode(&got)
	if w.Code != status || got["error"] != message || w.Header().Get("Content-Type") != "application/json; charset=utf-8" {
		t.Fatalf("code=%d body=%v header=%q", w.Code, got, w.Header().Get("Content-Type"))
	}
}
func assertTotal(t *testing.T, w *httptest.ResponseRecorder, want float64) {
	t.Helper()
	var got struct {
		Total float64 `json:"total"`
	}
	_ = json.NewDecoder(w.Body).Decode(&got)
	if w.Code != 200 || got.Total != want {
		t.Fatal(w.Code, got.Total)
	}
}
