package pkg

// Questionnaire is a representation of the Hub's Questionnaire model that is fit for seeding.
type Questionnaire struct {
	UUID         string
	Name         string
	Description  string
	Required     bool
	Sections     []Section
	Thresholds   Thresholds
	RiskMessages RiskMessages
}

// Section represents a group of questions in a questionnaire.
type Section struct {
	Order     uint
	Name      string
	Questions []Question
}

// Question represents a question in a questionnaire.
type Question struct {
	Order       uint
	Text        string
	Explanation string
	IncludeFor  []CategorizedTag
	ExcludeFor  []CategorizedTag
	Answers     []Answer
}

// Answer represents an answer to a question in a questionnaire.
type Answer struct {
	Order         uint
	Text          string
	Risk          string
	Rationale     string
	Mitigation    string
	ApplyTags     []CategorizedTag
	AutoAnswerFor []CategorizedTag
	Selected      bool
	AutoAnswered  bool
}

// CategorizedTag represents a human-readable pair of category and tag.
type CategorizedTag struct {
	Category string
	Tag      string
}

// RiskMessages contains messages to display for each risk level.
type RiskMessages struct {
	Red     string
	Yellow  string
	Green   string
	Unknown string
}

// Thresholds contains the threshold values for determining risk for the questionnaire.
type Thresholds struct {
	Red     uint
	Yellow  uint
	Unknown uint
}
