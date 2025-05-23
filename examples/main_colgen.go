// Code generated by colgen devel; DO NOT EDIT.
package main

type NewsList []News

func (ll NewsList) IDs() []int {
	r := make([]int, len(ll))
	for i := range ll {
		r[i] = ll[i].ID
	}
	return r
}

func (ll NewsList) Index() map[int]News {
	r := make(map[int]News, len(ll))
	for i := range ll {
		r[ll[i].ID] = ll[i]
	}
	return r
}

func (ll NewsList) IndexByTitle() map[string]News {
	r := make(map[string]News, len(ll))
	for i := range ll {
		r[ll[i].Title] = ll[i]
	}
	return r
}

func (ll NewsList) GroupByTitle() map[string]NewsList {
	r := make(map[string]NewsList, len(ll))
	for i := range ll {
		r[ll[i].Title] = append(r[ll[i].Title], ll[i])
	}
	return r
}

func (ll NewsList) UniqueTitles() []string {
	idx := make(map[string]struct{}, len(ll))
	for i := range ll {
		if _, ok := idx[ll[i].Title]; !ok {
			idx[ll[i].Title] = struct{}{}
		}
	}

	r, i := make([]string, len(idx)), 0
	for k := range idx {
		r[i] = k
		i++
	}
	return r
}

func (ll NewsList) UniqueTagIDs() []int {
	idx := make(map[int]struct{}, len(ll))
	for i := range ll {
		for _, v := range ll[i].TagIDs {
			if _, ok := idx[v]; !ok {
				idx[v] = struct{}{}
			}
		}
	}

	r, i := make([]int, len(idx)), 0
	for k := range idx {
		r[i] = k
		i++
	}
	return r
}

type Tags []Tag

func (ll Tags) IDs() []int {
	r := make([]int, len(ll))
	for i := range ll {
		r[i] = ll[i].ID
	}
	return r
}

func (ll Tags) Index() map[int]Tag {
	r := make(map[int]Tag, len(ll))
	for i := range ll {
		r[ll[i].ID] = ll[i]
	}
	return r
}
