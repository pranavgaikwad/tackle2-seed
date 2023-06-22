package pkg

// TagCategory is a representation of the Hub's TagCategory that is fit for seeding.
type TagCategory struct {
	UUID  string
	Name  string
	Color string
	Tags  []struct {
		UUID string
		Name string
	}
}
