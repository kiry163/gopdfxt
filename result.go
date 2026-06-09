package gopdfxt

type Result struct {
	Articles []Article
}

type Article struct {
	Title   string
	Content string
	Pages   PageRange
}

type PageRange struct {
	Start int
	End   int
}
