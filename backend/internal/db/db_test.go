package db

import (
	"errors"
	"testing"

	"github.com/lib/pq"
)

func TestMissingDatabaseName(t *testing.T) {
	t.Run("does not exist error", func(t *testing.T) {
		err := &pq.Error{Code: "3D000", Message: `database "bia_energy" does not exist`}
		name, ok := missingDatabaseName(err)
		if !ok || name != "bia_energy" {
			t.Fatalf("got (%q, %v), want (bia_energy, true)", name, ok)
		}
	})

	t.Run("unrelated pq error", func(t *testing.T) {
		err := &pq.Error{Code: "42501", Message: "permission denied"}
		if _, ok := missingDatabaseName(err); ok {
			t.Fatal("expected ok=false for a non-3D000 error")
		}
	})

	t.Run("non-pq error", func(t *testing.T) {
		if _, ok := missingDatabaseName(errors.New("boom")); ok {
			t.Fatal("expected ok=false for a plain error")
		}
	})
}
