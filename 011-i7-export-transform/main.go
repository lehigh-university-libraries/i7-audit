package main

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
)

type IslandoraObject struct {
	Nid []IntField `json:"nid"`
}

type IntField struct {
	Value int `json:"value"`
}

var (
	redirectCache          = make(map[string]int)
	mergedOrDroppedColumns = []string{
		// field_member_of
		"RELS_EXT_isConstituentOf_uri_ms",
		"RELS_EXT_isMemberOfCollection_uri_ms",
		"RELS_EXT_isMemberOf_uri_ms",
		"RELS_EXT_isPageOf_uri_ms",
		// field_linked_agent
		"dc.creator",
		"mods_name_creator_namePart_ms",
		"dc.contributor",
		"dc.publisher",
		"mods_name_photographer_namePart_ms",
		"mods_name_thesis_advisor_namePart_ms",
		// title
		"mods_titleInfo_title_all_ms",
		"mods_titleInfo_title_ms",
		"dc.title",
		// field_description
		"mods_abstract_mt",
		"dc.description",
		// field_resource_type
		"dc.type",
		"mods_typeOfResource_ms",
		"mods_typeOfResource_ss",
		// field_language
		"dc.language",
		"mods_language_languageTerm_ms",
		// field_rights
		"dc.rights",
		"mods_accessCondition_use_and_reproduction_ms",
		// field_edtf_date_created
		"dc.date",
		"mods_originInfo_dateCreated_mdt",
		// field_geographic_subject
		"mods_subject_authority_naf_geographic_ss",
		"mods_subject_geographic_ms",
		"dc.coverage",
		// field_subject
		"mods_subject_topic_ms",
		"dc.subject",
		// ignored
		"ID",
		"file",
		"mods_part_detail_issue_number_ss",
		"mods_part_detail_volume_number_ss",
		// alias of dc.publisher
		"mods_originInfo_publisher_ms",
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
		case "mods_name_creator_affiliation_email_ss":
			updatedHeader = append(updatedHeader, "field_creator_email")
		case "RELS_EXT_embargo-expiry-notification-date_literal_s":
			updatedHeader = append(updatedHeader, columnName)
		case "RELS_EXT_embargo-expiry-notification-date_literal_ss":
			updatedHeader = append(updatedHeader, columnName)
		case "dc.format":
			updatedHeader = append(updatedHeader, columnName)
		case "dc.identifier":
			updatedHeader = append(updatedHeader, "field_identifier")
		case "dc.relation":
			updatedHeader = append(updatedHeader, "field_relation")
		case "dc.source":
			updatedHeader = append(updatedHeader, "field_source")
		case "mods_genre_ms":
			updatedHeader = append(updatedHeader, "field_genre")
		case "mods_genre_valueURI_ms":
			updatedHeader = append(updatedHeader, "field_genre_uri")
		case "mods_identifier_call-number_ms":
			updatedHeader = append(updatedHeader, "field_call_number")
		case "mods_identifier_oclc_ms":
			updatedHeader = append(updatedHeader, "field_oclc_number")
		case "mods_identifier_uri_displayLabel_ms":
			updatedHeader = append(updatedHeader, "field_uri_identifier.title")
		case "mods_identifier_uri_ms":
			updatedHeader = append(updatedHeader, "field_uri_identifier.uri")
		case "mods_location_physicalLocation_ms":
			updatedHeader = append(updatedHeader, "field_physical_location")
		case "mods_name_corporate_department_namePart_ms":
			updatedHeader = append(updatedHeader, "field_department_name")
		case "mods_note_capture_device_ms":
			updatedHeader = append(updatedHeader, "field_capture_device")
		case "mods_note_category_ms":
			updatedHeader = append(updatedHeader, "field_category")
		case "mods_note_ppi_ms":
			updatedHeader = append(updatedHeader, "field_ppi")
		case "mods_note_staff_ms":
			updatedHeader = append(updatedHeader, "field_staff")
		case "mods_originInfo_dateCaptured_ms":
			updatedHeader = append(updatedHeader, "field_date_captured")
		case "mods_originInfo_dateOther_ms":
			updatedHeader = append(updatedHeader, "field_edtf_date")
		case "mods_originInfo_point_end_dateOther_mdt":
			updatedHeader = append(updatedHeader, "field_end_date")
		case "mods_originInfo_point_start_dateOther_mdt":
			updatedHeader = append(updatedHeader, "field_start_date")
		case "mods_originInfo_type_season_dateOther_ms":
			updatedHeader = append(updatedHeader, "field_date_season")
		case "mods_originInfo_type_year_dateOther_ms":
			updatedHeader = append(updatedHeader, "field_date_other")
		case "mods_part_detail_issue_number_s":
			updatedHeader = append(updatedHeader, "field_issue_number")
		case "mods_part_detail_volume_number_s":
			updatedHeader = append(updatedHeader, "field_volume_number")
		case "mods_physicalDescription_digitalOrigin_mt":
			updatedHeader = append(updatedHeader, "field_digital_origin")
		case "mods_physicalDescription_extent_ms":
			updatedHeader = append(updatedHeader, "field_extent")
		case "mods_physicalDescription_form_ms":
			updatedHeader = append(updatedHeader, "field_physical_description")
		case "mods_physicalDescription_form_valueURI_ms":
			updatedHeader = append(updatedHeader, "field_physical_description_uri")
		case "mods_physicalDescription_internetMediaType_ms":
			updatedHeader = append(updatedHeader, "field_media_type")
		case "mods_relatedItem_host_titleInfo_title_ms":
			updatedHeader = append(updatedHeader, "field_host")
		case "mods_relatedItem_original_titleInfo_title_ms":
			updatedHeader = append(updatedHeader, "field_original_title")
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
		"field_resource_type",
		"field_language",
		"field_linked_agent",
		"field_rights",
		"field_edtf_date_created",
		"field_geographic_subject",
		"field_subject",
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

	pids := map[string]bool{}
	for {
		record, err := csvReader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			fmt.Println("Error reading CSV:", err)
			return
		}

		if pids[record[0]] {
			continue
		}
		pids[record[0]] = true

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
	cleanIdentifier(record, columnIndices)

	// the order in which we call these matters since we're appending the CSV header
	// along with appending the new value in the CSV
	// TODO: we should consider refactoring to coordinate this instead
	newRecord := mergeMemberOf(record, columnIndices)
	newRecord = mergeTitle(newRecord, columnIndices)
	newRecord = mergeDescription(newRecord, columnIndices)
	newRecord = mergeType(newRecord, columnIndices)
	newRecord = mergeLanguage(newRecord, columnIndices)
	newRecord = mergeLinkedAgent(newRecord, columnIndices)
	newRecord = mergeRights(newRecord, columnIndices)
	newRecord = mergeDateCreated(newRecord, columnIndices)
	newRecord = mergeGeographicSubject(newRecord, columnIndices)
	newRecord = mergeTopicalSubject(newRecord, columnIndices)

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

		field := getFieldName(columnIndices, k)

		// remove solr's escaped commas
		cell = strings.ReplaceAll(cell, "\\,", ",")
		cell = strings.TrimSpace(cell)

		// remove comma separated values from date fields
		if strings.Contains(field, "date") {
			values := strings.Split(cell, ",")
			newValues := map[string]bool{}
			for _, v := range values {
				v = strings.TrimSpace(v)
				newValues[v] = true
			}

			dates := []string{}
			for date, _ := range newValues {
				dates = append(dates, date)
			}
			cell = strings.Join(dates, "|")
		}

		transformedRecord = append(transformedRecord, cell)
	}

	return transformedRecord
}

