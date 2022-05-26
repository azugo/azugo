package paginator

import (
	"fmt"
	"strconv"
)

// Links return the pagination links
func (p *Paginator) Links() []string {
	links := make([]string, 0, 4)
	if p.HasNext() {
		u := *p.pageURL
		queries := u.Query()
		queries.Set(QueryParameterPage, strconv.Itoa(p.Next()))
		queries.Set(QueryParameterPerPage, strconv.Itoa(p.pageSize))
		u.RawQuery = queries.Encode()
		links = append(links, fmt.Sprintf("<%s>; rel=\"next\"", u.String()))
	}
	if !p.IsLast() {
		u := *p.pageURL
		queries := u.Query()
		queries.Set(QueryParameterPage, strconv.Itoa(p.TotalPages()))
		queries.Set(QueryParameterPerPage, strconv.Itoa(p.pageSize))
		u.RawQuery = queries.Encode()
		links = append(links, fmt.Sprintf("<%s>; rel=\"last\"", u.String()))
	}
	if !p.IsFirst() {
		u := *p.pageURL
		queries := u.Query()
		queries.Set(QueryParameterPage, "1")
		queries.Set(QueryParameterPerPage, strconv.Itoa(p.pageSize))
		u.RawQuery = queries.Encode()
		links = append(links, fmt.Sprintf("<%s>; rel=\"first\"", u.String()))
	}
	if p.HasPrevious() {
		u := *p.pageURL
		queries := u.Query()
		queries.Set(QueryParameterPage, strconv.Itoa(p.Previous()))
		queries.Set(QueryParameterPerPage, strconv.Itoa(p.pageSize))
		u.RawQuery = queries.Encode()
		links = append(links, fmt.Sprintf("<%s>; rel=\"prev\"", u.String()))
	}
	return links
}
