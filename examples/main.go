package main

//go:generate go run ../cmd/colgen/colgen.go

//colgen:News,Tag,Author
//colgen:News:Index(Title),Group(Title),UniqueTitle,UniqueTagIDs,UniqueAuthorID,Group(TagIDs),Count(StatusID),Fill(Tags,TagIDs),Fill(Author,AuthorID),Cast(Searchable)

func main() {

}

type News struct {
	ID       int
	Title    string
	URL      string
	StatusID int
	TagIDs   []int
	Tags     []Tag
	AuthorID *int
	Author   *Author
}

type Tag struct {
	ID   int
	Name string
}

type Author struct {
	ID   int
	Name string
}

type Searchable interface {
	SearchTitle() string
}

func (n *News) SearchTitle() string { return n.Title }
