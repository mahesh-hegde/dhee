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
	"1": {ReadableName: "first person", SanskritName: "uttamapuruṣaḥ", Color: "var(--bs-primary-text-emphasis)"},
	"2": {ReadableName: "second person", SanskritName: "madhyamapuruṣaḥ", Color: "var(--bs-success-text-emphasis)"},
	"3": {ReadableName: "third person", SanskritName: "prathamapuruṣaḥ", Color: "var(--bs-secondary-text-emphasis)"},

	// Cases
	"ABL": {ReadableName: "ablative", SanskritName: "pañcamī", BackgroundColor: "#cfe2ff", Color: "#000"},
	"ACC": {ReadableName: "accusative", SanskritName: "dvitīyā", BackgroundColor: "#e2e3e5", Color: "#000"},
	"DAT": {ReadableName: "dative", SanskritName: "caturthī", BackgroundColor: "#d1e7dd", Color: "#000"},
	"GEN": {ReadableName: "genitive", SanskritName: "ṣaṣṭhī", BackgroundColor: "#f8d7da", Color: "#000"},
	"INS": {ReadableName: "instrumental", SanskritName: "tṛtīyā", BackgroundColor: "#fff3cd", Color: "#000"},
	"LOC": {ReadableName: "locative", SanskritName: "saptamī", BackgroundColor: "#cff4fc", Color: "#000"},
	"NOM": {ReadableName: "nominative", SanskritName: "prathamā", BackgroundColor: "#f8f9fa", Color: "#000"},
	"VOC": {ReadableName: "vocative", SanskritName: "saṃbodhana", BackgroundColor: "#ced4da", Color: "#000"},

	// Voices
	"ACT":  {ReadableName: "active", SanskritName: ""},
	"MED":  {ReadableName: "middle voice", SanskritName: "ātmanepada"},
	"PASS": {ReadableName: "passive voice", SanskritName: ""},

	// Tenses/Moods
	"AOR":    {ReadableName: "aorist", SanskritName: "", BackgroundColor: "#fff3cd", Color: "#000"},
	"COND":   {ReadableName: "conditional", SanskritName: ""},
	"FUT":    {ReadableName: "future", SanskritName: "", BackgroundColor: "#f8d7da", Color: "#000"},
	"IMP":    {ReadableName: "imperative", SanskritName: ""},
	"IND":    {ReadableName: "indicative", SanskritName: ""},
	"INF":    {ReadableName: "infinitive", SanskritName: ""},
	"INJ":    {ReadableName: "injuctive", SanskritName: ""},
	"IPRF":   {ReadableName: "imperfect", SanskritName: "", BackgroundColor: "#e2e3e5", Color: "#000"},
	"OPT":    {ReadableName: "optative", SanskritName: ""},
	"PLUPRF": {ReadableName: "past perfect", SanskritName: "", BackgroundColor: "#cff4fc", Color: "#000"},
	"PRF":    {ReadableName: "perfect", SanskritName: "", BackgroundColor: "#d1e7dd", Color: "#000"},
	"PRS":    {ReadableName: "present", SanskritName: "", BackgroundColor: "#cfe2ff", Color: "#000"},
	"SBJV":   {ReadableName: "subjunctive", SanskritName: ""},

	// Numbers
	"DU": {ReadableName: "dual", SanskritName: "dvivacanam", BorderStyle: "secondary"},
	"PL": {ReadableName: "plural", SanskritName: "bahuvacanam", BorderStyle: "success"},
	"SG": {ReadableName: "singular", SanskritName: "ekavacanam", BorderStyle: "primary"},

	// Genders
	"F": {ReadableName: "feminine", SanskritName: "strīliṅga", UnderlineStyle: "dotted"},
	"M": {ReadableName: "masculine", SanskritName: "puṃlliṅga", UnderlineStyle: "dashed"},
	"N": {ReadableName: "neuter", SanskritName: "napuṃsakaliṅga"},

	// Participles and other
	"CVB":  {ReadableName: "converb", SanskritName: ""},
	"PPP":  {ReadableName: "participle perfective passive", SanskritName: ""},
	"PTCP": {ReadableName: "participle", SanskritName: ""},
}
