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
	"ACT":  {ReadableName: "active", SanskritName: "kartari"},
	"MED":  {ReadableName: "middle voice", SanskritName: "ātmanepada"},
	"PASS": {ReadableName: "passive voice", SanskritName: "karmaṇi"},

	// Tenses/Moods
	"AOR":    {ReadableName: "aorist", SanskritName: "luṅ", BackgroundColor: "#fff3cd", Color: "#000"},
	"COND":   {ReadableName: "conditional", SanskritName: "lṛṅ"},
	"FUT":    {ReadableName: "future", SanskritName: "lṛṭ", BackgroundColor: "#f8d7da", Color: "#000"},
	"IMP":    {ReadableName: "imperative", SanskritName: "loṭ"},
	"IND":    {ReadableName: "indicative", SanskritName: "nirdeśa"},
	"INF":    {ReadableName: "infinitive", SanskritName: "tumun"},
	"INJ":    {ReadableName: "injuctive", SanskritName: "āgamarahita luṅ"},
	"IPRF":   {ReadableName: "imperfect", SanskritName: "laṅ", BackgroundColor: "#e2e3e5", Color: "#000"},
	"OPT":    {ReadableName: "optative", SanskritName: "vidhiliṅ"},
	"PLUPRF": {ReadableName: "past perfect", SanskritName: "luṅ (sometimes)", BackgroundColor: "#cff4fc", Color: "#000"},
	"PRF":    {ReadableName: "perfect", SanskritName: "liṭ", BackgroundColor: "#d1e7dd", Color: "#000"},
	"PRS":    {ReadableName: "present", SanskritName: "laṭ", BackgroundColor: "#cfe2ff", Color: "#000"},
	"SBJV":   {ReadableName: "subjunctive", SanskritName: "leṭ"},

	// Numbers
	"DU": {ReadableName: "dual", SanskritName: "dvivacanam", BorderStyle: "secondary"},
	"PL": {ReadableName: "plural", SanskritName: "bahuvacanam", BorderStyle: "success"},
	"SG": {ReadableName: "singular", SanskritName: "ekavacanam", BorderStyle: "primary"},

	// Genders
	"F": {ReadableName: "feminine", SanskritName: "strīliṅga", UnderlineStyle: "dotted"},
	"M": {ReadableName: "masculine", SanskritName: "puṃlliṅga", UnderlineStyle: "dashed"},
	"N": {ReadableName: "neuter", SanskritName: "napuṃsakaliṅga"},

	// Participles and other
	"CVB":  {ReadableName: "converb", SanskritName: "ktvā"},
	"PPP":  {ReadableName: "participle perfective passive", SanskritName: "kta"},
	"PTCP": {ReadableName: "participle", SanskritName: "kṛdanta"},
}
