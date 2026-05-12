package schema

import (
	"database/sql/driver"
	"encoding/json"
)

type Rule struct {
	Min      int      `json:"min"`
	Max      int      `json:"max"`
	Type     string   `json:"type"`
	Unique   bool     `json:"unique"`
	Required []string `json:"required"`
	Regular  string   `json:"regular,omitempty"`
	Safe     bool     `json:"safe,omitempty"`
}

func (n *Rule) Scan(value any) error {
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

// Value implements the driver Valuer interface.
func (n Rule) Value() (driver.Value, error) {
	return json.Marshal(n)
}
