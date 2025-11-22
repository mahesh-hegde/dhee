// Dhee client-side utilities

// Paste slp1_mappings.json content here as a JS object.
// For example: const slp1MappingsJSON = { ... };
const slp1MappingsJSON = {
    "mappings": {
        "slp1_to_iast": {
            "vowels": { "a": "a", "A": "ā", "i": "i", "I": "ī", "u": "u", "U": "ū", "f": "ṛ", "F": "ṝ", "x": "ḷ", "X": "ḹ", "è": "è", "e": "e", "E": "ai", "ò": "ò", "o": "o", "O": "au" },
            "yogavaahas": { "M": "ṃ", "H": "ḥ", "~": "~", "M£": "m̐" },
            "virama": { "": "" },
            "consonants": { "k": "k", "K": "kh", "g": "g", "G": "gh", "N": "ṅ", "c": "c", "C": "ch", "j": "j", "J": "jh", "Y": "ñ", "w": "ṭ", "W": "ṭh", "q": "ḍ", "Q": "ḍh", "R": "ṇ", "t": "t", "T": "th", "d": "d", "D": "dh", "n": "n", "p": "p", "P": "ph", "b": "b", "B": "bh", "m": "m", "y": "y", "r": "r", "l": "l", "v": "v", "S": "ś", "z": "ṣ", "s": "s", "h": "h", "L": "ḻ", "kz": "kṣ", "jY": "jñ" },
            "symbols": { "0": "0", "1": "1", "2": "2", "3": "3", "4": "4", "5": "5", "6": "6", "7": "7", "8": "8", "9": "9", "AUM": "oṃ", "'": "'", ".": "." },
            "accents": { "̭": "̭", "\\": "॒", "^": "̀", "/": "́", "²": "²", "³": "³", "⁴": "⁴", "⁵": "⁵", "⁶": "⁶", "⁷": "⁷", "⁸": "⁸", "⁹": "⁹", "꣪": "꣪", "꣫": "꣫", "꣬": "꣬", "꣭": "꣭", "꣮": "꣮", "꣯": "꣯", "꣰": "꣰", "꣱": "꣱" },
            "extra_consonants": { "k0": "q", "K0": "k͟h", "g0": "ġ", "j0": "z", "q0": "r̤", "Q0": "r̤h", "P0": "f", "Y0": "ẏ", "r2": "ṟ", "L0": "l̤" },
            "shortcuts": {}
        },
        "slp1_to_hk": {
            "vowels": { "a": "a", "A": "A", "i": "i", "I": "I", "u": "u", "U": "U", "f": "R", "F": "RR", "x": "lR", "X": "lRR", "è": "è", "e": "e", "E": "ai", "ò": "ò", "o": "o", "O": "au" },
            "yogavaahas": { "M": "M", "H": "H", "~": "~" },
            "virama": { "": "" },
            "consonants": { "k": "k", "K": "kh", "g": "g", "G": "gh", "N": "G", "c": "c", "C": "ch", "j": "j", "J": "jh", "Y": "J", "w": "T", "W": "Th", "q": "D", "Q": "Dh", "R": "N", "t": "t", "T": "th", "d": "d", "D": "dh", "n": "n", "p": "p", "P": "ph", "b": "b", "B": "bh", "m": "m", "y": "y", "r": "r", "l": "l", "v": "v", "S": "z", "z": "S", "s": "s", "h": "h", "L": "L", "kz": "kS", "jY": "jJ" },
            "symbols": { "0": "0", "1": "1", "2": "2", "3": "3", "4": "4", "5": "5", "6": "6", "7": "7", "8": "8", "9": "9", "AUM": "OM", "'": "'", ".": "." },
            "accents": {},
            "extra_consonants": { "k0": "q", "K0": "qh", "g0": "g2", "j0": "z2", "q0": "r3", "Q0": "r3h", "P0": "f", "Y0": "Y", "r2": "r2", "L0": "zh" },
            "shortcuts": {}
        },
        "slp1_to_devanagari": {
            "vowels": { "a": "अ", "A": "आ", "i": "इ", "I": "ई", "u": "उ", "U": "ऊ", "f": "ऋ", "F": "ॠ", "x": "ऌ", "X": "ॡ", "è": "ऎ", "e": "ए", "E": "ऐ", "ò": "ऒ", "o": "ओ", "O": "औ" },
            "yogavaahas": { "M": "ं", "H": "ः", "~": "ँ", "M£": "ꣳ" },
            "virama": { "": "्" },
            "consonants": { "k": "क", "K": "ख", "g": "ग", "G": "घ", "N": "ङ", "c": "च", "C": "छ", "j": "ज", "J": "झ", "Y": "ञ", "w": "ट", "W": "ठ", "q": "ड", "Q": "ढ", "R": "ण", "t": "त", "T": "थ", "d": "द", "D": "ध", "n": "न", "p": "प", "P": "फ", "b": "ब", "B": "भ", "m": "म", "y": "य", "r": "र", "l": "ल", "v": "व", "S": "श", "z": "ष", "s": "स", "h": "ह", "L": "ळ", "kz": "क्ष", "jY": "ज्ञ" },
            "symbols": { "0": "०", "1": "१", "2": "२", "3": "३", "4": "४", "5": "५", "6": "६", "7": "७", "8": "८", "9": "९", "AUM": "ॐ", "'": "ऽ", ".": "।", "..": "॥" },
            "accents": { "̭": "॑", "\\": "॒", "^": "᳡", "/": "꣡", "²": "꣢", "³": "꣣", "⁴": "꣤", "⁵": "꣥", "⁶": "꣦", "⁷": "꣧", "⁸": "꣨", "⁹": "꣩", "꣪": "꣪", "꣫": "꣫", "꣬": "꣬", "꣭": "꣭", "꣮": "꣮", "꣯": "꣯", "꣰": "꣰", "꣱": "꣱" },
            "extra_consonants": { "k0": "क़", "K0": "ख़", "g0": "ग़", "j0": "ज़", "q0": "ड़", "Q0": "ढ़", "P0": "फ़", "Y0": "य़", "r2": "ऱ", "L0": "ऴ" },
            "shortcuts": { "|": "Lh" }
        }
    }
};