func transformModel(record []string, columnIndices map[string]int) {
	column := "RELS_EXT_hasModel_uri_s"
	index := columnIndices[column]
	record[index] = getModel(record[index])
}

func cleanIdentifier(record []string, columnIndices map[string]int) {
	column := "dc.identifier"
	index := columnIndices[column]
	prefixesToIgnore := []string{"islandora:", "digitalcollections:", "preserve:"}

	identifiers := []string{}
	for _, identifier := range strings.Split(record[index], ",") {
		if strStartsWith(identifier, prefixesToIgnore) {
			continue
		}
		identifiers = append(identifiers, strings.TrimSpace(identifier))
	}

	record[index] = strings.Join(identifiers, "|")
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
	} else {
		record = append(record, "[Untitled]")
	}

	return record
}

func mergeRights(record []string, columnIndices map[string]int) []string {
	index1, _ := columnIndices["dc.rights"]
	index2, _ := columnIndices["mods_accessCondition_use_and_reproduction_ms"]
	if record[index1] != "" {
		record = append(record, record[index1])
	} else if record[index2] != "" {
		record = append(record, record[index2])
	} else {
		record = append(record, "")
	}

	return record
}

func mergeType(record []string, columnIndices map[string]int) []string {
	index1, _ := columnIndices["dc.type"]
	index2, _ := columnIndices["mods_typeOfResource_ss"]
	index3, _ := columnIndices["mods_typeOfResource_ms"]
	if record[index1] != "" {
		record = append(record, record[index1])
	} else if record[index2] != "" {
		record = append(record, record[index2])
	} else if record[index3] != "" {
		record = append(record, record[index3])
	} else {
		record = append(record, "")
	}

	return record
}

func mergeLanguage(record []string, columnIndices map[string]int) []string {
	index1, _ := columnIndices["dc.language"]
	index2, _ := columnIndices["mods_language_languageTerm_ms"]
	if record[index1] != "" {
		record = append(record, record[index1])
	} else if record[index2] != "" {
		record = append(record, record[index2])
	} else {
		record = append(record, "")
	}

	return record
}

