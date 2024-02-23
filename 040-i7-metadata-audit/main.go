package main

import (
	"encoding/csv"
	"encoding/xml"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"strings"
	"unicode"
	"unicode/utf8"
)

type Mods struct {
	XMLName             xml.Name  `xml:"mods"`
	TitleInfo           []Element `xml:"titleInfo>title"`
	Names               []Element `xml:"name"`
	Abstract            []Element `xml:"abstract"`
	AccessCondition     []Element `xml:"accessCondition"`
	Classification      []Element `xml:"classification"`
	Genre               []Element `xml:"genre"`
	Identifier          []Element `xml:"identifier"`
	Language            []Element `xml:"language>languageTerm"`
	PhysicalLocation    []Element `xml:"location>physicalLocation"`
	Note                []Element `xml:"note"`
	DateCaptured        []Element `xml:"originInfo>dateCaptured"`
	DateCreated         []Element `xml:"originInfo>dateCreated"`
	DateIssued          []Element `xml:"originInfo>dateIssued"`
	DateOther           []Element `xml:"originInfo>dateOther"`
	DateValid           []Element `xml:"originInfo>dateValid"`
	Edition             []Element `xml:"originInfo>edition"`
	Extent              []Element `xml:"physicalDescription>extent"`
	Form                []Element `xml:"physicalDescription>form"`
	InternetMediaType   []Element `xml:"physicalDescription>internetMediaType"`
	Issuance            []Element `xml:"originInfo>issuance"`
	Origin              []Element `xml:"physicalDescription>digitalOrigin"`
	Place               []Element `xml:"originInfo>place>placeTerm"`
	PhysicalDescription []Element `xml:"physicalDescription>note"`
	RecordOrigin        []Element `xml:"recordInfo>recordOrigin"`
	RelatedItem         []Element `xml:"relatedItem"`
	ResourceType        []Element `xml:"typeOfResource"`
	Subject             []Element `xml:"subject>topic"`
	SubjectGeographic   []Element `xml:"subject>geographic"`
	SubjectName         []Element `xml:"mods>subject>name>namePart"`
	TableOfContents     []Element `xml:"tableOfContents"`
	Publisher           []Element `xml:"originInfo>publisher"`
}

type Element struct {
	Authority  string    `xml:"authority,attr"`
	Type       string    `xml:"type,attr"`
	Value      string    `xml:",innerxml"`
	Identifier string    `xml:"identifier"`
	Number     string    `xml:"part>detail>number"`
	Title      string    `xml:"titleInfo>title"`
	NamePart   string    `xml:"namePart"`
	Role       []Element `xml:"role>roleTerm"`
}

var (
	pids           = map[string]string{}
	header         = []string{}
	fieldsToAccess = map[string]string{
		"field_description":          "Abstract",
		"field_rights":               "AccessCondition",
		"field_classification":       "Classification",
		"field_genre":                "Genre",
		"field_identifier":           "Identifier",
		"field_language":             "Language",
		"field_physical_location":    "PhysicalLocation",
		"field_note":                 "Note",
		"field_date_captured":        "DateCaptured",
		"field_edtf_date_created":    "DateCreated",
		"field_edtf_date_issued":     "DateIssued",
		"field_date_other":           "DateOther",
		"field_date_valid":           "DateValid",
		"field_edition":              "Edition",
		"field_extent":               "Extent",
		"field_physical_form":        "Form",
		"field_media_type":           "InternetMediaType",
		"field_mode_of_issuance":     "Issuance",
		"field_digital_origin":       "Origin",
		"field_place_published":      "Place",
		"field_record_origin":        "RecordOrigin",
		"field_table_of_contents":    "TableOfContents",
		"field_physical_description": "PhysicalDescription",
		"field_resource_type":        "ResourceType",
		"field_subject":              "Subject",
		"field_geographic_subject":   "SubjectGeographic",
		"field_subjects_name":        "SubjectName",
		"field_linked_agent":         "Names",
		"field_related_item":         "RelatedItem",
	}
)

func init() {
	cacheCsv(pids, "pids.csv")
}

