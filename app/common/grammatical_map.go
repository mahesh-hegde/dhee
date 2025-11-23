package common

type GrammaticalTagStyle struct {
	ReadableName    string
	SanskritName    string
	BackgroundColor string
	Color           string
	UnderlineStyle  string // "dotted", "dashed", "none"
	BorderStyle     string // "primary", "secondary", etc. for border color
}

var GrammaticalTags = map[string]GrammaticalTagStyle{
	// Persons
	"1": {ReadableName: "first person", SanskritName: "उत्तमपुरुषः", Color: "var(--bs-primary-text-emphasis)"},
	"2": {ReadableName: "second person", SanskritName: "मध्यमपुरुषः", Color: "var(--bs-success-text-emphasis)"},
	"3": {ReadableName: "third person", SanskritName: "प्रथमपुरुषः", Color: "var(--bs-secondary-text-emphasis)"},

	// Cases
	"ABL": {ReadableName: "ablative", SanskritName: "पञ्चमी", BackgroundColor: "#cfe2ff", Color: "#000"},
	"ACC": {ReadableName: "accusative", SanskritName: "द्वितीया", BackgroundColor: "#e2e3e5", Color: "#000"},
	"DAT": {ReadableName: "dative", SanskritName: "चतुर्थी", BackgroundColor: "#d1e7dd", Color: "#000"},
	"GEN": {ReadableName: "genitive", SanskritName: "षष्ठी", BackgroundColor: "#f8d7da", Color: "#000"},
	"INS": {ReadableName: "instrumental", SanskritName: "तृतीया", BackgroundColor: "#fff3cd", Color: "#000"},
	"LOC": {ReadableName: "locative", SanskritName: "सप्तमी", BackgroundColor: "#cff4fc", Color: "#000"},
	"NOM": {ReadableName: "nominative", SanskritName: "प्रथमा", BackgroundColor: "#f8f9fa", Color: "#000"},
	"VOC": {ReadableName: "vocative", SanskritName: "संबोधन", BackgroundColor: "#ced4da", Color: "#000"},

	// Voices
	"ACT":  {ReadableName: "active", SanskritName: "कर्तरि"},
	"MED":  {ReadableName: "middle voice", SanskritName: "आत्मनेपद"},
	"PASS": {ReadableName: "passive voice", SanskritName: "कर्मणि"},

	// Tenses/Moods
	"AOR":    {ReadableName: "aorist", SanskritName: "लुङ्", BackgroundColor: "#fff3cd", Color: "#000"},
	"COND":   {ReadableName: "conditional", SanskritName: "लृङ्"},
	"FUT":    {ReadableName: "future", SanskritName: "लृट्", BackgroundColor: "#f8d7da", Color: "#000"},
	"IMP":    {ReadableName: "imperative", SanskritName: "लोट्"},
	"IND":    {ReadableName: "indicative", SanskritName: "निर्देश"},
	"INF":    {ReadableName: "infinitive", SanskritName: "तुमुन्"},
	"INJ":    {ReadableName: "injuctive", SanskritName: "आगमरहित लुङ्"},
	"IPRF":   {ReadableName: "imperfect", SanskritName: "लङ्", BackgroundColor: "#e2e3e5", Color: "#000"},
	"OPT":    {ReadableName: "optative", SanskritName: "विधिलिङ्"},
	"PLUPRF": {ReadableName: "past perfect", SanskritName: "लुङ् (sometimes)", BackgroundColor: "#cff4fc", Color: "#000"},
	"PRF":    {ReadableName: "perfect", SanskritName: "लिट्", BackgroundColor: "#d1e7dd", Color: "#000"},
	"PRS":    {ReadableName: "present", SanskritName: "लट्", BackgroundColor: "#cfe2ff", Color: "#000"},
	"SBJV":   {ReadableName: "subjunctive", SanskritName: "लेट्"},

	// Numbers
	"DU": {ReadableName: "dual", SanskritName: "द्विवचनम्", BorderStyle: "secondary"},
	"PL": {ReadableName: "plural", SanskritName: "बहुवचनम्", BorderStyle: "success"},
	"SG": {ReadableName: "singular", SanskritName: "एकवचनम्", BorderStyle: "primary"},

	// Genders
	"F": {ReadableName: "feminine", SanskritName: "स्त्रीलिङ्ग", UnderlineStyle: "dotted"},
	"M": {ReadableName: "masculine", SanskritName: "पुंल्लिङ्ग", UnderlineStyle: "dashed"},
	"N": {ReadableName: "neuter", SanskritName: "नपुंसकलिङ्ग"},

	// Participles and other
	"CVB":  {ReadableName: "converb", SanskritName: "क्त्वा"},
	"PPP":  {ReadableName: "participle perfective passive", SanskritName: "क्त"},
	"PTCP": {ReadableName: "participle", SanskritName: "कृदन्त"},
}
