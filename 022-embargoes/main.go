package main

import (
	"bufio"
	"encoding/csv"
	"fmt"
	"io"
	"log"
	"os"
)

var (
	pids = map[string]string{}
	vids = map[string]string{}
)

func init() {
	cacheCsv(pids, "pids.csv")
	cacheCsv(vids, "revisions.csv")
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
	inputFilePath := "embargoes.csv"
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

	tables := []string{
		"node__field_embargo_expiry",
		"node_revision__field_embargo_expiry",
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
		// pid,embargo
		nid, nidFound := pids[record[0]]
		vid, vidFound := vids[record[0]]

		if !nidFound || !vidFound || nid == "" || vid == "" {
			log.Fatalf("Missing nid<->pid mapping for %s", record[0])
		}

		for _, table := range tables {
			sql := fmt.Sprintf(`INSERT INTO %s (bundle, deleted, entity_id, revision_id, langcode, delta, field_embargo_expiry_value)
			VALUES
				('islandora_object', 0, %s, %s, 'en', 0, '%s');
`, table, nid, vid, record[1])
			if _, err := w.WriteString(sql); err != nil {
				fmt.Println("Error writing SQL:", err)
				return
			}
		}
	}

	w.Flush()

	fmt.Println("SQL transformation complete. Output written to", outputFilePath)
}
