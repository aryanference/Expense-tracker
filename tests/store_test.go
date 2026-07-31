package tests

import (
	"path/filepath"
	"sync"
	"testing"

	"github.com/yourname/expense-tracker/src/models"
	"github.com/yourname/expense-tracker/src/store"
)

func expense(id string) models.Expense {
	return models.Expense{ID: id, Title: id, Amount: 1, Category: "Food", Date: "2026-07-31"}
}
func TestStore_Add_And_All_PreservesOrder(t *testing.T) {
	s := store.NewExpenseStore(nil)
	_ = s.Add(expense("a"))
	_ = s.Add(expense("b"))
	got := s.All()
	if len(got) != 2 || got[0].ID != "a" || got[1].ID != "b" {
		t.Fatal(got)
	}
}
func TestStore_Delete_RemovesCorrectItem(t *testing.T) {
	s := store.NewExpenseStore(nil)
	_ = s.Add(expense("a"))
	_ = s.Add(expense("b"))
	found, err := s.Delete("a")
	if !found || err != nil || len(s.All()) != 1 || s.All()[0].ID != "b" {
		t.Fatal(found, err, s.All())
	}
}
func TestStore_Delete_NonexistentID_ReturnsFalse(t *testing.T) {
	found, err := store.NewExpenseStore(nil).Delete("no")
	if found || err != nil {
		t.Fatal(found, err)
	}
}
func TestStore_ConcurrentAdds_NoLostWrites(t *testing.T) {
	s := store.NewExpenseStore(nil)
	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(i int) { defer wg.Done(); _ = s.Add(expense(string(rune('a' + i)))) }(i)
	}
	wg.Wait()
	if len(s.All()) != 50 {
		t.Fatal(len(s.All()))
	}
}
func TestStore_AddThenDelete_PersistsToDisk(t *testing.T) {
	p := store.NewPersister(filepath.Join(t.TempDir(), "expenses.json"))
	s := store.NewExpenseStore(p)
	_ = s.Add(expense("a"))
	saved, err := p.Load()
	if err != nil || len(saved) != 1 {
		t.Fatal(saved, err)
	}
	_, _ = s.Delete("a")
	saved, err = p.Load()
	if err != nil || len(saved) != 0 {
		t.Fatal(saved, err)
	}
}
