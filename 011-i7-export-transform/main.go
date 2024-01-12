package main

import (
	"encoding/csv"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
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

		transformColumns(record, columnIndices)

		if err := csvWriter.Write(record); err != nil {
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

func transformColumns(record []string, columnIndices map[string]int) {
	renameAndTransform(record, columnIndices, "RELS_EXT_isMemberOfCollection_uri_ms")
	renameAndTransform(record, columnIndices, "RELS_EXT_isMemberOf_uri_ms")
	renameAndTransform(record, columnIndices, "RELS_EXT_isPageOf_uri_ms")
}

func renameAndTransform(record []string, columnIndices map[string]int, columnName string) {
	index, found := columnIndices[columnName]
	if !found {
		return
	}

	cell := record[index]
	cell = strings.ReplaceAll(cell, "info:fedora/", "https://islandora-stage.lib.lehigh.edu/islandora/object/")

	if number, err := getNumberFromRedirect(cell); err == nil {
		record[index] = number
	}
}

var redirectCache = make(map[string]string)

func getNumberFromRedirect(url string) (string, error) {
	if url == "" {
		return "", nil
	}
	if cachedNumber, found := redirectCache[url]; found {
		return cachedNumber, nil
	}

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
