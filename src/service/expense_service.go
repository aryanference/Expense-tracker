package service

import (
	"errors"
	"fmt"
	"math"
	"strings"

	"github.com/yourname/expense-tracker/src/models"
	"github.com/yourname/expense-tracker/src/store"
)

var ErrExpenseNotFound = errors.New("expense not found")
var ErrPersistFailed = errors.New("failed to save expense data")

type ExpenseService struct{ store *store.ExpenseStore }

func NewExpenseService(s *store.ExpenseStore) *ExpenseService { return &ExpenseService{store: s} }

func (svc *ExpenseService) AddExpense(req models.CreateExpenseRequest) (models.Expense, error) {
	validated, err := models.ValidateCreateRequest(req)
	if err != nil {
		return models.Expense{}, err
	}
	id, err := models.NewID()
	if err != nil {
		return models.Expense{}, ErrPersistFailed
	}
	expense := models.Expense{ID: id, Title: validated.Title, Amount: validated.Amount, Category: validated.Category, Date: validated.Date}
	if err := svc.store.Add(expense); err != nil {
		return models.Expense{}, fmtPersist(err)
	}
	return expense, nil
}

func (svc *ExpenseService) ListExpenses(category string) []models.Expense {
	all := svc.store.All()
	category = strings.TrimSpace(category)
	if category == "" {
		return all
	}
	result := make([]models.Expense, 0)
	for _, expense := range all {
		if strings.EqualFold(expense.Category, category) {
			result = append(result, expense)
		}
	}
	return result
}

func (svc *ExpenseService) GetTotal(category string) float64 {
	total := 0.0
	for _, expense := range svc.ListExpenses(category) {
		total += expense.Amount
	}
	return math.Round(total*100) / 100
}

func (svc *ExpenseService) DeleteExpense(id string) error {
	found, err := svc.store.Delete(id)
	if err != nil {
		return fmtPersist(err)
	}
	if !found {
		return ErrExpenseNotFound
	}
	return nil
}
func fmtPersist(err error) error { return fmt.Errorf("%w: %v", ErrPersistFailed, err) }
