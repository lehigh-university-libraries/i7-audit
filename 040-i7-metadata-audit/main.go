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
	Names               []Name    `xml:"name"`
	Abstract            Element   `xml:"abstract"`
	AccessCondition     Element   `xml:"accessCondition"`
	Classification      Element   `xml:"classification"`
	Genre               Element   `xml:"genre"`
	Identifier          []Element `xml:"identifier"`
	Language            Element   `xml:"language>languageTerm"`
	PhysicalLocation    Element   `xml:"location>physicalLocation"`
	Note                Element   `xml:"note"`
	DateCaptured        Element   `xml:"originInfo>dateCaptured"`
	DateCreated         Element   `xml:"originInfo>dateCreated"`
	DateIssued          Element   `xml:"originInfo>dateIssued"`
	DateOther           Element   `xml:"originInfo>dateOther"`
	DateValid           Element   `xml:"originInfo>dateValid"`
	Edition             Element   `xml:"originInfo>edition"`
	Extent              Element   `xml:"physicalDescription>extent"`
	Form                Element   `xml:"physicalDescription>form"`
	InternetMediaType   Element   `xml:"physicalDescription>internetMediaType"`
	Issuance            Element   `xml:"originInfo>issuance"`
	Origin              Element   `xml:"physicalDescription>digitalOrigin"`
	Place               Element   `xml:"originInfo>place>placeTerm"`
	PhysicalDescription Element   `xml:"physicalDescription>note"`
	RecordOrigin        Element   `xml:"recordInfo>recordOrigin"`
	ResourceType        Element   `xml:"typeOfResource"`
	Subject             []Element `xml:"subject>topic"`
	SubjectGeographic   []Element `xml:"subject>geographic"`
	SubjectName         []Element `xml:"mods>subject>name>namePart"`
	TableOfContents     Element   `xml:"tableOfContents"`

	// mods/originInfo/publisher -> field_linked_agent:relators:pbl
}

type Element struct {
	Authority string `xml:"authority,attr"`
	Type      string `xml:"type,attr"`
	Value     string `xml:",innerxml"`
}

type Name struct {
	NamePart string `xml:"namePart"`
}

var (
	pids           = map[string]string{}
	fieldsToAccess = map[string]string{
		"field_description":          "Abstract",
		"field_rights":               "AccessConditions",
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
		"title":                      "TitleInfo",
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
	data := []string{"node_id"}
	for field, _ := range fieldsToAccess {
		data = append(data, field)
	}
	err = writer.Write(data)
	if err != nil {
		panic(err)
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
			log.Println("Comparing", path, " against ", url)
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

			row := modsMatch(pid, i7, i2)
			if len(row) > 0 {
				err = writer.Write(row)
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

func modsMatch(pid string, m1, m2 Mods) []string {
	row := []string{
		pids[pid],
	}
	i7 := reflect.ValueOf(m1)
	i2 := reflect.ValueOf(m2)
	mismatch := false
	for drupalField, fieldName := range fieldsToAccess {
		if !reflect.Indirect(i7).FieldByName(fieldName).IsValid() {
			row = append(row, "")
			continue
		}

		i7Element := reflect.Indirect(i7).FieldByName(fieldName).Interface().(Element)
		i2Element := reflect.Indirect(i2).FieldByName(fieldName).Interface().(Element)

		if i7Element.Value == "" && i2Element.Value == "" {
			row = append(row, i2Element.Value)
			continue
		}

		v1 := normalize(i7Element.Value)
		v2 := normalize(i2Element.Value)
		if !areStringsEqualIgnoringSpecialChars(v1, v2) {
			row = append(row, i7Element.Value)
			fmt.Println(drupalField, i7Element.Value, i2Element.Value)
			mismatch = true
			continue
		}
		row = append(row, i2Element.Value)
	}
	/*
		for i, name := range m1.Names {
			if i >= len(m2.Names) || name.NamePart != m2.Names[i].NamePart {
				return false
			}
		}
	*/

	if mismatch {
		return row
	}

	return []string{}
}

func normalize(s string) string {
	s = strings.ReplaceAll(s, "\n", " ")

	// replace all double spaces with a single space
	pattern := regexp.MustCompile(`\s+`)
	s = pattern.ReplaceAllString(s, " ")

	return strings.TrimSpace(s)
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
