package schema

import (
	"database/sql/driver"
	"encoding/json"
	"slices"
)

type (
	Scenarios []string
)

func (n Scenarios) Has(str string) bool {
	return slices.Contains(n, str)
}

func (n Scenarios) Value() (driver.Value, error) {
	return json.Marshal(n)
}

// Scan implements the Scanner interface.
func (n *Scenarios) Scan(value any) error {
	if value == nil {
		return nil
	}
	switch s := value.(type) {
	case string:
		return json.Unmarshal([]byte(s), n)
	case []byte:
		return json.Unmarshal(s, n)
	}
	return ErrUnsupportType
}
