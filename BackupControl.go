package main 

import (
	"github.com/zavla/dblist/v2" 
	
	//see go.mod file, it uses special module "github.com/zavla" and declares a replacement for it to use local packages in my GOPATH
	"github.com/zavla/dpapi" 
	"github.com/zavla/sendmail" 
	
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/pkg/profile"
)

type pathandfilenames struct {
	Path        string
	Filename    string
	Days        int
	Group       string
	modtime     time.Time
	hasAnyFiles bool // indicating there were some files to choose from
}
type email struct {
	Email []byte
}

// DecryptEmail returns password decrypted by Microsoft DPAPI
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
	defer profile.Start(profile.MemProfile, profile.ProfilePath(".")).Stop()
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

	ConfigItems, err := dblist.ReadConfig(*configfilename)
	if err != nil {
		log.Fatalf("%s\n%s\n", "Error: can't read config file.", err)
	}
	// config lines sorted ascending by filename and suffix
	sort.Slice(ConfigItems, func(i, j int) bool {
		// sorts ascending
		if ConfigItems[i].Filename < ConfigItems[j].Filename {
			return true
		}
		if ConfigItems[i].Filename == ConfigItems[j].Filename &&
			ConfigItems[i].Suffix < ConfigItems[j].Suffix {
			return true // ascending order
		}
		return false
	})

	uniqueconfigpaths := dblist.GetUniquePaths(ConfigItems)         // from config file gets unique folders
	actualFilesInfo := dblist.ReadFilesFromPaths(uniqueconfigpaths) // for every folder gets []filesinfo, retuns map[folder]

	currentSuffixes := []string{"-FULL.bak", "-differ.dif", "-FULL.rar", "-differ.rar", ".rar", ".7z"}

	type outdatedBackup struct {
		dblist.FileInfoWin
		pLine dblist.ConfigLine
	}
	var outdatedFiles []outdatedBackup

	for k := range actualFilesInfo {

		lastFiles := dblist.GetLastFilesGroupedByFunc(actualFilesInfo[k],
			dblist.GroupFunc,
			//bigger(more wide) suffixes comes first
			currentSuffixes)

		// next decides wether file is outdated
		for _, v := range lastFiles {
			line := dblist.FindConfigLineByFilename(v.Name(), currentSuffixes, ConfigItems)
			if line == nil {
				fmt.Printf("Error: for filename %s there is no line in config file.\n", v.Name())
				continue
			}
			days := line.Days
			days++ // backups copied at night. Next day in the morning they are here.
			
			if !line.HasAnyFiles {
				line.HasAnyFiles = true // mark config line that there are some files
			}
			howold := time.Now().Sub(v.ModTime())
			if howold > time.Hour*time.Duration(days*24) {
				aFile := outdatedBackup{v, *line}
				outdatedFiles = append(outdatedFiles, aFile)
			}
		}
	}
	// check if there are any files for evere config line
	var noFilesAtAll []dblist.ConfigLine
	for _, v := range ConfigItems {
		if !v.HasAnyFiles {
			noFilesAtAll = append(noFilesAtAll, v)
		}
	}

	if len(outdatedFiles) > 0 || len(noFilesAtAll) > 0 {
		// send an email
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
		var sb strings.Builder
		sb.WriteString("------ Last outdated backups: -------\n")
		for _, v := range outdatedFiles {
			sb.WriteString("every ")
			sb.WriteString(fmt.Sprintf("%d", v.pLine.Days))
			sb.WriteString(" days 		")
			sb.WriteString(v.Name())
			sb.WriteString(fmt.Sprintf("  file time = %v",v.ModTime()))
			sb.WriteString("\n")
		}
		for _, v := range noFilesAtAll {
			sb.WriteString(fmt.Sprintf(`no backups for config line: {"path":"%s", "filename":"%s", "suffix":"%s"}`+"\n", v.Path, v.Filename, v.Suffix))
		}
		body := sb.String()
		if true {
			fmt.Print(body)
			sendmail.SendMailToMe(c, "arch3", body, "arch3")
		} else {
			fmt.Print(body)
		}
	}
}
