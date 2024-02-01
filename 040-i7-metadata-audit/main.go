package main

import (
	"encoding/xml"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

type Mods struct {
	XMLName   xml.Name    `xml:"mods"`
	TitleInfo []TitleInfo `xml:"titleInfo"`
	Names     []Name      `xml:"name"`
	// Add other fields as per your XML structure
}

type TitleInfo struct {
	Title string `xml:"title"`
}

type Name struct {
	NamePart string `xml:"namePart"`
	// Include other sub-elements if necessary
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

	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			fmt.Printf("Error accessing %s: %v\n", path, err)
			return err
		}
		if !info.IsDir() {
			// read the i7 MODS we downloaded locally
			pid := fmt.Sprintf("%s:%s", filepath.Base(filepath.Dir(path)), strings.ReplaceAll(info.Name(), ".xml", ""))
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
			if !modsMatch(i7, i2) {
				return fmt.Errorf("MODS not identical for %s: %s\n\n================\n\n%s", pid, string(i7Mods), string(i2Mods))
			}
		}

		return nil
	})

	if err != nil {
		fmt.Printf("Error walking directory: %v\n", err)
		return
	}
}

func modsMatch(m1, m2 Mods) bool {
	for i, titleInfo := range m1.TitleInfo {
		if i >= len(m2.TitleInfo) || titleInfo.Title != m2.TitleInfo[i].Title {
			return false
		}
	}

	/*
		for i, name := range m1.Names {
			if i >= len(m2.Names) || name.NamePart != m2.Names[i].NamePart {
				return false
			}
		}
	*/

	return true
}
