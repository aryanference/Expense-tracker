package store

import (
	"sync"

	"github.com/yourname/expense-tracker/src/models"
)

type ExpenseStore struct {
	mu        sync.RWMutex
	expenses  map[string]models.Expense
	order     []string
	persister *Persister
}

func NewExpenseStore(p *Persister) *ExpenseStore {
	return &ExpenseStore{expenses: make(map[string]models.Expense), order: []string{}, persister: p}
}

func (s *ExpenseStore) Add(e models.Expense) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.expenses[e.ID] = e
	s.order = append(s.order, e.ID)
	// A persistence failure intentionally does not roll back the in-memory change.
	// The caller (service layer) wraps the error as ErrPersistFailed so the client
	// receives a 500 and knows the change may not survive a restart.
	return s.saveLocked()
}

func (s *ExpenseStore) All() []models.Expense {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.allLocked()
}

func (s *ExpenseStore) Delete(id string) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.expenses[id]; !ok {
		return false, nil
	}
	delete(s.expenses, id)
	for i, existingID := range s.order {
		if existingID == id {
			s.order = append(s.order[:i], s.order[i+1:]...)
			break
		}
	}
	return true, s.saveLocked()
}

func (s *ExpenseStore) LoadFromDisk() error {
	if s.persister == nil {
		return nil
	}
	expenses, err := s.persister.Load()
	if err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.expenses = make(map[string]models.Expense, len(expenses))
	s.order = make([]string, 0, len(expenses))
	for _, expense := range expenses {
		s.expenses[expense.ID] = expense
		s.order = append(s.order, expense.ID)
	}
	return nil
}

func (s *ExpenseStore) allLocked() []models.Expense {
	result := make([]models.Expense, 0, len(s.order))
	for _, id := range s.order {
		if expense, ok := s.expenses[id]; ok {
			result = append(result, expense)
		}
	}
	return result
}

func (s *ExpenseStore) saveLocked() error {
	if s.persister == nil {
		return nil
	}
	return s.persister.Save(s.allLocked())
}
