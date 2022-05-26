package paginator_test

import (
	"net/url"
	"testing"

	"azugo.io/azugo/paginator"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPaginatorNew(t *testing.T) {
	total := 105
	pageSize := 20
	current := 2
	paginator := paginator.New(total, pageSize, current)

	assert.Equal(t, total, paginator.Total(), "wrong paginator Total value")
	assert.Equal(t, pageSize, paginator.PageSize(), "wrong paginator PageSize value")
	assert.Equal(t, current, paginator.Current(), "wrong paginator Current value")
	assert.Equal(t, total/pageSize+1, paginator.TotalPages(), "wrong paginator TotalPages value")
	assert.False(t, paginator.IsFirst(), "wrong paginator IsFirst value")
	assert.False(t, paginator.IsLast(), "wrong paginator IsLast value")
	assert.True(t, paginator.HasNext(), "wrong paginator HasNext value")
	assert.True(t, paginator.HasPrevious(), "wrong paginator HasPrevious value")
	assert.Equal(t, current-1, paginator.Previous(), "wrong paginator Previous value")
	assert.Equal(t, current+1, paginator.Next(), "wrong paginator Next value")
}

func TestPaginatorNewEmpty(t *testing.T) {
	total := 0
	pageSize := 0
	current := 0
	paginator := paginator.New(total, pageSize, current)

	assert.Equal(t, 0, paginator.Total(), "wrong paginator Total value")
	assert.Equal(t, 1, paginator.PageSize(), "wrong paginator PageSize value")
	assert.Equal(t, 1, paginator.Current(), "wrong paginator Current value")
	assert.Equal(t, 1, paginator.TotalPages(), "wrong paginator TotalPages value")
	assert.True(t, paginator.IsFirst(), "wrong paginator IsFirst value")
	assert.True(t, paginator.IsLast(), "wrong paginator IsLast value")
	assert.False(t, paginator.HasNext(), "wrong paginator HasNext value")
	assert.False(t, paginator.HasPrevious(), "wrong paginator HasPrevious value")
	assert.Equal(t, 1, paginator.Previous(), "wrong paginator Previous value")
	assert.Equal(t, 1, paginator.Next(), "wrong paginator Next value")
}

func TestPaginatorCurrent(t *testing.T) {
	pag := paginator.New(1, 1, 2)
	assert.Equal(t, 1, pag.Current(), "wrong paginator Current value")
}

func TestPaginatorSetURL(t *testing.T) {
	paginator := paginator.New(1, 1, 1)
	testurl, err := url.Parse("http://localhost:3000")
	require.NoError(t, err)
	paginator.SetURL(testurl)
	assert.Equal(t, testurl, paginator.GetURL())
}

func TestPaginatorLinks(t *testing.T) {
	total := 105
	pageSize := 20
	current := 2
	paginator := paginator.New(total, pageSize, current)
	testurl, err := url.Parse("http://localhost:3000")
	require.NoError(t, err)
	paginator.SetURL(testurl)

	links := paginator.Links()
	assert.Equal(t, 4, len(links), "wrong paginator Links count")

	link := "<http://localhost:3000?page=3&per_page=20>; rel=\"next\""
	assert.Equal(t, link, links[0], "wrong paginator next link")
}
