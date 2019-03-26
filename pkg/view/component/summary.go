package component

import "encoding/json"

// SummaryConfig is the contents of a Summary
type SummaryConfig struct {
	Sections []SummarySection `json:"sections"`
}

// SummarySection is a section within a summary
type SummarySection struct {
	Header  string    `json:"header"`
	Content Component `json:"content"`
}

// SummarySections is a slice of summary sections
type SummarySections []SummarySection

func (s *SummarySections) Add(header string, view Component) {
	*s = append(*s, SummarySection{
		Header:  header,
		Content: view,
	})
}

// AddText adds a section with a single text component
func (s *SummarySections) AddText(header string, text string) {
	*s = append(*s, SummarySection{
		Header:  header,
		Content: NewText(text),
	})
}

func (t *SummarySection) UnmarshalJSON(data []byte) error {
	x := struct {
		Header  string      `json:"header,omitempty"`
		Content TypedObject `json:"content,omitempty"`
	}{}

	if err := json.Unmarshal(data, &x); err != nil {
		return err
	}

	t.Header = x.Header
	var err error
	t.Content, err = x.Content.ToComponent()
	if err != nil {
		return err
	}

	return nil
}

// Summary contains other Components
type Summary struct {
	base
	Config SummaryConfig `json:"config"`
}

// NewSummary creates a summary component
func NewSummary(title string, sections ...SummarySection) *Summary {
	s := append([]SummarySection(nil), sections...) // Make a copy
	return &Summary{
		base: newBase(typeSummary, TitleFromString(title)),
		Config: SummaryConfig{
			Sections: s,
		},
	}
}

// GetMetadata accesses the components metadata. Implements Component.
func (t *Summary) GetMetadata() Metadata {
	return t.Metadata
}

// Add adds additional items to the tail of the summary.
func (t *Summary) Add(sections ...SummarySection) {
	t.Config.Sections = append(t.Config.Sections, sections...)
}

type summaryMarshal Summary

// MarshalJSON implements json.Marshaler
func (t *Summary) MarshalJSON() ([]byte, error) {
	m := summaryMarshal(*t)
	m.Metadata.Type = "summary"
	return json.Marshal(&m)
}