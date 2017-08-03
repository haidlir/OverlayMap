package lib

import (
	"bufio"
	"encoding/csv"
	"errors"
	"fmt"
	geo "github.com/kellydunn/golang-geo"
	"github.com/gocarina/gocsv"
	// "io/ioutil"
	"os"

)

type Object struct { // Our example struct, you can use "-" to ignore a field
	Name    string	`csv:"nama"`
	Lat     float64	`csv:"lat"`
	Long 		float64 `csv:"long"`
	Geo			*geo.Point
}

type MapCalc struct { // Our example struct, you can use "-" to ignore a field
	Object_001    string
	Object_002    string
	Distance  		float64
}

func checkHeader(fileName string) error {
	csvFile, err := os.Open(fileName)
	if err != nil {
		return err
	}
	reader := csv.NewReader(bufio.NewReader(csvFile))
	firstLine, err := reader.Read()
	if err != nil {
		return err
	}
	statsFirstLine := make(map[string]bool)
	for _, column := range firstLine {
		statsFirstLine[column] = true
	}

	if !statsFirstLine["nama"] || !statsFirstLine["lat"] || !statsFirstLine["long"] {
		return errors.New(fmt.Sprintf("%s: Header is not complete or match (make sure typed in lower-case)", fileName))
	}

	return nil
}

func parseFile(fileName string) ([]*Object, error) {

	var err error
	err = checkHeader(fileName)
	if err != nil {
		return nil, err
	}

	inputFile, err := os.OpenFile(fileName, os.O_RDWR|os.O_CREATE, os.ModePerm)
	if err != nil {
		return nil, err
	}
	defer inputFile.Close()

	dataFile := []*Object{}

	if err := gocsv.UnmarshalFile(inputFile, &dataFile); err != nil { // Load clients from file		
		return nil, errors.New(fmt.Sprintf("%v: %v", fileName, err))
	}

	for _, obj := range dataFile {
		obj.Geo = geo.NewPoint(obj.Lat, obj.Long)
	}

	return dataFile, nil
}

// Return 3 data:
// 1. List of inter-Object's distance within the constraint
// 2. List of inter-Object's distance within the constraint categorized in object 1 (1st)
// 3. List of inter-Object's distance within the constraint categorized in object 2 (2nd)
func mapObject(dataFile_001, dataFile_002 []*Object, maxDist float64) ([]*MapCalc, map[*string][]*MapCalc, map[*string][]*MapCalc) {
	dataMap := []*MapCalc{}
	mapObj1 := make(map[*string][]*MapCalc)
	mapObj2 := make(map[*string][]*MapCalc)
	// maxIteration := float64(len(dataFile_001) * len(dataFile_002))
	// pos := 0

	for _, obj1 := range dataFile_001 {
		for _, obj2 := range dataFile_002 {
			if dist := obj1.Geo.GreatCircleDistance(obj2.Geo) * 1000.; dist < maxDist {
				dataMap = append(dataMap, &MapCalc{obj1.Name, obj2.Name, dist})
				mapObj1[&obj1.Name] = append(mapObj1[&obj1.Name], dataMap[len(dataMap)-1])
				mapObj2[&obj2.Name] = append(mapObj2[&obj2.Name], dataMap[len(dataMap)-1])
			}
			// pos += 1
			// if pos % 100000 == 0 {
			// 	fmt.Println(float64(pos)/maxIteration * 100.)
			// }
		}
	}

	return dataMap, mapObj1, mapObj2
}

func writeFile(projectName string, dataMap []*MapCalc, mapObj1, mapObj2 map[*string][]*MapCalc, pathToFile string) error {
	// First File
	fdataMap, err := os.Create(fmt.Sprintf("%spublic/assets/%v-dataMap.csv", pathToFile, projectName))
	if err != nil {
		return err
	}
	gocsv.MarshalFile(&dataMap, fdataMap)
	// First FIle Done

	//Second File
	fmapObj1, err := os.Create(fmt.Sprintf("%spublic/assets/%v-mapObj1.csv", pathToFile, projectName))
	if err != nil {
		return err
	}
	for key, row := range mapObj1 {
		cmapObj1 := []byte(fmt.Sprintf("%v,%v", *key, len(row)))
		for _, pairMap := range row {
			cmapObj1 = append(cmapObj1, []byte(fmt.Sprintf(",%v (%v)", pairMap.Object_002, pairMap.Distance))...)
		}
		cmapObj1 = append(cmapObj1, []byte("\n")...)
		_, err := fmapObj1.Write(cmapObj1)
		if err != nil {
			return err
		}
	}

	//Third File
	fmapObj2, err := os.Create(fmt.Sprintf("%vpublic/assets/%v-mapObj2.csv", pathToFile, projectName))
	if err != nil {
		return err
	}
	for key, row := range mapObj2 {
		cmapObj2 := []byte(fmt.Sprintf("%v,%v", *key, len(row)))
		for _, pairMap := range row {
			cmapObj2 = append(cmapObj2, []byte(fmt.Sprintf(",%v (%v)", pairMap.Object_002, pairMap.Distance))...)
		}
		cmapObj2 = append(cmapObj2, []byte("\n")...)
		_, err := fmapObj2.Write(cmapObj2)
		if err != nil {
			return err
		}
	}

	return nil
}

func OverlayFiles(fileName1, fileName2 string, projectName string, maxDist float64, pathToFile string) error {
	var err error
	err = nil
	
	dataFile_001, err := parseFile(fmt.Sprintf("%vpublic/assets/%v", pathToFile, fileName1))
	if err != nil {
		return errors.New(fmt.Sprintf("%v: %v", fileName1, err))
	}
	dataFile_002, err := parseFile(fmt.Sprintf("%vpublic/assets/%v", pathToFile, fileName2))
	if err != nil {
		return errors.New(fmt.Sprintf("%v: %v", fileName2, err))
	}

	dataMap, mapObj1, mapObj2 := mapObject(dataFile_001, dataFile_002, maxDist)
	err = writeFile(projectName, dataMap, mapObj1, mapObj2, pathToFile)
	if err != nil {
		fmt.Println(err)
	} else {
		fmt.Println("Results saved to disk.")
	}
	
	return err
}

// func main() {

// 	maxDist := 100. // in M

// 	projectName := "dp-odp-mgl"
// 	fileName1 := "dp_mgl.csv"
// 	fileName2 := "odp_mgl.csv"

// 	err := overlayFiles(fileName1, fileName2, projectName, maxDist)
// 	if err != nil {
// 		fmt.Println(err)
// 	}

// }