const TlSLP1 = "slp1";
const TlIAST = "iast";
const TlHK = "hk";
const TlNagari = "dn";


class Transliterator {
    constructor(options = {}) {
        this.options = options;
        this.fromSlp1 = {};
        this.toSlp1 = {};
        this.keys = {}; // For longest-match search
        this.slp1Vowels = {};
        this.slp1Consonants = {};

        this._init();
    }

    _init() {
        const mappingsData = slp1MappingsJSON;

        const baseMap = mappingsData.mappings["slp1_to_iast"];
        if (!baseMap) {
            throw new Error("base 'slp1_to_iast' mapping not found for character classification");
        }
        for (const k in baseMap.vowels) this.slp1Vowels[k] = true;
        for (const k in baseMap.consonants) this.slp1Consonants[k] = true;
        for (const k in baseMap.extra_consonants) this.slp1Consonants[k] = true;

        for (const schemeName in mappingsData.mappings) {
            const parts = schemeName.split("_to_");
            if (parts.length !== 2) continue;

            let targetSchemeStr = parts[1];
            if (targetSchemeStr === "devanagari") {
                targetSchemeStr = "dn";
            }
            const targetScheme = targetSchemeStr;

            const schemeMap = mappingsData.mappings[schemeName];
            const fromMap = {};
            const toMap = {};

            const groups = [
                schemeMap.vowels, schemeMap.yogavaahas, schemeMap.virama,
                schemeMap.consonants, schemeMap.symbols, schemeMap.accents,
                schemeMap.extra_consonants, schemeMap.shortcuts,
            ];

            for (const group of groups) {
                for (const slp1Char in group) {
                    const targetChar = group[slp1Char];
                    fromMap[slp1Char] = targetChar;
                    if (slp1Char !== "") {
                        toMap[targetChar] = slp1Char;
                    }
                }
            }
            this.fromSlp1[targetScheme] = fromMap;
            this.toSlp1[targetScheme] = toMap;
        }

        for (const scheme in this.toSlp1) {
            const convMap = this.toSlp1[scheme];
            const keyMap = {};
            for (const k in convMap) {
                if (k === "") continue;
                const firstChar = k[0];
                if (!keyMap[firstChar]) {
                    keyMap[firstChar] = [];
                }
                keyMap[firstChar].push(k);
            }
            for (const firstChar in keyMap) {
                keyMap[firstChar].sort((a, b) => b.length - a.length);
            }
            this.keys[scheme] = keyMap;
        }

        const slp1KeyMap = {};
        for (const scheme in this.fromSlp1) {
            const convMap = this.fromSlp1[scheme];
            for (const k in convMap) {
                if (k === "") continue;
                const firstChar = k[0];
                if (!slp1KeyMap[firstChar]) {
                    slp1KeyMap[firstChar] = [];
                }
                if (!slp1KeyMap[firstChar].includes(k)) {
                    slp1KeyMap[firstChar].push(k);
                }
            }
        }
        for (const firstChar in slp1KeyMap) {
            slp1KeyMap[firstChar].sort((a, b) => b.length - a.length);
        }
        this.keys[TlSLP1] = slp1KeyMap;
    }

