package main

//go:generate go run ../cmd/colgen/colgen.go

//colgen:News,Tag
//colgen:News:Index(Title),Group(Title),UniqueTitle,UniqueTagIDs

func main() {

}

type News struct {
	ID     int
	Title  string
	URL    string
	TagIDs []int
	Tags   []Tag
}

type Tag struct {
	ID   int
	Name string
}
