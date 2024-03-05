package main

import (
	"encoding/csv"
	"encoding/json"
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
	XMLName                       xml.Name  `xml:"mods"`
	TitleInfo                     []Element `xml:"titleInfo>title"`
	Names                         []Element `xml:"name"`
	Abstract                      []Element `xml:"abstract"`
	AccessCondition               []Element `xml:"accessCondition"`
	Classification                []Element `xml:"classification"`
	Genre                         []Element `xml:"genre"`
	Identifier                    []Element `xml:"identifier"`
	Language                      []Element `xml:"language>languageTerm"`
	PhysicalLocation              []Element `xml:"location>physicalLocation"`
	Note                          []Element `xml:"note"`
	DateCaptured                  []Element `xml:"originInfo>dateCaptured"`
	DateCreated                   []Element `xml:"originInfo>dateCreated"`
	DateIssued                    []Element `xml:"originInfo>dateIssued"`
	DateOther                     []Element `xml:"originInfo>dateOther"`
	DateValid                     []Element `xml:"originInfo>dateValid"`
	Edition                       []Element `xml:"originInfo>edition"`
	Issuance                      []Element `xml:"originInfo>issuance"`
	Place                         []Element `xml:"originInfo>place>placeTerm"`
	Publisher                     []Element `xml:"originInfo>publisher"`
	Extent                        []Element `xml:"physicalDescription>extent"`
	Form                          []Element `xml:"physicalDescription>form"`
	InternetMediaType             []Element `xml:"physicalDescription>internetMediaType"`
	Origin                        []Element `xml:"physicalDescription>digitalOrigin"`
	PhysicalDescription           []Element `xml:"physicalDescription>note"`
	RecordOrigin                  []Element `xml:"recordInfo>recordOrigin"`
	RelatedItem                   []Element `xml:"relatedItem"`
	ResourceType                  []Element `xml:"typeOfResource"`
	Subject                       []Element `xml:"subject"`
	TableOfContents               []Element `xml:"tableOfContents"`
	SubjectGeographic             []Element
	SubjectGeographicHierarchical []Element
	SubjectName                   []Element
	SubjectLcsh                   []Element
}

type Element struct {
	Authority              string                 `xml:"authority,attr"`
	Type                   string                 `xml:"type,attr"`
	Point                  string                 `xml:"point,attr"`
	Unit                   string                 `xml:"unit,attr"`
	Value                  string                 `xml:",innerxml"`
	Identifier             string                 `xml:"identifier"`
	Number                 string                 `xml:"part>detail>number"`
	Title                  string                 `xml:"titleInfo>title"`
	NamePart               string                 `xml:"namePart"`
	Role                   []Element              `xml:"role>roleTerm"`
	Geographic             SubElement             `xml:"geographic"`
	SubjectName            string                 `xml:"name>namePart"`
	Topic                  string                 `xml:"topic"`
	HierarchicalGeographic HierarchicalGeographic `xml:"hierarchicalGeographic"`
	Note                   string                 `xml:"note"`
	Language               string                 `xml:"languageTerm"`
	DateCaptured           string                 `xml:"dateCaptured"`
	DateCreated            string                 `xml:"dateCreated"`
	DateIssued             string                 `xml:"dateIssued"`
	DateOther              string                 `xml:"dateOther"`
	DateValid              string                 `xml:"dateValid"`
	Edition                string                 `xml:"edition"`
	Issuance               string                 `xml:"issuance"`
	Place                  string                 `xml:"place>placeTerm"`
	Publisher              string                 `xml:"publisher"`
	Extent                 string                 `xml:"extent"`
	Form                   string                 `xml:"form"`
	InternetMediaType      string                 `xml:"internetMediaType"`
	Origin                 string                 `xml:"digitalOrigin"`
	RecordOrigin           string                 `xml:"recordOrigin"`
	PhysicalLocation       string                 `xml:"physicalLocation"`
}

type SubElement struct {
	Authority string `xml:"authority,attr"`
	Type      string `xml:"type,attr"`
	Value     string `xml:",innerxml"`
}

type HierarchicalGeographic struct {
	City      string `xml:"city" json:"city,omitempty"`
	Continent string `xml:"continent" json:"continent,omitempty"`
	Country   string `xml:"country" json:"country,omitempty"`
	County    string `xml:"county" json:"county,omitempty"`
	State     string `xml:"state" json:"state,omitempty"`
	Territory string `xml:"territory" json:"territory,omitempty"`
}

type RelatedItem struct {
	Identifier string `json:"identifier,omitempty"`
	Title      string `json:"title,omitempty"`
	Number     string `json:"number,omitempty"`
}

