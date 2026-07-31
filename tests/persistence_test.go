package tests

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/yourname/expense-tracker/src/models"
	"github.com/yourname/expense-tracker/src/store"
)

func TestPersister_Load_NoExistingFile_ReturnsEmptySlice(t *testing.T) {
	got, err := store.NewPersister(filepath.Join(t.TempDir(), "none.json")).Load()
	if err != nil || len(got) != 0 || got == nil {
		t.Fatal(got, err)
	}
}
func TestPersister_Load_ValidFile_ReturnsExpensesInOrder(t *testing.T) {
	p := store.NewPersister(filepath.Join(t.TempDir(), "x.json"))
	want := []models.Expense{expense("a"), expense("b")}
	_ = p.Save(want)
	got, err := p.Load()
	if err != nil || !reflect.DeepEqual(got, want) {
		t.Fatal(got, err)
	}
}
func TestPersister_Load_CorruptFile_BacksUpAndReturnsEmpty(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "x.json")
	if err := os.WriteFile(path, []byte("garbage"), 0644); err != nil {
		t.Fatal(err)
	}
	got, err := store.NewPersister(path).Load()
	matches, _ := filepath.Glob(path + ".corrupt-*")
	if err != nil || len(got) != 0 || len(matches) != 1 {
		t.Fatal(got, err, matches)
	}
}
func TestPersister_Save_WritesAtomically(t *testing.T) {
	dir := t.TempDir()
	p := store.NewPersister(filepath.Join(dir, "x.json"))
	if err := p.Save([]models.Expense{expense("a")}); err != nil {
		t.Fatal(err)
	}
	matches, _ := filepath.Glob(filepath.Join(dir, "*.tmp-*"))
	if len(matches) != 0 {
		t.Fatal(matches)
	}
}
func TestPersister_Save_ThenLoad_RoundTripsCorrectly(t *testing.T) {
	p := store.NewPersister(filepath.Join(t.TempDir(), "x.json"))
	want := []models.Expense{expense("a")}
	if err := p.Save(want); err != nil {
		t.Fatal(err)
	}
	got, err := p.Load()
	if err != nil || !reflect.DeepEqual(got, want) {
		t.Fatal(got, err)
	}
}
func TestStore_LoadFromDisk_PopulatesStoreOnStartup(t *testing.T) {
	p := store.NewPersister(filepath.Join(t.TempDir(), "x.json"))
	_ = p.Save([]models.Expense{expense("a")})
	s := store.NewExpenseStore(p)
	if err := s.LoadFromDisk(); err != nil || len(s.All()) != 1 {
		t.Fatal(err, s.All())
	}
}
