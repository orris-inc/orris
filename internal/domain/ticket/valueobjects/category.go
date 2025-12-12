package valueobjects

import "fmt"

type Category string

const (
	CategoryTechnical Category = "technical"
	CategoryAccount   Category = "account"
	CategoryBilling   Category = "billing"
	CategoryFeature   Category = "feature"
	CategoryComplaint Category = "complaint"
	CategoryOther     Category = "other"
)

var validCategories = map[Category]bool{
	CategoryTechnical: true,
	CategoryAccount:   true,
	CategoryBilling:   true,
	CategoryFeature:   true,
	CategoryComplaint: true,
	CategoryOther:     true,
}

func (c Category) String() string {
	return string(c)
}

func (c Category) IsValid() bool {
	return validCategories[c]
}

func (c Category) IsTechnical() bool {
	return c == CategoryTechnical
}

func (c Category) IsAccount() bool {
	return c == CategoryAccount
}

func (c Category) IsBilling() bool {
	return c == CategoryBilling
}

func (c Category) IsFeature() bool {
	return c == CategoryFeature
}

func (c Category) IsComplaint() bool {
	return c == CategoryComplaint
}

func (c Category) IsOther() bool {
	return c == CategoryOther
}

func NewCategory(s string) (Category, error) {
	c := Category(s)
	if !c.IsValid() {
		return "", fmt.Errorf("invalid category: %s", s)
	}
	return c, nil
}