func cacheCsv(m map[string]string, f string) {
	file, err := os.Open(f)
	if err != nil {
		fmt.Println("Error opening CSV file:", err)
		return
	}
	defer file.Close()

	reader := csv.NewReader(file)

	// skip header
	reader.Read()

	for {
		record, err := reader.Read()
		if err != nil {
			break
		}
		if len(record) == 2 {
			m[record[1]] = record[0]
		}
	}
}

func main() {
	dir := os.Getenv("DIR")
	if dir == "" {
		fmt.Println("DIR environment variable is not set.")
		return
	}
	dir = filepath.Clean(dir)
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		fmt.Printf("Directory %s does not exist.\n", dir)
		return
	}

	file, err := os.Create("update.csv")
	if err != nil {
		panic(err)
	}
	defer file.Close()
	writer := csv.NewWriter(file)
	header = append(header, "node_id")
	for field, _ := range fieldsToAccess {
		header = append(header, field)
	}
	if err := writer.Write(header); err != nil {
		fmt.Println("Error:", err)
		return
	}

	err = filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if info.Name() == ".keep" {
			return nil
		}

		if err != nil {
			fmt.Printf("Error accessing %s: %v\n", path, err)
			return err
		}
		if !info.IsDir() {
			// read the i7 MODS we downloaded locally
			pid := strings.ReplaceAll(info.Name(), ".xml", "")
			i7Mods, err := os.ReadFile(path)
			if err != nil {
				return fmt.Errorf("Error reading file: %v", err)
			}

			// get the MODS output in i2
			url := fmt.Sprintf("https://islandora.dev/islandora/object/%s?_format=mods", pid)
			resp, err := http.Get(url)
			if err != nil {
				return fmt.Errorf("Error making GET request: %v", err)
			}
			defer resp.Body.Close()
			if resp.StatusCode != http.StatusOK {
				return fmt.Errorf("HTTP request failed with status code: %d", resp.StatusCode)
			}
			i2Mods, err := io.ReadAll(resp.Body)
			if err != nil {
				return fmt.Errorf("Error reading response body: %v", err)
			}

			// compare i7 vs i2
			var i7, i2 Mods
			xml.Unmarshal(i7Mods, &i7)
			xml.Unmarshal(i2Mods, &i2)
			log.Println(pid)
			row := modsMatch(pid, i7, i2)
			if len(row) > 0 {
				var record []string
				for _, key := range header {
					cell := strings.Join(row[key], "|")
					record = append(record, cell)
				}
				err = writer.Write(record)
				if err != nil {
					panic(err)
				}
				writer.Flush()
				if err := writer.Error(); err != nil {
					panic(err)
				}
			}
		}

		return nil
	})

	if err != nil {
		fmt.Printf("Error walking directory: %v\n", err)
		return
	}
}

func modsMatch(pid string, m1, m2 Mods) map[string][]string {
	row := map[string][]string{
		"node_id": []string{pids[pid]},
	}
	i7 := reflect.ValueOf(m1)
	i2 := reflect.ValueOf(m2)
	mismatch := false
	for drupalField, fieldName := range fieldsToAccess {
		row[drupalField] = []string{}
		i7Elements := reflect.Indirect(i7).FieldByName(fieldName).Interface().([]Element)
		i2Elements := reflect.Indirect(i2).FieldByName(fieldName).Interface().([]Element)
		for k, e1 := range i7Elements {
			if len(i2Elements) < k+1 {
				row[drupalField] = append(row[drupalField], e1.Value)
				mismatch = true
				continue
			}

			e2 := i2Elements[k]
			if e1.Value == "" && e2.Value == "" {
				row[drupalField] = append(row[drupalField], "")
				continue
			}

			v1 := normalize(e1.Value)
			v2 := normalize(e2.Value)
			if !areStringsEqualIgnoringSpecialChars(v1, v2) {
				fmt.Println(drupalField, v1, v2)
				mismatch = true
			}
			row[drupalField] = append(row[drupalField], e1.Value)
		}
	}
	if mismatch {
		return row
	}

	return map[string][]string{}
}

func normalize(s string) string {
	s = strings.ReplaceAll(s, "\n", " ")

	// replace all double spaces with a single space
	pattern := regexp.MustCompile(`\s+`)
	s = pattern.ReplaceAllString(s, " ")

	s = strings.ToLower(s)
	s = strings.TrimSpace(s)
	if isDateString(s) {
		s = removeTimeFromDate(s)
	}

	return s
}

