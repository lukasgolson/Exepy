package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"github.com/maja42/ember"
	"io"
	"lukasolson.net/common"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"
)

func main() {

	executablePath, err := os.Executable()
	if err != nil {
		fmt.Println("Error getting executable path:", err)
		return
	}
	myHash, err := common.Md5SumFile(executablePath)

	if err != nil {
		fmt.Println("Error getting hash of executable:", err)
		return
	}

	if common.DoesPathExist("hash") {
		// read the hash from the file and compare it to the hash of the executable
		fileHash, err := os.ReadFile("hash")
		if err != nil {
			fmt.Println("Error reading hash file:", err)
			return
		}

		if strings.TrimSpace(string(fileHash)) != myHash {
			fmt.Println("Error: Executable hash does not match previously accepted hash. File may have been tampered with.")
			return
		} else {
			fmt.Println("Hashes match. File integrity validated.")
		}

	} else {

		fmt.Println("Please validate my Md5 hash with the one supplied by my distributor before continuing")
		fmt.Println("While the hash is not a guarantee of safety, it is a good indicator of file integrity.")
		fmt.Println("You can validate my hash by running the following command in the command line:")
		fmt.Println("certutil -hashfile", os.Args[0], "MD5")
		fmt.Println("It should also match my self-reported hash:", myHash)
		fmt.Println("")
		fmt.Println("Note: If three hash values do not match, the file may have been tampered with.")

		PressButtonToContinue()

		err = common.SaveContentsToFile("hash", myHash)
		if err != nil {
			fmt.Println("Error saving hash to file:", err)
			return
		}
	}

	attachments, err := ember.Open()
	if err != nil {
		fmt.Println("Error opening attachments:", err)
		return
	}
	defer attachments.Close()

	if ValidateHashes(attachments) {
		fmt.Println("Hashes validated successfully.")
	} else {
		fmt.Println("Error validating hashes.")
		return
	}

	// for each hash, compare the hash of the file to the hash in the map
	// if any of the hashes do not match, return an error

	// settings hash

	settings, err := GetSettings(attachments)
	if err != nil {
		fmt.Println("Error reading settings:", err)
		return
	}

	// check if the bootstrap has already been run
	if _, err := os.Stat("bootstrapped"); os.IsNotExist(err) {
		// if the bootstrap has not been run, extract the Python and program files

		fmt.Println("Performing first time setup...")

		PythonReader := attachments.Reader(common.GetPythonEmbedName())

		if PythonReader == nil {
			fmt.Println("Error reading Python. Ensure it is embedded in the binary.")
			return
		}

		PayloadReader := attachments.Reader(common.GetPayloadEmbedName())

		if PayloadReader == nil {
			fmt.Println("Error reading payload. Ensure it is embedded in the binary.")
			return
		}

		// EXTRACT THE WHEELS ZIP FILE
		wheelsReader := attachments.Reader(common.GetWheelsEmbedName())
		if wheelsReader == nil {
			fmt.Println("Error reading wheels. Ensure it is embedded in the binary.")
			return
		}

		// EXTRACT THE PYTHON ZIP FILE
		err = common.DecompressIOStream(PythonReader, settings.PythonExtractDir)
		if err != nil {
			fmt.Println("Error extracting Python zip file:", err)
			return
		}

		// EXTRACT THE PIPELINE ZIP FILE
		err = common.DecompressIOStream(PayloadReader, "")
		if err != nil {
			fmt.Println("Error extracting payload zip file:", err)
			return
		}

		wheelsDir := path.Join(settings.PythonExtractDir, common.GetWheelsEmbedName())

		// EXTRACT THE WHEELS ZIP FILE
		err = common.DecompressIOStream(wheelsReader, wheelsDir)
		if err != nil {
			fmt.Println("Error extracting wheels zip file:", err)
			return
		}

		pythonPath := filepath.Join(settings.PythonExtractDir, "python.exe")

		if err := common.RunCommand(pythonPath, []string{common.GetPipName(settings.PythonExtractDir), "install", "pip", "setuptools", "wheel"}); err != nil {
			fmt.Println("Error building wheels:", err)
			return
		}

		// if requirements.txt exists, install the requirements
		if _, err := os.Stat(settings.RequirementsFile); err == nil {
			if err := common.RunCommand(pythonPath, []string{common.GetPipName(settings.PythonExtractDir), "install", "--find-links", path.Join(wheelsDir) + "/", "--only-binary=:all:", "-r", settings.RequirementsFile}); err != nil {
				fmt.Println("Error while installing requirements from disk... Continuing...", err)
			}
		}

		// run the setup.py file if configured

		if settings.SetupScript != "" {
			if err := common.RunCommand(pythonPath, []string{settings.SetupScript}); err != nil {
				fmt.Println("Error running "+settings.SetupScript+":", err)
				return
			}
		}

		// save a text file to the current directory to indicate that the bootstrap has been run
		if err := os.WriteFile("bootstrapped", []byte("Bootstrap has been run"), os.ModePerm); err != nil {
			fmt.Println("Error saving bootstrap text file:", err)
			return
		}
	}

	attachments.Close()

	// run the payload script

	fmt.Println("Running script...")

	appendedArguments := append([]string{settings.PayloadScript}, os.Args[1:]...)

	if err := common.RunCommand(filepath.Join(settings.PythonExtractDir, "python.exe"), appendedArguments); err != nil {
		fmt.Println("Error running Python script:", err)
		return
	}

	fmt.Println("Script completed.")
	PressButtonToContinue()

}

