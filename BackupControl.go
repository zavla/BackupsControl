//https://stackoverflow.com/questions/33516053/windows-encrypted-rdp-passwords-in-golang
package main // import "BackupsControl"

import (
	"BackupsControl/dpapi" 
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"BackupsControl/sendmail"
	"sort"
	"strconv"
	"strings"
	"time"
	"github.com/pkg/profile"
)

type pathandfilenames struct {
	Path        string
	Filename    string
	Days        int
	modtime     time.Time
	hasAnyFiles bool // indicating there were some files to choose from
}
type email struct {
	Email []byte
}

func readConfig(filename string) []pathandfilenames {
	f, err := os.Open(filename)
	if err != nil {
		log.Fatalf("%s", err)
	}
	b, err := ioutil.ReadAll(f)
	if err != nil {
		log.Fatalf("%s", err)
	}
	if b[0] == 0xEF || b[0] == 0xBB || b[1] == 0xBB {
		b = b[3:]
	}

	var datastruct []pathandfilenames
	err = json.Unmarshal(b, &datastruct)
	if err != nil {
		log.Fatalf("Zaerror: json structure bad.\n%s", err)
	}
	return datastruct
}

func DecryptEmail(b []byte) string {
	var emailobj email
	err := json.Unmarshal(b, &emailobj)
	if err != nil {
		log.Fatalf("%s", err)
	}
	dec, err := dpapi.Decrypt(emailobj.Email)
	if err != nil {
		log.Fatalf("%s", err)
	}
	s := string(dec)
	return s
}

func extractGroupName(s string) string {
	// all filenames are in the form dbnamehere_2018-10-08T08-00-00-497-FULL.bak
	// if strings.HasPrefix(s, "ubcd_sklad_2010") {
	// 	fmt.Print(s)
	// }
	year := time.Now().Year()
	pos := strings.Index(s, "_"+strconv.Itoa(year))
	if pos == -1 {
		return ""
	}
	return string(s[:pos])
}
func findLastWithGroupName(m []pathandfilenames) {

}
func getUniquePaths(configstruct []pathandfilenames) map[string]int {
	retmap := make(map[string]int)
	for _, str := range configstruct {
		if _, ok := retmap[str.Path]; !ok {
			retmap[str.Path] = 1
		}
	}
	return retmap
}
func readFilesFrompaths(uniqueconfigpaths map[string]int) map[string][]os.FileInfo {

	retmap := make(map[string][]os.FileInfo)
	for k := range uniqueconfigpaths {
		filesinfo, err := ioutil.ReadDir(k)
		if err != nil {
			log.Fatalf("%s", err)
		}

		retmap[k] = filesinfo

	}
	return retmap // map of slices of fileinfos
}

// FindMissedBackups seeks for last file in every group
// looks like MAXIMUM(Filename) GROUP BY groupName
func FindMissedBackups(existingFiles map[string][]os.FileInfo,
	configstruct []pathandfilenames, uniqueconfigpaths map[string]int) []string {

	lenconfig := len(configstruct)
	ret := make([]string, 0, 20)
	for k := range existingFiles {
		sort.Slice(existingFiles[k], func(i, j int) bool {
			return existingFiles[k][i].Name() < existingFiles[k][i].Name()
		})

		// searching the last filename in the group
		var prevGroupName string
		for _, val := range existingFiles[k] {

			prevGroupName = extractGroupName(val.Name())
			if prevGroupName != "" {

				break
			}
		}
		var lastFiles []pathandfilenames // last by time files in the group
		for i, stPathAndFilenames := range existingFiles[k] {
			groupName := extractGroupName(stPathAndFilenames.Name())
			if groupName != prevGroupName { // start of next group groupName == FoodTechnologiesGrishinaCopy
				if prevGroupName == "" {
					prevGroupName = groupName // current group ended
					//fmt.Fprintf(os.Stderr, "Not possible to extract group name: %s\n", stPathAndFilenames.Name())
					continue

				}

				if existingFiles[k][i-1].IsDir() {
					prevGroupName = groupName // current group ended
					continue
				}
				prevGroupName = groupName // current group ended
				// adding _previous_ Filename
				lastFiles = append(lastFiles,
					pathandfilenames{
						Path:     k,
						Filename: existingFiles[k][i-1].Name(),
						modtime:  existingFiles[k][i-1].ModTime(),
					})

			}
		}
		//the last element may trugger appending to lastFiles
		dl := len(existingFiles[k])
		groupName := extractGroupName(existingFiles[k][dl-1].Name())
		if !existingFiles[k][dl-1].IsDir() && groupName != "" {
			lastFiles = append(lastFiles,
				pathandfilenames{
					Path:     k,
					Filename: existingFiles[k][dl-1].Name(),
					modtime:  existingFiles[k][dl-1].ModTime(),
				})
		}

		// For every file from lastFiles checks difference with current datetime
		curTime := time.Now()
		minTimeOfFile := time.Date(curTime.Year(),
			curTime.Month(), curTime.Day(), 0, 0, 0, 0, curTime.Location())
		if minTimeOfFile.Weekday() == time.Monday {
			minTimeOfFile.AddDate(0, 0, -3)
		}
		for _, pathandfile := range lastFiles {
			groupName := extractGroupName(pathandfile.Filename)
			if groupName == "" {
				continue
			}
			configline := sort.Search(lenconfig, func(n int) bool {

				return configstruct[n].Filename >= groupName
			})
			if configline == lenconfig ||
				configstruct[configline].Filename != groupName {
				continue // actual file is not in the configstruct
			}
			days := configstruct[configline].Days
			if days == 0 {
				days = 1
			}

			configstruct[configline].hasAnyFiles = true

			timepoint := minTimeOfFile.Add(-24 * time.Duration(days) * time.Hour)

			if timepoint.Sub(pathandfile.modtime) > 0 {
				//fmt.Printf("%s vs. %s \t %s\n", pathandfile.modtime, timepoint, pathandfile.Filename)
				ret = append(ret, pathandfile.Filename+"\t\t in "+pathandfile.Path)
			}
		}

	}
	// if there were no files for backup group
	for _, confstr := range configstruct {
		if !confstr.hasAnyFiles {
			//fmt.Printf("%s \t\t in %s\n", confstr.Filename, confstr.Path)
			ret = append(ret, confstr.Filename+" \t\t  in "+confstr.Path)
		}
	}
	return ret
}

