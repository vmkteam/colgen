package colgen

// test types.
type (
	Author struct {
		ID   int
		Name string
	}

	Category struct {
		ID int
	}

	News struct {
		ID       int
		Title    string
		StatusID int
		TagIDs   []int
		Tags     []Tag
		AuthorID *int
		Author   *Author
	}

	Tag struct {
		ID          int
		OrderNumber int64
		Name        string
	}
)