func PressButtonToContinue() {
	fmt.Println("Press enter to continue.")
	fmt.Print("\a")

	stop := make(chan bool)

	go func() {
		animation := []string{" ", " ", " ", "o", "O", "o", " ", " ", " "}
		i := 0
		for {
			select {
			case <-stop:
				fmt.Printf("\r%s", strings.Repeat(" ", len(strings.Join(animation, ""))))
				return
			default:
				fmt.Printf("\r%s", strings.Join(animation, ""))
				time.Sleep(100 * time.Millisecond)
				animation = append(animation[1:], animation[0])
				i++
				if i == len(animation) {
					i = 0
					animation = []string{" ", " ", " ", "o", "O", "o", " ", " ", " "}
				}
			}
		}
	}()

	reader := bufio.NewReader(os.Stdin)
	_, _ = reader.ReadString('\n')

	stop <- true
}

func GetSettings(attachments *ember.Attachments) (common.PythonSetupSettings, error) {
	ConfigReader := attachments.Reader(common.GetConfigEmbedName())

	if ConfigReader == nil {
		fmt.Println("Error reading config. Ensure it is embedded in the binary.")
		return common.PythonSetupSettings{}, fmt.Errorf("error reading config. Ensure it is embedded in the binary")
	}
	config, err := io.ReadAll(ConfigReader)

	var settings common.PythonSetupSettings
	err = json.Unmarshal(config, &settings)
	return settings, err
}

func GetHashmap(attachments *ember.Attachments) (map[string]string, error) {
	HashReader := attachments.Reader(common.GetHashEmbedName())
	if HashReader == nil {
		fmt.Println("Error reading hash. Ensure it is embedded in the binary.")

		// throw a new error to prevent further execution
		return nil, fmt.Errorf("error reading hash. Ensure it is embedded in the binary")
	}

	hash, err := io.ReadAll(HashReader)

	if err != nil {
		fmt.Println("Error reading hash:", err)
		return nil, err
	}

	var hashMap map[string]string

	err = json.Unmarshal(hash, &hashMap)

	if err != nil {
		fmt.Println("Error unmarshalling hash:", err)
		return nil, err
	}

	return hashMap, nil
}

func ValidateHash(seeker io.ReadSeeker, expectedHash string) (actualHash string, equal bool) {
	actualHash, err := common.HashReadSeeker(seeker)
	if err != nil {
		fmt.Println("Error reading hash:", err)
		return "", false
	}

	if actualHash != expectedHash {
		return actualHash, false
	}

	return actualHash, true
}

func ValidateHashes(attachments *ember.Attachments) bool {

	attachmentList := attachments.List()

	hashMap, err := GetHashmap(attachments)
	if err != nil {
		return false
	}

	allHashesMatch := true

	for _, attachment := range attachmentList {
		if attachment == common.GetHashEmbedName() {
			continue
		}

		attachmentReader := attachments.Reader(attachment)

		if attachmentReader == nil {
			fmt.Println("Error reading attachment:", attachment)
			return false
		}

		actualHash, hashesMatch := ValidateHash(attachmentReader, hashMap[attachment])

		if !hashesMatch {
			fmt.Println("Error validating hash for:", attachment, " -> Expected:", hashMap[attachment], "Actual:", actualHash)
			allHashesMatch = false
		} else {
			fmt.Println("Hash validated for:", attachment, " -> Expected:", hashMap[attachment], "Actual:", actualHash)
		}
	}

	return allHashesMatch
}
