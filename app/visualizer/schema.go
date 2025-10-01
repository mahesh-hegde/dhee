package visualizer

type ChartType string

const (
	Bar     ChartType = "bar"
	Scatter ChartType = "scatter"
)

type VisualizationRequest struct {
	ChartType        ChartType
	Words            []string
	IncludeWordForms bool   // Include all forms of the root word
	GroupBy          string // one of the strings common to hierarchy of all texts, author, text or meter
	Sources          []string
}

type IncludedWord struct {
	Word  string
	Forms []string
}

type Point struct {
	X          string
	Y          string
	Color      string
	SourceName string
	Path       []string
}

// Temporary draft: to be refined based on what kind of data is supported by d3.js / uplot to draw
// charts.
type VisualizationResponse struct {
	Words  []IncludedWord
	AxisX  []string
	AxisY  []string
	Points []Point
	Bars   map[string]string
}
