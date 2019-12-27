package main

import (
	"github.com/zavla/dblist/v2"

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
	configfilename := flag.String("configfilename", "", `json config file name. Content example: [{"path":"j:/b", "Filename":"A2", "suffix":"-FULL.bak", "Days":10},]`)
	savepassword := flag.String("savepassword", "", "Saves your email password using DPAPI in your config file.")
	noemail := flag.Bool("noemail", false, "Do not send email.")
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
		fi         dblist.FileInfoWin
		pLine      dblist.ConfigLine
		expiredfor time.Duration
	}
	var outdatedFiles []outdatedBackup

	for k := range actualFilesInfo {

		//bigger(more wide) suffixes need to come first
		lastFiles := dblist.GetLastFilesGroupedByFunc(actualFilesInfo[k],
			dblist.GroupFunc,
			currentSuffixes,
			1)

		// next decides if file is outdated
		for _, v := range lastFiles {
			// v is a last file in its group.
			// Lets find a config parameters for this group.
			line := dblist.FindConfigLineByFilename(v.Name(), currentSuffixes, ConfigItems)
			if line == nil {
				fmt.Printf("Error: for filename %s there is no line in config file.\n", v.Name())
				continue
			}
			days := line.Days
			// days++ // backups are copied at night. The next day in the morning they are here.
			// If they are not then they are missing for at least a day.

			if !line.HasAnyFiles {
				line.HasAnyFiles = true // mark the config line that there are some backup files
			}
			howoldafile := time.Now().Sub(v.ModTime())
			allowedAge := time.Hour * time.Duration(days*24)
			if howoldafile > allowedAge {
				missedfor := (howoldafile) / (time.Hour * 24 * time.Duration(days))
				aFile := outdatedBackup{fi: v, pLine: *line, expiredfor: missedfor}
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
			sb.WriteString("missed ")
			sb.WriteString(fmt.Sprintf("%d backups, ", v.expiredfor))
			sb.WriteString("expected every ")
			sb.WriteString(fmt.Sprintf("%d", v.pLine.Days))
			sb.WriteString(" days, 		last is ")
			sb.WriteString(fmt.Sprintf("  time:%v	", v.fi.ModTime()))
			sb.WriteString(v.fi.Name())
			sb.WriteString("\n")
		}
		for _, v := range noFilesAtAll {
			sb.WriteString(fmt.Sprintf(`no backups for config line: {"path":"%s", "filename":"%s", "suffix":"%s"}`+"\n", v.Path, v.Filename, v.Suffix))
		}
		body := sb.String()
		fmt.Print(body)

		if !*noemail {
			sendmail.SendMailToMe(c, "arch3", body, "arch3")
		}
	}
}