    _findLongestMatch(source, offset, keyMap) {
        if (offset >= source.length) {
            return "";
        }
        const firstChar = source[offset];
        const sortedKeys = keyMap[firstChar];
        if (!sortedKeys) {
            return "";
        }

        for (const key of sortedKeys) {
            if (source.substring(offset).startsWith(key)) {
                return key;
            }
        }
        return "";
    }

    _doConvert(source, convMap, keyMap) {
        let result = "";
        let i = 0;
        const sourceLen = source.length;
        while (i < sourceLen) {
            const match = this._findLongestMatch(source, i, keyMap);

            if (match !== "") {
                result += convMap[match];
                i += match.length;
            } else {
                if (this.options.FallbackCharacter) {
                    result += this.options.FallbackCharacter;
                } else {
                    result += source[i];
                }
                i += 1;
            }
        }
        return result;
    }

    _doConvertDevanagari(source, convMap, keyMap) {
        let result = "";
        let i = 0;
        const sourceLen = source.length;

        const vowelToMatra = {
            "A": "ा", "i": "ि", "I": "ी", "u": "ु", "U": "ू",
            "f": "ृ", "F": "ॄ", "x": "ॢ", "X": "ॣ", "e": "े",
            "E": "ै", "o": "ो", "O": "ौ",
        };

        while (i < sourceLen) {
            const match = this._findLongestMatch(source, i, keyMap);

            if (match === "") {
                if (this.options.FallbackCharacter) {
                    result += this.options.FallbackCharacter;
                } else {
                    result += source[i];
                }
                i += 1;
                continue;
            }

            const isConsonant = this.slp1Consonants[match];
            const isVowel = this.slp1Vowels[match];

            if (isConsonant) {
                result += convMap[match];

                const nextMatch = this._findLongestMatch(source, i + match.length, keyMap);
                const isNextVowel = this.slp1Vowels[nextMatch];

                if (isNextVowel) {
                    if (vowelToMatra[nextMatch]) {
                        result += vowelToMatra[nextMatch];
                    }
                    i += match.length + nextMatch.length;
                } else {
                    result += convMap[""]; // virama
                    i += match.length;
                }
            } else if (isVowel) {
                result += convMap[match];
                i += match.length;
            } else {
                result += convMap[match];
                i += match.length;
            }
        }
        return result;
    }

