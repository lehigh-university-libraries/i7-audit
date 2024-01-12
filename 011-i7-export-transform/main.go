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
	redirectCache = make(map[string]string)
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
		if columnName != "RELS_EXT_isMemberOfCollection_uri_ms" &&
			columnName != "RELS_EXT_isMemberOf_uri_ms" &&
			columnName != "RELS_EXT_isPageOf_uri_ms" {

			switch columnName {
			case "PID":
				updatedHeader = append(updatedHeader, "field_pid")
			case "RELS_EXT_hasModel_uri_s":
				updatedHeader = append(updatedHeader, "field_model")
			default:
				updatedHeader = append(updatedHeader, columnName)
			}
		}
	}
	updatedHeader = append(updatedHeader, "field_member_of")
	columnNames = append(columnNames, "field_member_of")

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
	// merge the various field_member_of columns into a single column
	index1, _ := columnIndices["RELS_EXT_isMemberOfCollection_uri_ms"]
	index2, _ := columnIndices["RELS_EXT_isMemberOf_uri_ms"]
	index3, _ := columnIndices["RELS_EXT_isPageOf_uri_ms"]
	if record[index1] != "" {
		record = append(record, record[index1])
	} else if record[index2] != "" {
		record = append(record, record[index2])
	} else if record[index3] != "" {
		record = append(record, record[index3])
	}
	renameAndTransform(record, columnIndices, "field_member_of")

	// get the islandora model
	column := "RELS_EXT_hasModel_uri_s"
	index := columnIndices[column]
	record[index] = transformModel(record[index])

	transformedRecord := []string{}
	for k, v := range record {
		if k != index1 && k != index2 && k != index3 {
			transformedRecord = append(transformedRecord, v)
		}
	}

	return transformedRecord
}

func renameAndTransform(record []string, columnIndices map[string]int, columnName string) {
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

func transformModel(model string) string {
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
		return "Digital Document"
	case "info:fedora/islandora:sp_large_image_cmodel":
		return "Image"
	case "info:fedora/islandora:sp_pdf":
		return "Digital Document"
	case "info:fedora/islandora:sp_videoCModel":
		return "Video"
	case "info:fedora/islandora:sp_web_archive":
		return "Binary"
	}

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
	if resp.StatusCode == http.StatusOK {
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
	}

	return url, nil
}
