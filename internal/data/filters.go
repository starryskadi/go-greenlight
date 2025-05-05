package data

import (
	"fmt"
	"strings"

	"kyawzayarwin.com/greenlight/internal/validator"
)

type Filters struct {
	Page int 
	PageSize int 
	Sort string
	SortSafelist []string
}

func ValidateFilter(v *validator.Validator, f Filters) {
	v.Check(f.Page > 0, "page", "must be greater than zero") 
	v.Check(f.Page <= 10_000_000, "page", "must be lesser than 10000") 
	v.Check(f.PageSize > 0, "page_size", "must be greater than zero")
	v.Check(f.PageSize <= 100, "page_size", "must be a maximum of 100")

	v.Check(validator.In(f.Sort, f.SortSafelist...), "sort", fmt.Sprintf("invalid sort value, must be one of %s", strings.Join(f.SortSafelist, ",")))
}