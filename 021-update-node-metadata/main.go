package main

import (
	"bufio"
	"encoding/csv"
	"fmt"
	"io"
	"log"
	"os"
	"regexp"
	"time"
)

var (
	users = map[string]string{}
	pids  = map[string]string{}
)

func init() {
	cacheCsv(pids, "pids.csv")
	cacheCsv(users, "users.csv")
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
	inputFilePath := "metadata.csv"
	outputFilePath := "update.sql"

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
	w := bufio.NewWriter(outputFile)

	// skip header
	csvReader.Read()

	// the format the time strings are in
	layout := "2006-01-02T15:04:05Z"
	fractionalSecondsPattern := `\.\d+Z`
	tables := []string{
		"node_field_data",
		"node_field_revision",
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
		// pid,owner,status,created,changed
		uid, uidFound := users[record[1]]
		nid, nidFound := pids[record[0]]

		if !uidFound || !nidFound || uid == "" || nid == "" {
			continue
		}

		createdStr := regexp.MustCompile(fractionalSecondsPattern).ReplaceAllString(record[3], "Z")
		created, err := time.Parse(layout, createdStr)
		if err != nil {
			log.Println("Error parsing datetime:", err)
			continue
		}

		changedStr := regexp.MustCompile(fractionalSecondsPattern).ReplaceAllString(record[4], "Z")
		changed, err := time.Parse(layout, changedStr)
		if err != nil {
			log.Println("Error parsing datetime:", err)
			continue
		}

		for _, table := range tables {
			sql := fmt.Sprintf("UPDATE %s SET uid = %s, created = %d, changed = %d WHERE nid = %s;\n", table, uid, created.Unix(), changed.Unix(), nid)
			if _, err := w.WriteString(sql); err != nil {
				fmt.Println("Error writing SQL:", err)
				return
			}
		}
	}

	w.Flush()

	fmt.Println("SQL transformation complete. Output written to", outputFilePath)
}
