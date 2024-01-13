package main

import (
	"encoding/csv"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
)

var (
	redirectCache          = make(map[string]string)
	mergedOrDroppedColumns = []string{
		// field_member_of
		"RELS_EXT_isConstituentOf_uri_ms",
		"RELS_EXT_isMemberOfCollection_uri_ms",
		"RELS_EXT_isMemberOf_uri_ms",
		"RELS_EXT_isPageOf_uri_ms",
		// field_linked_agent
		"dc.creator",
		"dc.contributor",
		"mods_name_photographer_namePart_ms",
		"mods_name_corporate_department_namePart_ms",
		"mods_name_thesis_advisor_namePart_ms",
		// title
		"mods_titleInfo_title_all_ms",
		"mods_titleInfo_title_ms",
		"dc.title",
		// field_description
		"mods_abstract_mt",
		"dc.description",
		// ignored
		"ID",
		"file",
	}
)

func main() {
	inputFilePath := "input.csv"
	outputFilePath := "output.csv"

	inputFile, err := os.Open(inputFilePath)
	if err != nil {
		fmt.Println("Error opening input file:", err)
		return
	}
	defer inputFile.Close()

	outputFile, err := os.Create(outputFilePath)
	if err != nil {
		fmt.Println("Error creating output file:", err)
		return
	}
	defer outputFile.Close()

	csvReader := csv.NewReader(inputFile)
	csvWriter := csv.NewWriter(outputFile)

	columnNames, err := csvReader.Read()
	if err != nil {
		fmt.Println("Error reading CSV header:", err)
		return
	}

	// Remove the columns to be transformed and add "field_member_of" to the header
	updatedHeader := []string{}
	for _, columnName := range columnNames {
		if strInSlice(columnName, mergedOrDroppedColumns) {
			continue
		}

		switch columnName {
		case "PID":
			updatedHeader = append(updatedHeader, "field_pid")
		case "dc.title":
			updatedHeader = append(updatedHeader, "title")
		case "RELS_EXT_hasModel_uri_s":
			updatedHeader = append(updatedHeader, "field_model")
		case "sequence":
			updatedHeader = append(updatedHeader, "field_weight")
		case "mods_name_1_nameIdentifier_orcid_ms":
			updatedHeader = append(updatedHeader, "field_orcid_num")
		case "mods_subject_name_personal_namePart_ms":
			updatedHeader = append(updatedHeader, "field_subjects_name")
		case "mods_name_creator_affiliation_institution_mt":
			updatedHeader = append(updatedHeader, "field_affiliated_institution")
		case "mods_name_addressee_namePart_ms":
			updatedHeader = append(updatedHeader, "")
		case "mods_name_creator_description_ms":
			updatedHeader = append(updatedHeader, "")
		case "mods_name_creator_affiliation_email_ss":
			updatedHeader = append(updatedHeader, "field_creator_email")
		case "mods_name_creator_namePart_ms":
			updatedHeader = append(updatedHeader, "")
		case "mods_name_personal_author_namePart_ms":
			updatedHeader = append(updatedHeader, "")
		default:
			updatedHeader = append(updatedHeader, columnName)
		}
	}

	// the order of this slice matters.
	// see the calls to merge*() in transformColumns()
	newColumns := []string{
		"field_member_of",
		"title",
		"field_description",
	}
	for _, newColumn := range newColumns {
		updatedHeader = append(updatedHeader, newColumn)
		columnNames = append(columnNames, newColumn)
	}

	// Write the updated header to the output CSV
	if err := csvWriter.Write(updatedHeader); err != nil {
		fmt.Println("Error writing CSV header:", err)
		return
	}

	columnIndices := make(map[string]int)
	for i, name := range columnNames {
		columnIndices[name] = i
	}

	for {
		record, err := csvReader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			fmt.Println("Error reading CSV:", err)
			return
		}

		transformedRecord := transformColumns(record, columnIndices)

		if err := csvWriter.Write(transformedRecord); err != nil {
			fmt.Println("Error writing CSV:", err)
			return
		}
	}

	csvWriter.Flush()

	if err := csvWriter.Error(); err != nil {
		fmt.Println("Error writing CSV:", err)
		return
	}

	fmt.Println("CSV transformation complete. Output written to", outputFilePath)
}