func mergeDateCreated(record []string, columnIndices map[string]int) []string {
	index1, _ := columnIndices["mods_originInfo_dateCreated_mdt"]
	index2, _ := columnIndices["dc.date"]
	if record[index1] != "" {
		record = append(record, record[index1])
	} else if record[index2] != "" {
		record = append(record, record[index2])
	} else {
		record = append(record, "")
	}

	return record
}

func mergeGeographicSubject(record []string, columnIndices map[string]int) []string {
	fields := map[string]string{
		"mods_subject_authority_naf_geographic_ss": "geographic_naf",
		"mods_subject_geographic_ms":               "geo_location",
		"dc.coverage":                              "geo_location",
	}
	subjects := map[string]bool{}
	for field, vocabulary := range fields {
		index, _ := columnIndices[field]
		if strings.TrimSpace(record[index]) == "" {
			continue
		}

		values := strings.Split(record[index], ";")
		for _, subject := range values {
			subject = fmt.Sprintf("%s:%s", vocabulary, strings.TrimSpace(subject))
			subjects[subject] = true
		}
	}

	uniqSubjects := []string{}
	for subject, _ := range subjects {
		uniqSubjects = append(uniqSubjects, subject)
	}

	record = append(record, strings.Join(uniqSubjects, "|"))
	return record
}

func mergeTopicalSubject(record []string, columnIndices map[string]int) []string {
	fields := []string{
		"mods_subject_topic_ms",
		"dc.subject",
	}
	subjects := map[string]bool{}
	for _, field := range fields {
		index, _ := columnIndices[field]
		if strings.TrimSpace(record[index]) == "" {
			continue
		}

		values := strings.Split(record[index], ";")
		for _, subject := range values {
			subject = strings.TrimSpace(subject)
			subjects[subject] = true
		}
	}

	uniqSubjects := []string{}
	for subject, _ := range subjects {
		uniqSubjects = append(uniqSubjects, subject)
	}

	record = append(record, strings.Join(uniqSubjects, "|"))
	return record
}

func mergeLinkedAgent(record []string, columnIndices map[string]int) []string {
	fields := map[string]string{
		"dc.creator":                           "cre",
		"dc.contributor":                       "ctb",
		"dc.publisher":                         "pbl",
		"mods_name_photographer_namePart_ms":   "pht",
		"mods_name_thesis_advisor_namePart_ms": "ths",
	}
	agents := map[string]bool{}
	for field, relator := range fields {
		index, _ := columnIndices[field]
		if strings.TrimSpace(record[index]) == "" {
			continue
		}

		values := strings.Split(record[index], ";")
		for _, agent := range values {
			agent = fmt.Sprintf("relators:%s:person:%s", relator, strings.TrimSpace(agent))
			agents[agent] = true
		}
	}

	uniqAgents := []string{}
	for agent, _ := range agents {
		uniqAgents = append(uniqAgents, agent)
	}

	record = append(record, strings.Join(uniqAgents, "|"))
	return record
}

func mergeDescription(record []string, columnIndices map[string]int) []string {
	index1, _ := columnIndices["mods_abstract_mt"]
	index2, _ := columnIndices["dc.description"]
	if record[index1] != "" {
		record = append(record, record[index1])
	} else if record[index2] != "" {
		record = append(record, record[index2])
	} else {
		record = append(record, "")
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
	} else {
		record = append(record, "info:fedora/null")
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
	cell = fmt.Sprintf("%s?_format=json", cell)
	if number, err := pid2nid(cell); err == nil {
		record[index] = strconv.Itoa(number)
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

func pid2nid(url string) (int, error) {
	if url == "" {
		return 0, nil
	}
	if cachedNumber, found := redirectCache[url]; found {
		return cachedNumber, nil
	}
	log.Println(url)
	resp, err := http.Get(url)
	if err != nil {
		fmt.Println("Error fetching URL:", err)
		return 0, err
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNotFound {
		redirectCache[url] = 322431
		return redirectCache[url], nil
	}

	if resp.StatusCode != http.StatusOK {
		log.Fatalf("Unable to find node ID for parent %s", url)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Fatalln("Error reading response body:", err)
	}

	var node IslandoraObject
	if err := json.Unmarshal(body, &node); err != nil {
		log.Fatalln("Error unmarshaling JSON:", err)
	}

	redirectCache[url] = node.Nid[0].Value
	return redirectCache[url], nil
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

func strStartsWith(str string, prefixes []string) bool {
	for _, prefix := range prefixes {
		if strings.HasPrefix(str, prefix) {
			return true
		}
	}
	return false
}

func getFieldName(m map[string]int, i int) string {
	for k, v := range m {
		if i == v {
			return k
		}
	}
	return ""
}
