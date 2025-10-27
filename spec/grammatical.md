## Specifications for grammatical analysis tables

There should be a foldable table in the end of the excerpt display screen (excerpts.html) which shows the morphological glossings of each word as stored in the `ExcerptWithWords` structure.

Since many sorts of information are sparse and not applicable on every glossing, we want to show them as bootstrap badges instead.

## Tabular format
```
| Surface | Lemma | Kind  | Information
| {{ .Surface }} | {{ .Lemma }} or N/A | {{ .Gramm }} or N/A | Labels as described below 
```

## Mappings of various strings

Fields like gender, number or tense may not be applicable to all words in morphological glossings. Eg (from debug output)

```
scripture.ExcerptGlossing {Surface: "adhithāḥ", Lemma: "√dhā- 1", Gramm: "root", Case: "", Number: "SG", Gender: "", Tense: "AOR", Voice: "MED", Person: "2", Mood: "IND", Root: "", Modifiers: []github.com/mahesh-hegde/dhee/app/scripture.Modifier len:...
```

For this case and gender are not really applicable since its a verb.

For each applicable field, we should display it as a badge in the format

```
number=SG mood=IND tense=AOR
```

Upon hover, tooltip should display the full name of the abbreviated badge eg "Indirect" or "Aorist" if available. A mapping is given and it can be defined and passed to ExcerptTemplateData.


The mapping is given below in HTML format. You can define it as a go dictionary in common/ package.

```
<thead>
<tr>
<th>Code</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td><code>1</code></td>
<td>first person</td>
</tr>
<tr>
<td><code>2</code></td>
<td>second person</td>
</tr>
<tr>
<td><code>3</code></td>
<td>third person</td>
</tr>
<tr>
<td><code>ABL</code></td>
<td>ablative</td>
</tr>
<tr>
<td><code>ACC</code></td>
<td>accusative</td>
</tr>
<tr>
<td><code>ACT</code></td>
<td>active</td>
</tr>
<tr>
<td><code>AOR</code></td>
<td>aorist</td>
</tr>
<tr>
<td><code>COND</code></td>
<td>conditional</td>
</tr>
<tr>
<td><code>CVB</code></td>
<td>converb</td>
</tr>
<tr>
<td><code>DAT</code></td>
<td>dative</td>
</tr>
<tr>
<td><code>DU</code></td>
<td>dual</td>
</tr>
<tr>
<td><code>F</code></td>
<td>feminine</td>
</tr>
<tr>
<td><code>FUT</code></td>
<td>future</td>
</tr>
<tr>
<td><code>GEN</code></td>
<td>genitive</td>
</tr>
<tr>
<td><code>IMP</code></td>
<td>imperative</td>
</tr>
<tr>
<td><code>IND</code></td>
<td>indicative</td>
</tr>
<tr>
<td><code>INF</code></td>
<td>infinitive</td>
</tr>
<tr>
<td><code>INJ</code></td>
<td>injuctive</td>
</tr>
<tr>
<td><code>INS</code></td>
<td>instrumental</td>
</tr>
<tr>
<td><code>IPRF</code></td>
<td>imperfect</td>
</tr>
<tr>
<td><code>LOC</code></td>
<td>locative</td>
</tr>
<tr>
<td><code>M</code></td>
<td>mascuiline</td>
</tr>
<tr>
<td><code>MED</code></td>
<td>middle voice</td>
</tr>
<tr>
<td><code>N</code></td>
<td>neuter</td>
</tr>
<tr>
<td><code>NOM</code></td>
<td>nominative</td>
</tr>
<tr>
<td><code>OPT</code></td>
<td>optative</td>
</tr>
<tr>
<td><code>PASS</code></td>
<td>passive voice</td>
</tr>
<tr>
<td><code>PL</code></td>
<td>plural</td>
</tr>
<tr>
<td><code>PLUPRF</code></td>
<td>past perfect</td>
</tr>
<tr>
<td><code>PPP</code></td>
<td>na participle perfective passive</td>
</tr>
<tr>
<td><code>PPP</code></td>
<td>ta participle perfective passive</td>
</tr>
<tr>
<td><code>PRF</code></td>
<td>perfect</td>
</tr>
<tr>
<td><code>PRS</code></td>
<td>present</td>
</tr>
<tr>
<td><code>PTCP</code></td>
<td>participle</td>
</tr>
<tr>
<td><code>SBJV</code></td>
<td>subjunctive</td>
</tr>
<tr>
<td><code>SG</code></td>
<td>singular</td>
</tr>
<tr>
<td><code>VOC</code></td>
<td>vocative</td>
</tr>
</tbody>
```

If a modifier is not in this list, it should be still displayed as a badge but hover can be omitted.

Finally, if a word has one or more entries in the ScriptureWithWords.Words map, it should display all the Word.Body-s separated by semicolons upon hovering the surface column or lemma column. (For whichever the match has been found). If such matches exist for glossing.Surface or glossing.Lemma, you must highlight those words with a dotted underline inside the `<td>`.