    _doConvertFromDevanagari(source, convMap) {
        let result = "";
        const sourceRunes = Array.from(source);

        const matraToVowel = {
            'ा': "A", 'ि': "i", 'ी': "I", 'ु': "u", 'ू': "U",
            'ृ': "f", 'ॄ': "F", 'ॢ': "x", 'ॣ': "X", 'े': "e",
            'ै': "E", 'ो': "o", 'ौ': "O",
        };
        const virama = '्';

        let i = 0;
        while (i < sourceRunes.length) {
            const char = sourceRunes[i];
            const slp1Char = convMap[char];
            const isMapped = slp1Char !== undefined;

            if (isMapped) {
                const isConsonant = this.slp1Consonants[slp1Char];

                if (isConsonant) {
                    if (i + 1 < sourceRunes.length) {
                        const nextChar = sourceRunes[i + 1];
                        if (matraToVowel[nextChar]) {
                            result += slp1Char;
                            result += matraToVowel[nextChar];
                            i += 2;
                            continue;
                        } else if (nextChar === virama) {
                            result += slp1Char;
                            i += 2;
                            continue;
                        }
                    }
                    result += slp1Char;
                    result += "a";
                    i++;
                } else {
                    result += slp1Char;
                    i++;
                }
            } else {
                if (this.options.FallbackCharacter) {
                    result += this.options.FallbackCharacter;
                } else {
                    result += char;
                }
                i++;
            }
        }
        return result;
    }

    convert(source, sourceTl, targetTl) {
        if (sourceTl === targetTl) {
            return source;
        }

        let slp1Text;

        if (sourceTl === TlSLP1) {
            slp1Text = source;
        } else {
            const sourceMap = this.toSlp1[sourceTl];
            if (!sourceMap) {
                throw new Error(`Unsupported source transliteration: ${sourceTl}`);
            }
            if (sourceTl === TlNagari) {
                slp1Text = this._doConvertFromDevanagari(source, sourceMap);
            } else {
                slp1Text = this._doConvert(source, sourceMap, this.keys[sourceTl]);
            }
        }

        if (targetTl === TlSLP1) {
            return slp1Text;
        }

        const targetMap = this.fromSlp1[targetTl];
        if (!targetMap) {
            throw new Error(`Unsupported target transliteration: ${targetTl}`);
        }

        let result;
        if (targetTl === TlNagari) {
            result = this._doConvertDevanagari(slp1Text, targetMap, this.keys[TlSLP1]);
        } else {
            result = this._doConvert(slp1Text, targetMap, this.keys[TlSLP1]);
        }

        return result;
    }
}

if (typeof module !== 'undefined' && module.exports) {
    module.exports = { Transliterator, TlSLP1, TlIAST, TlHK, TlNagari };
}

// Check if running in browser
if (typeof window !== 'undefined') {
    document.addEventListener('DOMContentLoaded', () => {
        const transliterator = new Transliterator({});

        function updateSuggestion(form) {
            const input = form.querySelector('.search-input');
            const tlSelect = form.querySelector('.transliteration-select');
            const suggestionEl = form.nextElementSibling?.querySelector('.transliteration-suggestion');

            if (!input || !tlSelect || !suggestionEl) {
                return;
            }

            const query = input.value;
            const sourceTl = tlSelect.value;

            if (query.trim() === '') {
                suggestionEl.innerHTML = ' ';
                return;
            }
            
            if (sourceTl === TlIAST) {
                suggestionEl.innerHTML = ' ';
                return;
            }

            try {
                const iast = transliterator.convert(query, sourceTl, TlIAST);
                suggestionEl.textContent = `🔎 ${iast}`;
            } catch (e) {
                console.error("Transliteration failed", e);
                suggestionEl.textContent = ' ';
            }
        }

        document.querySelectorAll('.dictionary-search-form, .scripture-search-form').forEach(form => {
            const input = form.querySelector('.search-input');
            const tlSelect = form.querySelector('.transliteration-select');

            if (input && tlSelect) {
                input.addEventListener('input', () => updateSuggestion(form));
                tlSelect.addEventListener('change', () => updateSuggestion(form));
                updateSuggestion(form); // Initial check
            }
        });
    });
}