type TypedText struct {
	Attr0 string `json:"attr0,omitempty"`
	Attr1 string `json:"attr1,omitempty"`
	Value string `json:"value"`
}

var (
	pids = map[string]string{}

	header         = []string{}
	fieldsToAccess = map[string]string{
		"field_abstract":                 "Abstract",
		"field_rights":                   "AccessCondition",
		"field_classification":           "Classification",
		"field_genre":                    "Genre",
		"field_identifier":               "Identifier",
		"field_language":                 "Language",
		"field_physical_location":        "PhysicalLocation",
		"field_note":                     "Note",
		"field_date_captured":            "DateCaptured",
		"field_edtf_date_created":        "DateCreated",
		"field_edtf_date_issued":         "DateIssued",
		"field_date_valid":               "DateValid",
		"field_edition":                  "Edition",
		"field_extent":                   "Extent",
		"field_physical_form":            "Form",
		"field_media_type":               "InternetMediaType",
		"field_mode_of_issuance":         "Issuance",
		"field_digital_origin":           "Origin",
		"field_place_published":          "Place",
		"field_record_origin":            "RecordOrigin",
		"field_table_of_contents":        "TableOfContents",
		"field_physical_description":     "PhysicalDescription",
		"field_resource_type":            "ResourceType",
		"field_subject":                  "Subject",
		"field_geographic_subject":       "SubjectGeographic",
		"field_subjects_name":            "SubjectName",
		"field_linked_agent":             "Names",
		"field_related_item":             "RelatedItem",
		"field_lcsh_topic":               "SubjectLcsh",
		"field_subject_hierarchical_geo": "SubjectGeographicHierarchical",
		"field_alt_title":                "",
		"field_title_part_name":          "",
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
		} else if len(record) == 1 {
			m[record[0]] = "ok"
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
				log.Printf("HTTP request failed with status code: %d for %s", resp.StatusCode, pid)
				return nil
			}
			i2Mods, err := io.ReadAll(resp.Body)
			if err != nil {
				return fmt.Errorf("Error reading response body: %v", err)
			}

			// compare i7 vs i2
			var i7, i2 Mods
			xml.Unmarshal(i7Mods, &i7)
			xml.Unmarshal(i2Mods, &i2)
			log.Println(pid, pids[pid])

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
				fmt.Println(pid, "\t", pids[pid], "\t", drupalField, "\t", e1, "\t", k, "\tMismatch")
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
				fmt.Println(pid, "\t", pids[pid], "\t", drupalField, "\t", v1, "\t", v2)
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

				ri := RelatedItem{
					Title:      e.Title,
					Identifier: e.Identifier,
					Number:     e.Number,
				}
				jsonData, err := json.Marshal(ri)
				if err != nil {
					fmt.Println("Error marshaling JSON:", err)
					return err
				}

				e.Value = string(jsonData)
				m.RelatedItem = append(m.RelatedItem, e)

			case "name":
				var e Element
				if err := d.DecodeElement(&e, &t); err != nil {
					return err
				}
				if e.NamePart == "" {
					return nil
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
			case "subject":
				var e Element
				if err := d.DecodeElement(&e, &t); err != nil {
					return err
				}
				if e.Topic != "" {
					e.Value = e.Topic
					if e.Authority == "lcsh" {
						m.SubjectLcsh = append(m.SubjectLcsh, e)
					} else if e.Authority != "" {
						log.Println(e.Authority)
					} else {
						m.Subject = append(m.Subject, e)
					}
				} else if e.Geographic.Value != "" {
					vid := "geo_location"
					if e.Geographic.Authority == "naf" {
						vid = "geographic_naf"
					} else if e.Geographic.Authority == "local" {
						vid = "geographic_local"
					}
					e.Value = fmt.Sprintf("%s:%s", vid, e.Geographic.Value)
					m.SubjectGeographic = append(m.SubjectGeographic, e)
				} else if e.SubjectName != "" {
					e.Value = e.SubjectName
					m.SubjectName = append(m.SubjectName, e)
				} else if !e.HierarchicalGeographic.Empty() {
					e.Value, err = e.HierarchicalGeographic.Json()
					if err != nil {
						log.Println("Failed to unmarshal hierarchicalGeographic")
						return fmt.Errorf("Failed to marshal hierarchical geographic as JSON")
					}
					m.SubjectGeographicHierarchical = append(m.SubjectGeographicHierarchical, e)
				} else {
					log.Println(e)
					return fmt.Errorf("Didn't catch this subject")
				}
			case "abstract", "identifier", "note":
				var e Element
				if err := d.DecodeElement(&e, &t); err != nil {
					return err
				}

				tt := TypedText{
					Attr0: e.Type,
					Attr1: e.Point,
					Value: e.Value,
				}
				jsonData, err := json.Marshal(tt)
				if err != nil {
					fmt.Println("Error marshaling JSON:", err)
					return err
				}

				e.Value = string(jsonData)
				switch t.Name.Local {
				case "abstract":
					m.Abstract = append(m.Abstract, e)
				case "identifier":
					m.Identifier = append(m.Identifier, e)
				case "note":
					m.Note = append(m.Note, e)
				}
			default:
				e := Element{}
				if err := d.DecodeElement(&e, &t); err != nil {
					return err
				}
				switch t.Name.Local {
				case "accessCondition":
					m.AccessCondition = append(m.AccessCondition, e)
				case "classification":
					m.Classification = append(m.Classification, e)
				case "genre":
					m.Genre = append(m.Genre, e)
				case "language":
					if e.Language != "" {
						e.Value = e.Language
						m.Language = append(m.Language, e)
					}
				case "location":
					if e.PhysicalLocation != "" {
						e.Value = e.PhysicalLocation
						m.PhysicalLocation = append(m.PhysicalLocation, e)
					}
				case "originInfo":
					if e.DateCaptured != "" {
						e.Value = e.DateCaptured
						m.DateCaptured = append(m.DateCaptured, e)
					}
					if e.DateCreated != "" {
						e.Value = e.DateCreated
						m.DateCreated = append(m.DateCreated, e)
					}
					if e.DateIssued != "" {
						e.Value = e.DateIssued
						m.DateIssued = append(m.DateIssued, e)
					}
					if e.DateOther != "" {
						e.Value = e.DateOther
						m.DateOther = append(m.DateOther, e)
					}
					if e.DateValid != "" {
						e.Value = e.DateValid
						m.DateValid = append(m.DateValid, e)
					}
					if e.Place != "" {
						e.Value = e.Place
						m.Place = append(m.Place, e)
					}
					if e.Publisher != "" {
						e.Value = fmt.Sprintf("relators:pbl:corporate_body:%s", e.Publisher)
						m.Names = append(m.Names, e)
					}
					if e.Edition != "" {
						e.Value = e.Edition
						m.Edition = append(m.Edition, e)
					}
					if e.Issuance != "" {
						e.Value = e.Issuance
						m.Issuance = append(m.Issuance, e)
					}

				case "physicalDescription":
					if e.Extent != "" {
						tt := TypedText{
							Value: e.Extent,
							Attr0: e.Unit,
						}
						jsonData, err := json.Marshal(tt)
						if err != nil {
							fmt.Println("Error marshaling JSON:", err)
							return err
						}
						e.Value = string(jsonData)
						m.Extent = append(m.Extent, e)
					}
					if e.Form != "" {
						e.Value = e.Form
						m.Form = append(m.Form, e)
					}
					if e.InternetMediaType != "" {
						e.Value = e.InternetMediaType
						m.InternetMediaType = append(m.InternetMediaType, e)
					}
					if e.Origin != "" {
						e.Value = e.Origin
						m.Origin = append(m.Origin, e)
					}
					if e.Note != "" {
						ttNote := TypedText{
							Value: e.Note,
							Attr0: e.Type,
						}
						jsonDataNote, err := json.Marshal(ttNote)
						if err != nil {
							fmt.Println("Error marshaling JSON:", err)
							return err
						}
						e.Value = string(jsonDataNote)
						m.PhysicalDescription = append(m.PhysicalDescription, e)
					}
				case "recordInfo":
					if e.RecordOrigin != "" {
						e.Value = e.RecordOrigin
						m.RecordOrigin = append(m.RecordOrigin, e)
					}
				case "typeOfResource":
					m.ResourceType = append(m.ResourceType, e)
				case "tableOfContents":
					m.TableOfContents = append(m.TableOfContents, e)
				}

			}
		case xml.EndElement:
			if t == start.End() {
				return nil
			}
		}
	}
}

func (hg *HierarchicalGeographic) Empty() bool {
	return hg.City == "" && hg.Continent == "" && hg.Country == "" && hg.County == "" && hg.State == "" && hg.Territory == ""
}

func (hg *HierarchicalGeographic) Json() (string, error) {
	jsonData, err := json.Marshal(hg)
	if err != nil {
		fmt.Println("Error marshaling JSON:", err)
		return "", err
	}

	return string(jsonData), nil
}

func strInMap(e string, s []string) bool {
	for _, a := range s {
		if a == e {
			return true
		}
	}
	return false
}
