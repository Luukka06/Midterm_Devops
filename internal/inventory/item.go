package inventory

import (
	"errors"
	"strings"
	"time"
	"unicode/utf8"
)

var (
	ErrInvalid  = errors.New("invalid item")
	ErrNotFound = errors.New("item not found")
)

type Item struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Category  string    `json:"category"`
	Quantity  int       `json:"quantity"`
	MinStock  int       `json:"min_stock"`
	Unit      string    `json:"unit"`
	UpdatedAt time.Time `json:"updated_at"`
}

type ItemInput struct {
	Name     string `json:"name"`
	Category string `json:"category"`
	Quantity int    `json:"quantity"`
	MinStock int    `json:"min_stock"`
	Unit     string `json:"unit"`
}

type UpdateInput struct {
	Quantity *int `json:"quantity"`
	MinStock *int `json:"min_stock"`
}

type Summary struct {
	TotalItems int `json:"total_items"`
	TotalUnits int `json:"total_units"`
	LowStock   int `json:"low_stock"`
	OutOfStock int `json:"out_of_stock"`
}

func (in ItemInput) Normalize() (ItemInput, error) {
	in.Name = strings.TrimSpace(in.Name)
	in.Category = strings.TrimSpace(in.Category)
	in.Unit = strings.TrimSpace(in.Unit)
	if in.Unit == "" {
		in.Unit = "pcs"
	}
	if in.Category == "" {
		in.Category = "General"
	}
	if in.Name == "" || utf8.RuneCountInString(in.Name) > 70 ||
		utf8.RuneCountInString(in.Category) > 40 || utf8.RuneCountInString(in.Unit) > 16 ||
		in.Quantity < 0 || in.MinStock < 0 || in.Quantity > 1000000 || in.MinStock > 1000000 {
		return ItemInput{}, ErrInvalid
	}
	return in, nil
}

// Status is the same rule used by the dashboard and the unit tests.
func Status(item Item) string {
	if item.Quantity == 0 {
		return "out"
	}
	if item.Quantity <= item.MinStock {
		return "low"
	}
	return "healthy"
}

func Summarize(items []Item) Summary {
	s := Summary{TotalItems: len(items)}
	for _, item := range items {
		s.TotalUnits += item.Quantity
		switch Status(item) {
		case "out":
			s.OutOfStock++
		case "low":
			s.LowStock++
		}
	}
	return s
}

func DemoItems() []Item {
	now := time.Now().UTC()
	examples := []struct {
		id, name, category, unit string
		qty, min                 int
	}{
		{"SP-101", "Mechanical keyboard", "Electronics", "pcs", 24, 8},
		{"SP-102", "USB-C docking station", "Electronics", "pcs", 6, 8},
		{"SP-103", "A4 recycled paper", "Office", "packs", 47, 12},
		{"SP-104", "Desk organizer", "Office", "pcs", 18, 5},
		{"SP-105", "Wireless mouse", "Electronics", "pcs", 0, 6},
		{"SP-106", "Cleaning spray", "Supplies", "bottles", 11, 4},
		{"SP-107", "Shipping labels", "Supplies", "rolls", 32, 10},
	}
	items := make([]Item, 0, len(examples))
	for _, example := range examples {
		items = append(items, Item{
			ID: example.id, Name: example.name, Category: example.category,
			Unit: example.unit, Quantity: example.qty, MinStock: example.min,
			UpdatedAt: now,
		})
	}
	return items
}