func isDateString(str string) bool {
	dateRegex := `\d{4}-\d{2}-\d{2}t\d{2}:\d{2}:\d{2}Z`
	match, _ := regexp.MatchString(dateRegex, str)
	return match
}

func removeTimeFromDate(str string) string {
	parts := strings.Split(str, "t")
	return parts[0]
}

func isAlphanumeric(r rune) bool {
	if unicode.IsLetter(r) || unicode.IsDigit(r) {
		return true
	}

	if (r >= '\u0030' && r <= '\u1FFF') || unicode.In(r, unicode.Mark, unicode.Sk, unicode.Lm) {
		return true
	}

	return false
}

func areStringsEqualIgnoringSpecialChars(s1, s2 string) bool {
	// Compare the strings while ignoring characters that are not alphanumeric.
	i, j := 0, 0
	for i < len(s1) && j < len(s2) {
		r1, size1 := utf8.DecodeRuneInString(s1[i:])
		r2, size2 := utf8.DecodeRuneInString(s2[j:])
		if isAlphanumeric(r1) && isAlphanumeric(r2) {
			if r1 != r2 {
				return false
			}
		}
		i += size1
		j += size2
	}

	// Check if any remaining characters are alphanumeric.
	for i < len(s1) {
		if isAlphanumeric(rune(s1[i])) {
			return false
		}
		i++
	}
	for j < len(s2) {
		if isAlphanumeric(rune(s2[j])) {
			return false
		}
		j++
	}

	return true
}

func (m *Mods) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	type modAlias Mods

	for {
		token, err := d.Token()
		if err != nil {
			return err
		}
		switch t := token.(type) {
		case xml.StartElement:
			switch t.Name.Local {
			case "mods":
				var alias modAlias
				if err := d.DecodeElement(&alias, &t); err != nil {
					return err
				}
				*m = Mods(alias)
			case "relatedItem":
				var e Element
				if err := d.DecodeElement(&e, &t); err != nil {
					return err
				}
				e.Value = fmt.Sprintf("title:%s:number:%s:identifier:%s", e.Title, e.Number, e.Identifier)
				m.RelatedItem = append(m.RelatedItem, e)

			case "name":
				var e Element
				if err := d.DecodeElement(&e, &t); err != nil {
					return err
				}
				relator := "cre"
				for _, r := range e.Role {
					if r.Type == "code" {
						relator = r.Value
						break
					}
				}
				e.Value = fmt.Sprintf("relators:%s:person:%s", relator, e.NamePart)
				m.Names = append(m.Names, e)
			case "publisher":
				var e Element
				if err := d.DecodeElement(&e, &t); err != nil {
					return err
				}
				e.Value = fmt.Sprintf("relators:pbl:corporate_body:%s", e.Value)
				m.Names = append(m.Names, e)
			case "geographic":
				var e Element
				if err := d.DecodeElement(&e, &t); err != nil {
					return err
				}
				vid := "geo_location"
				if e.Authority == "naf" {
					vid = "geographic_naf"
				} else if e.Authority == "local" {
					vid = "geographic_local"
				}
				e.Value = fmt.Sprintf("%s:%s", vid, e.Value)
				m.SubjectGeographic = append(m.SubjectGeographic, e)
			case "abstract", "dateOther", "identifier", "note":
				var e Element
				if err := d.DecodeElement(&e, &t); err != nil {
					return err
				}
				if e.Type != "" {
					e.Value = fmt.Sprintf("attr0:%s:%s", e.Type, e.Value)
				}
				switch t.Name.Local {
				case "abstract":
					m.Abstract = append(m.Abstract, e)
				case "dateOther":
					m.DateOther = append(m.DateOther, e)
				case "identifier":
					m.Identifier = append(m.Identifier, e)
				case "note":
					m.Note = append(m.Note, e)
				}
			default:
				if err := d.DecodeElement(&m, &t); err != nil {
					return err
				}
			}
		case xml.EndElement:
			if t == start.End() {
				return nil
			}
		}
	}
}
