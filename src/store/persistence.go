package store

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/yourname/expense-tracker/src/models"
)

type Persister struct{ path string }

func NewPersister(path string) *Persister { return &Persister{path: path} }

func (p *Persister) Load() ([]models.Expense, error) {
	data, err := os.ReadFile(p.path)
	if errors.Is(err, os.ErrNotExist) {
		log.Println("no existing data file found, starting fresh")
		return []models.Expense{}, nil
	}
	if err != nil {
		return nil, err
	}
	var expenses []models.Expense
	if err := json.Unmarshal(data, &expenses); err != nil {
		backup := fmt.Sprintf("%s.corrupt-%d", p.path, time.Now().Unix())
		if renameErr := os.Rename(p.path, backup); renameErr != nil {
			return nil, fmt.Errorf("back up corrupt data: %w", renameErr)
		}
		log.Printf("data file was corrupt and has been backed up to %s", backup)
		return []models.Expense{}, nil
	}
	if expenses == nil {
		expenses = []models.Expense{}
	}
	return expenses, nil
}

func (p *Persister) Save(expenses []models.Expense) error {
	if err := os.MkdirAll(filepath.Dir(p.path), 0755); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(p.path), filepath.Base(p.path)+".tmp-")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	// renamed tracks whether the atomic rename succeeded. The deferred Remove
	// only cleans up the temp file if we never renamed it; after a successful
	// Rename the file lives at p.path and must not be removed.
	renamed := false
	defer func() {
		if !renamed {
			os.Remove(tmpName)
		}
	}()
	encoder := json.NewEncoder(tmp)
	if err := encoder.Encode(expenses); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Sync(); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := os.Rename(tmpName, p.path); err != nil {
		return err
	}
	renamed = true
	return nil
}