func savePasswordToFile(configfilename *string, savepassword *string) {
	f, err := os.OpenFile(*configfilename+"_email", os.O_CREATE|os.O_TRUNC, 0)
	if err != nil {
		log.Fatalf("%s", err)
	}
	defer f.Close()
	enc, err := dpapi.Encrypt([]byte(*savepassword))
	if err != nil {
		log.Fatalf("%s", "Encrypting mail password failed.n")
	}
	emailobj := email{Email: enc}

	jsonstr, err := json.Marshal(emailobj)
	if err != nil {
		log.Fatalf("%s", err)
	}
	f.Write([]byte(jsonstr))

}
func printUsage(w io.Writer) {
	fmt.Fprintf(w, "%s", "Usage: BackupsControl -configfilename <name> [-savepassword password]\n")
	return
}
func main() {
	defer profile.Start(profile.MemProfile,profile.ProfilePath(".")).Stop()
	configfilename := flag.String("configfilename", "", `json config file name. Content is [{"path":"j:\b", "Filename":"base1", "Days":2}, ...]`)
	savepassword := flag.String("savepassword", "", "Saves your email password using DPAPI in your config file.")
	flag.Parse()
	if !flag.Parsed() {
		printUsage(os.Stderr)
		flag.PrintDefaults()
		return
	}
	if *configfilename == "" {
		printUsage(os.Stderr)
		flag.PrintDefaults()
		return
	}
	if *savepassword != "" {
		savePasswordToFile(configfilename, savepassword)
		return
	}
	configstruct := readConfig(*configfilename)
	sort.Slice(configstruct, func(i, j int) bool { // to have binary search
		return configstruct[i].Filename < configstruct[j].Filename
	})
	uniqueconfigpaths := getUniquePaths(configstruct)        // from config file gets unique folders
	actualFilesInfo := readFilesFrompaths(uniqueconfigpaths) // for every folder gets []filesinfo, retuns map[folder]

	absentBackups := FindMissedBackups(actualFilesInfo, configstruct, uniqueconfigpaths)

	if len(absentBackups) > 0 {
		// mail
		newconn, _, err := sendmail.GetTLSConnection()
		defer newconn.Close()
		if err != nil {
			log.Fatalf("%s", err)
		}
		fe, err := os.Open(*configfilename + "_email")
		if err != nil {
			log.Fatalf("%s\n File %s not found. Use --savepassword switch.", "Password for the mail decryption failed.\n", *configfilename+"_email")
		}
		b, err := ioutil.ReadAll(fe)
		if err != nil {
			log.Fatalf("%s", err)
		}
		password := DecryptEmail(b)
		c := sendmail.Authenticate(newconn, password)

		body := strings.Join(absentBackups, "\n")
		if true {
			fmt.Print(body)
			sendmail.SendMailToMe(c, "arch3", body, "arch3")
		} else {
			fmt.Print(body)
		}
	}
}
