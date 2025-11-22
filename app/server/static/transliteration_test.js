const { Transliterator, TlSLP1, TlIAST, TlHK, TlNagari, foldAccents } = require('./common.js');

let failed = 0;
let passed = 0;

function assertEqual(expected, actual, message) {
    if (expected !== actual) {
        console.error(`FAIL: ${message}`);
        console.error(`  Expected: ${expected}`);
        console.error(`  Actual:   ${actual}`);
        failed++;
    } else {
        passed++;
    }
}

function assertNoError(fn, message) {
    try {
        fn();
        passed++;
    } catch (e) {
        console.error(`FAIL: ${message}`);
        console.error(`  Unexpected error: ${e.message}`);
        failed++;
    }
}

function assertError(fn, message) {
    try {
        fn();
        console.error(`FAIL: ${message}`);
        console.error(`  Expected an error, but none was thrown.`);
        failed++;
    } catch (e) {
        passed++;
    }
}

function runTest(name, testFn) {
    testFn();
}

function testTransliteratorConvert() {
    const transliterator = new Transliterator({});

    const testCases = [
        { name: "SLP1 to IAST", source: "saMskfta", sourceTl: TlSLP1, targetTl: TlIAST, expected: "saṃskṛta", expectErr: false },
        { name: "SLP1 to IAST with hyphen", source: "saMskfta-BAzA", sourceTl: TlSLP1, targetTl: TlIAST, expected: "saṃskṛta-bhāṣā", expectErr: false },
        { name: "SLP1 to HK", source: "saMskfta", sourceTl: TlSLP1, targetTl: TlHK, expected: "saMskRta", expectErr: false },
        { name: "SLP1 to Devanagari", source: "saMskftam", sourceTl: TlSLP1, targetTl: TlNagari, expected: "संस्कृतम्", expectErr: false },
        { name: "SLP1 to Devanagari Complex", source: "rAmaH kfzRaSca", sourceTl: TlSLP1, targetTl: TlNagari, expected: "रामः कृष्णश्च", expectErr: false },
        { name: "SLP1 to Devanagari Vowel Start", source: "indra", sourceTl: TlSLP1, targetTl: TlNagari, expected: "इन्द्र", expectErr: false },
        { name: "IAST to SLP1", source: "saṃskṛta", sourceTl: TlIAST, targetTl: TlSLP1, expected: "saMskfta", expectErr: false },
        { name: "HK to SLP1", source: "saMskRta", sourceTl: TlHK, targetTl: TlSLP1, expected: "saMskfta", expectErr: false },
        { name: "Devanagari to SLP1", source: "संस्कृतम्", sourceTl: TlNagari, targetTl: TlSLP1, expected: "saMskftam", expectErr: false },
        { name: "Devanagari to SLP1 Complex", source: "रामः कृष्णश्च", sourceTl: TlNagari, targetTl: TlSLP1, expected: "rAmaH kfzRaSca", expectErr: false },
        { name: "Devanagari to SLP1 Vowel Start", source: "इन्द्र", sourceTl: TlNagari, targetTl: TlSLP1, expected: "indra", expectErr: false },
        { name: "IAST to Devanagari", source: "saṃskṛtam", sourceTl: TlIAST, targetTl: TlNagari, expected: "संस्कृतम्", expectErr: false },
        { name: "Devanagari to HK", source: "संस्कृतम्", sourceTl: TlNagari, targetTl: TlHK, expected: "saMskRtam", expectErr: false },
        { name: "HK to IAST", source: "saMskRta", sourceTl: TlHK, targetTl: TlIAST, expected: "saṃskṛta", expectErr: false },
        { name: "IAST to IAST", source: "saṃskṛta", sourceTl: TlIAST, targetTl: TlIAST, expected: "saṃskṛta", expectErr: false },
        { name: "SLP1 to SLP1", source: "saMskfta", sourceTl: TlSLP1, targetTl: TlSLP1, expected: "saMskfta", expectErr: false },
        { name: "Empty String", source: "", sourceTl: TlSLP1, targetTl: TlIAST, expected: "", expectErr: false },
        { name: "Unsupported Source", source: "test", sourceTl: "unsupported", targetTl: TlIAST, expected: "", expectErr: true },
        { name: "Unsupported Target", source: "test", sourceTl: TlIAST, targetTl: "unsupported", expected: "", expectErr: true },
        { name: "Unmapped characters", source: "abc_123", sourceTl: TlSLP1, targetTl: TlIAST, expected: "abc_123", expectErr: false },
    ];

    testCases.forEach(tc => {
        runTest(tc.name, () => {
            if (tc.expectErr) {
                assertError(() => transliterator.convert(tc.source, tc.sourceTl, tc.targetTl), tc.name);
            } else {
                let actual;
                assertNoError(() => {
                    actual = transliterator.convert(tc.source, tc.sourceTl, tc.targetTl);
                }, tc.name);
                assertEqual(tc.expected, actual, tc.name);
            }
        });
    });
}

function testTransliteratorConvertWithFallback() {
    const transliterator = new Transliterator({ FallbackCharacter: "?" });

    const testCases = [
        { name: "Unmapped with fallback", source: "abc_123", sourceTl: TlSLP1, targetTl: TlIAST, expected: "abc?123" },
        { name: "Unmapped Devanagari with fallback", source: "अ_ब", sourceTl: TlNagari, targetTl: TlSLP1, expected: "a?ba" },
    ];

    testCases.forEach(tc => {
        runTest(tc.name, () => {
            let actual;
            assertNoError(() => {
                actual = transliterator.convert(tc.source, tc.sourceTl, tc.targetTl);
            }, tc.name);
            assertEqual(tc.expected, actual, tc.name);
        });
    });
}

function testFoldAccents() {
    const testCases = [
        { name: "Fold single accent", input: "devásya", expected: "devasya" },
        { name: "Fold multiple accents", input: "br̥hád vadema vidáthe suvī́rāḥ", expected: "bṛhad vadema vidathe suvīrāḥ" },
        { name: "No accents", input: "devasya", expected: "devasya" },
        { name: "Empty string", input: "", expected: "" },
        { name: "Fold m with dot", input: "saṁskṛtam", expected: "saṃskṛtam" },
    ];

    testCases.forEach(tc => {
        runTest(tc.name, () => {
            const actual = foldAccents(tc.input);
            assertEqual(tc.expected, actual, tc.name);
        });
    });
}

function testTransliteratorConvertNormalized() {
    const transliterator = new Transliterator({});
    const testCases = [
        { name: "Fold IAST to SLP1", source: "devásya", sourceTl: TlIAST, targetTl: TlSLP1, expected: "devasya" },
        { name: "No fold for SLP1 to IAST", source: "devasya", sourceTl: TlSLP1, targetTl: TlIAST, expected: "devasya" },
        { name: "Fold IAST to Devanagari", source: "agním", sourceTl: TlIAST, targetTl: TlNagari, expected: "अग्निम्" },
    ];

    testCases.forEach(tc => {
        runTest(tc.name, () => {
            let actual;
            assertNoError(() => {
                actual = transliterator.convertNormalized(tc.source, tc.sourceTl, tc.targetTl);
            }, tc.name);
            assertEqual(tc.expected, actual, tc.name);
        });
    });
}

console.log("Running transliteration tests...");
testTransliteratorConvert();
testTransliteratorConvertWithFallback();
console.log("\nRunning accent folding tests...");
testFoldAccents();
testTransliteratorConvertNormalized();

console.log(`\nTests finished. Passed: ${passed}, Failed: ${failed}`);
if (failed > 0) {
    process.exit(1);
}
