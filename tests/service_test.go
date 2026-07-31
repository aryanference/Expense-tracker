package tests

import (
	"strings"
	"testing"

	"github.com/yourname/expense-tracker/src/models"
	"github.com/yourname/expense-tracker/src/service"
	"github.com/yourname/expense-tracker/src/store"
)

func newService() *service.ExpenseService {
	return service.NewExpenseService(store.NewExpenseStore(nil))
}

func TestAddExpense_TrimsWhitespaceFromTitleAndCategory(t *testing.T) {
	e, err := newService().AddExpense(models.CreateExpenseRequest{Title: " Coffee ", Amount: 2, Category: " Food ", Date: "2026-07-31"})
	if err != nil || e.Title != "Coffee" || e.Category != "Food" {
		t.Fatalf("expense=%+v err=%v", e, err)
	}
}
func TestAddExpense_IgnoresClientSuppliedID(t *testing.T) {
	e, err := newService().AddExpense(models.CreateExpenseRequest{Title: "Coffee", Amount: 2, Category: "Food", Date: "2026-07-31"})
	if err != nil || e.ID == "" {
		t.Fatalf("server did not generate ID: %+v, %v", e, err)
	}
}
func TestGetTotal_RoundsToTwoDecimalPlaces(t *testing.T) {
	s := newService()
	for _, a := range []float64{0.1, 0.2} {
		_, _ = s.AddExpense(models.CreateExpenseRequest{Title: "x", Amount: a, Category: "f", Date: "2026-07-31"})
	}
	if got := s.GetTotal(""); got != 0.3 {
		t.Fatalf("got %v", got)
	}
}
func TestValidationRules(t *testing.T) {
	cases := []struct {
		req  models.CreateExpenseRequest
		want string
	}{
		{models.CreateExpenseRequest{Amount: 1, Category: "a", Date: "2026-01-01"}, "title is required"},
		{models.CreateExpenseRequest{Title: "a", Amount: 0, Category: "a", Date: "2026-01-01"}, "amount must be greater than 0"},
		{models.CreateExpenseRequest{Title: "a", Amount: 1, Date: "2026-01-01"}, "category is required"},
		{models.CreateExpenseRequest{Title: "a", Amount: 1, Category: "a", Date: "31-01-2026"}, "date must be in YYYY-MM-DD format"},
		{models.CreateExpenseRequest{Title: strings.Repeat("x", 201), Amount: 1, Category: "a", Date: "2026-01-01"}, "title must be 200 characters or fewer"},
	}
	for _, tc := range cases {
		_, err := newService().AddExpense(tc.req)
		if err == nil || err.Error() != tc.want {
			t.Fatalf("got %v want %q", err, tc.want)
		}
	}
}
