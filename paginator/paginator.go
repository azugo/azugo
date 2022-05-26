package paginator

import (
	"net/url"
)

var (
	// QueryParameterPage is URL query parameter to specify page number.
	QueryParameterPage = "page"
	// QueryParameterPage is URL query parameter to specify page size.
	QueryParameterPerPage = "per_page"
)

// Paginator represents a set of results of pagination calculations.
type Paginator struct {
	total    int
	pageSize int
	current  int
	pageURL  *url.URL
}

// New initialize a new pagination calculation and returns a Paginator as result.
func New(total, pageSize, current int) *Paginator {
	if pageSize <= 0 {
		pageSize = 1
	}
	if current <= 0 {
		current = 1
	}
	p := &Paginator{
		total:    total,
		pageSize: pageSize,
		current:  current,
	}
	if p.current > p.TotalPages() {
		p.current = p.TotalPages()
	}
	return p
}

// SetURL sets the page URL used for link generation.
func (p *Paginator) SetURL(u *url.URL) {
	p.pageURL = u
}

// GetURL gets the page URL used for link generation.
func (p *Paginator) GetURL() *url.URL {
	return p.pageURL
}

// IsFirst returns true if current page is the first page.
func (p *Paginator) IsFirst() bool {
	return p.current == 1
}

// HasPrevious returns true if there is a previous page relative to current page.
func (p *Paginator) HasPrevious() bool {
	return p.current > 1
}
func (p *Paginator) Previous() int {
	if !p.HasPrevious() {
		return p.current
	}
	return p.current - 1
}

// HasNext returns true if there is a next page relative to current page.
func (p *Paginator) HasNext() bool {
	return p.total > p.current*p.pageSize
}
func (p *Paginator) Next() int {
	if !p.HasNext() {
		return p.current
	}
	return p.current + 1
}

// IsLast returns true if current page is the last page.
func (p *Paginator) IsLast() bool {
	if p.total == 0 {
		return true
	}
	return p.total > (p.current-1)*p.pageSize && !p.HasNext()
}

// Total returns number of total rows.
func (p *Paginator) Total() int {
	return p.total
}

// TotalPage returns number of total pages.
func (p *Paginator) TotalPages() int {
	if p.total == 0 {
		return 1
	}
	if p.total%p.pageSize == 0 {
		return p.total / p.pageSize
	}
	return p.total/p.pageSize + 1
}

// Current returns current page number.
func (p *Paginator) Current() int {
	return p.current
}

// PageSize returns page size.
func (p *Paginator) PageSize() int {
	return p.pageSize
}