func transformColumns(record []string, columnIndices map[string]int) []string {
	transformModel(record, columnIndices)

	// the order in which we call these matters since we're appending the CSV header
	// along with appending the new value in the CSV
	// TODO: we should consider refactoring to coordinate this instead
	newRecord := mergeMemberOf(record, columnIndices)
	newRecord = mergeTitle(newRecord, columnIndices)
	newRecord = mergeDescription(newRecord, columnIndices)

	// remove the columns we've merged into a single new column
	hiddenIndices := []int{}
	for _, column := range mergedOrDroppedColumns {
		index := columnIndices[column]
		hiddenIndices = append(hiddenIndices, index)
	}
	transformedRecord := []string{}
	for k, cell := range newRecord {
		if intInSlice(k, hiddenIndices) {
			continue
		}

		// remove solr's escaped commas
		cell = strings.ReplaceAll(cell, "\\,", ",")
		transformedRecord = append(transformedRecord, cell)
	}

	return transformedRecord
}

func transformModel(record []string, columnIndices map[string]int) {
	column := "RELS_EXT_hasModel_uri_s"
	index := columnIndices[column]
	record[index] = getModel(record[index])
}

func mergeTitle(record []string, columnIndices map[string]int) []string {
	index1, _ := columnIndices["mods_titleInfo_title_all_ms"]
	index2, _ := columnIndices["mods_titleInfo_title_ms"]
	index3, _ := columnIndices["dc.title"]
	if record[index1] != "" {
		record = append(record, record[index1])
	} else if record[index2] != "" {
		record = append(record, record[index2])
	} else if record[index3] != "" {
		record = append(record, record[index3])
	}

	return record
}

func mergeDescription(record []string, columnIndices map[string]int) []string {
	index1, _ := columnIndices["mods_abstract_mt"]
	index2, _ := columnIndices["dc.description"]
	if record[index1] != "" {
		record = append(record, record[index1])
	} else if record[index2] != "" {
		record = append(record, record[index2])
	}

	return record
}

func mergeMemberOf(record []string, columnIndices map[string]int) []string {
	// merge the various field_member_of columns into a single column
	index1, _ := columnIndices["RELS_EXT_isMemberOfCollection_uri_ms"]
	index2, _ := columnIndices["RELS_EXT_isMemberOf_uri_ms"]
	index3, _ := columnIndices["RELS_EXT_isPageOf_uri_ms"]
	index4, _ := columnIndices["RELS_EXT_isConstituentOf_uri_ms"]
	if record[index1] != "" {
		record = append(record, record[index1])
	} else if record[index2] != "" {
		record = append(record, record[index2])
	} else if record[index3] != "" {
		record = append(record, record[index3])
	} else if record[index4] != "" {
		record = append(record, record[index4])
	}

	memberOfStringToEntityId(record, columnIndices, "field_member_of")

	return record
}

func memberOfStringToEntityId(record []string, columnIndices map[string]int, columnName string) {
	index, found := columnIndices[columnName]
	if !found {
		return
	}

	cell := record[index]
	cell = strings.ReplaceAll(cell, "info:fedora/", "https://islandora-stage.lib.lehigh.edu/islandora/object/")

	if number, err := pid2nid(cell); err == nil {
		record[index] = number
	}
}

func getModel(model string) string {
	switch model {
	case "info:fedora/islandora:binaryObjectCModel":
		return "Binary"
	case "info:fedora/islandora:bookCModel":
		return "Paged Content"
	case "info:fedora/islandora:collectionCModel":
		return "Sub-Collection"
	case "info:fedora/islandora:pageCModel":
		return "Page"
	case "info:fedora/islandora:sp_basic_image":
		return "Image"
	case "info:fedora/islandora:sp_document":
		return "Binary"
	case "info:fedora/islandora:sp_large_image_cmodel":
		return "Image"
	case "info:fedora/islandora:sp_pdf":
		return "Digital Document"
	case "info:fedora/islandora:sp_videoCModel":
		return "Video"
	case "info:fedora/islandora:sp_web_archive":
		return "Binary"
	}

	log.Fatalf("Missing model: %s", model)

	return ""
}

func pid2nid(url string) (string, error) {
	if url == "" {
		return "", nil
	}
	if cachedNumber, found := redirectCache[url]; found {
		return cachedNumber, nil
	}
	log.Println(url)
	resp, err := http.Get(url)
	if err != nil {
		fmt.Println("Error fetching URL:", err)
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		log.Fatalf("Unable to find node ID for parent %s", url)
	}
	links := resp.Header.Values("Link")
	for _, link := range links {
		if strings.Contains(link, "?_format=json>") {
			index := strings.Index(link, "?_format=json")
			parts := strings.Split(link[:index], "/")
			if len(parts) >= 2 {
				redirectCache[url] = parts[len(parts)-1]
				return redirectCache[url], nil
			}
		}
	}
	log.Fatalf("Unable to find node ID for parent %s: %v", url, links)

	return url, nil
}

func intInSlice(e int, s []int) bool {
	for _, a := range s {
		if a == e {
			return true
		}
	}
	return false
}

func strInSlice(e string, s []string) bool {
	for _, a := range s {
		if a == e {
			return true
		}
	}
	return false
}
