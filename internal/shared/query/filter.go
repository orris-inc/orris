package query

type PageFilter struct {
	Page     int
	PageSize int
}

func (f PageFilter) Offset() int {
	if f.Page <= 0 {
		return 0
	}
	return (f.Page - 1) * f.PageSize
}

func (f PageFilter) Limit() int {
	if f.PageSize <= 0 {
		return 10
	}
	if f.PageSize > 100 {
		return 100
	}
	return f.PageSize
}

func (f PageFilter) Validate() error {
	return nil
}

type SortFilter struct {
	SortBy    string
	SortOrder string
}

func (f SortFilter) IsDescending() bool {
	return f.SortOrder == "desc" || f.SortOrder == "DESC"
}

func (f SortFilter) IsAscending() bool {
	return f.SortOrder == "asc" || f.SortOrder == "ASC" || f.SortOrder == ""
}

func (f SortFilter) OrderClause() string {
	if f.SortBy == "" {
		return ""
	}
	order := "ASC"
	if f.IsDescending() {
		order = "DESC"
	}
	return f.SortBy + " " + order
}

type BaseFilter struct {
	PageFilter
	SortFilter
}

type FilterOption func(*BaseFilter)

func WithPage(page, pageSize int) FilterOption {
	return func(f *BaseFilter) {
		f.Page = page
		f.PageSize = pageSize
	}
}

func WithSort(sortBy, sortOrder string) FilterOption {
	return func(f *BaseFilter) {
		f.SortBy = sortBy
		f.SortOrder = sortOrder
	}
}

func NewBaseFilter(opts ...FilterOption) BaseFilter {
	f := BaseFilter{
		PageFilter: PageFilter{
			Page:     1,
			PageSize: 10,
		},
		SortFilter: SortFilter{
			SortOrder: "DESC",
		},
	}
	for _, opt := range opts {
		opt(&f)
	}
	return f
}
