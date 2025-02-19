package main

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"github.com/maja42/ember"
	"io"
	"lukasolson.net/common"
	"os"
	"path"
	"path/filepath"
	"strings"
)

const bootstrappedFileName = "bootstrapped"

//go:embed run.bat
var runScript string

func bootstrap() error {

	exit := ValidateExecutableHash()
	if exit {
		return fmt.Errorf("error validating executable hash")
	}

	attachments, err := ember.Open()
	if err != nil {
		return err
	}
	defer attachments.Close()

	if !ValidateAttachmentHashes(attachments) {
		return fmt.Errorf("error validating installer integrity")
	}

	settings, err := GetSettings(attachments)
	if err != nil {
		return err
	}

	applicationName := *settings.ApplicationName

	if applicationName != "" {
		println("Installing " + applicationName)
	}

	// check if the bootstrap has already been run
	scriptExtractDir := *settings.ScriptExtractDir

	pythonExtractDir := *settings.PythonExtractDir

	integrityChecked := false

	if _, err := os.Stat(bootstrappedFileName); os.IsNotExist(err) {
		// if the bootstrap has not been run, extract the Python and program files

		fmt.Println("Performing first time setup...")

		fmt.Println("Reading embedded files...")

		PythonReader := attachments.Reader(common.PythonFilename)

		if PythonReader == nil {
			return fmt.Errorf("error reading Python. Ensure it is embedded in the binary")
		}

		PayloadReader := attachments.Reader(common.ScriptsFilename)

		if PayloadReader == nil {
			return fmt.Errorf("error reading payload. Ensure it is embedded in the binary")
		}

		wheelsReader := attachments.Reader(common.WheelsFolderName)
		if wheelsReader == nil {
			return fmt.Errorf("error reading wheels. Ensure it is embedded in the binary")
		}

		rootFilesReader := attachments.Reader(common.CopyToRootFilename)
		if rootFilesReader == nil {
			return fmt.Errorf("error reading files to copy to root. Ensure it is embedded in the binary")
		}

		// extract the files to the disk
		fmt.Println("Extracting Python redistributable...")

		err = common.StreamToDir(PythonReader, pythonExtractDir)
		if err != nil {
			println("error extracting Python zip file")
			return err
		}

		fmt.Println("Extracting Scripts...")
		err = common.StreamToDir(PayloadReader, scriptExtractDir)
		if err != nil {
			println("error extracting payload zip file")
			return err
		}

		fmt.Println("Extracting Wheels...")

		wheelsDir := path.Join(pythonExtractDir, common.WheelsFolderName)
		requiredWheelsDir := filepath.Join(wheelsDir, "required")
		setupWheelsDir := filepath.Join(wheelsDir, "setup")

		err = common.StreamToDir(wheelsReader, wheelsDir)
		if err != nil {
			println("error extracting wheels zip file")
			return err
		}

		fmt.Println("Extracting files to copy to root...")

		currentWorkingDir, err := os.Getwd()

		err = common.StreamToDir(rootFilesReader, currentWorkingDir)
		if err != nil {
			println("error extracting files to copy to root")
			return err
		}

		fmt.Println("Extracted files successfully.")

		// Validate the integrity of the extracted files
		EmbeddedIntegrityHashes := attachments.Reader(common.ScriptIntegrityFilename)
		if EmbeddedIntegrityHashes == nil {
			panic("Error reading integrity hashes. Ensure they are embedded in the binary.")
		}

		integrityData, err := io.ReadAll(EmbeddedIntegrityHashes)
		if err != nil {
			panic("Error reading data from reader: " + err.Error())
		}

		err, isIntegral := VerifyExtractionIntegrity(integrityData, scriptExtractDir)

		if isIntegral != true {
			fmt.Println("Error validating integrity of extracted files. Please try again or contact the distributor.")

			// quit the program with an error code
			return fmt.Errorf("file integrity check failed")
		}

		integrityChecked = true

		// install the required packages

		fmt.Println("Installing required packages...")

		pythonPath := filepath.Join(pythonExtractDir, "python.exe")

		// Install all setup wheels first
		if err := installWheels(pythonExtractDir, setupWheelsDir); err != nil {
			fmt.Println("Error installing offline wheels.")
		}

		requirementsPath := path.Join(scriptExtractDir, *settings.RequirementsFile)

		// if requirements.txt exists, install the offline requirements first
		if _, err := os.Stat(requirementsPath); err == nil {
			if err := common.RunCommand(pythonPath, []string{common.GetPipName(pythonExtractDir), "install", "--find-links",
				path.Join(requiredWheelsDir) + "/", "--only-binary=:all:", "-r", requirementsPath}); err != nil {
				fmt.Println("Error while installing requirements from disk... ")
			}
		}

		if *settings.OnlineRequirements {
			// Install the online requirements next
			// Install missing requirements from PyPI *without* upgrading
			if err := common.RunCommand(pythonPath, []string{
				common.GetPipName(pythonExtractDir),
				"install",
				"--upgrade-strategy", "only-if-needed",
				"-r", requirementsPath,
			}); err != nil {
				fmt.Println("Error installing missing requirements:", err)
				return err
			}
		}

		// setup script path is relative to the extracted script directory
		setupScriptName := *settings.SetupScript

		// run the setup.py file if configured
		if setupScriptName != "" {
			setupScriptPath := path.Join(scriptExtractDir, setupScriptName)
			if err := common.RunCommand(pythonPath, []string{setupScriptPath}); err != nil {
				fmt.Println("Error running "+setupScriptName+":", err)
				return err
			}
		}

		myHash, err := calculateSelfHash()

		err = common.SaveContentsToFile(bootstrappedFileName, myHash)
		if err != nil {
			fmt.Println("Error saving hash to file:", err)
		}

	}

	// Copy the files to the root directory if they are listed in the settings and they exist
	for _, file := range settings.FilesToCopyToRoot {
		filePath := path.Join(scriptExtractDir, file)
		if common.DoesPathExist(filePath) {
			err = common.CopyFile(filePath, file)
			if err != nil {
				fmt.Println("Error copying file to root")
				return err
			}
		}
	}

	if integrityChecked != true {
		// Validate the integrity of the extracted files
		EmbeddedIntegrityHashes := attachments.Reader(common.ScriptIntegrityFilename)
		if EmbeddedIntegrityHashes == nil {
			panic("Error reading integrity hashes. Ensure they are embedded in the binary.")
		}

		integrityData, err := io.ReadAll(EmbeddedIntegrityHashes)
		if err != nil {
			panic("Error reading data from reader: " + err.Error())
		}

		err, isIntegral := VerifyExtractionIntegrity(integrityData, scriptExtractDir)

		if isIntegral != true {
			fmt.Println("Please rerun the installer to reinstall the environment.")
			err := os.Remove(bootstrappedFileName)
			if err != nil {
				return err
			}

			// quit the program with an error code
			return fmt.Errorf("file integrity check failed")
		}
	}

	// if main script is not set, exit, as there is nothing to run
	if *settings.MainScript == "" {
		fmt.Println("Files installed. Exiting.")
		return nil
	} else {

		// Create the run.bat
		pythonExecutablePath := filepath.Join(pythonExtractDir, "python.exe")
		mainScriptPath := path.Join(scriptExtractDir, *settings.MainScript)

		// replace the placeholders in the runscript with the actual values
		runScript = strings.ReplaceAll(runScript, "{{PYTHON_EXE}}", pythonExecutablePath)
		runScript = strings.ReplaceAll(runScript, "{{MAIN_SCRIPT}}", mainScriptPath)
		runScript = strings.ReplaceAll(runScript, "{{SCRIPTS_DIR}}", scriptExtractDir)

		runBatPath, err := generateRunBatPath(*settings.ApplicationName)
		if err != nil {
			return err
		}

		err = os.WriteFile(runBatPath, []byte(runScript), 0644)

		if err != nil {
			fmt.Println("Error getting absolute path for run.bat")
			return err
		}

		if *settings.RunAfterInstall {
			fmt.Println("Running script...")

			if err := common.RunCommand(runBatPath, os.Args[1:]); err != nil {
				fmt.Println("Error running script")
				return err
			}

			fmt.Println("Script completed.")
		} else {
			fmt.Println("Please run the following command in the command line to run the script:")
			fmt.Println(runBatPath)
		}

	}

	return nil
}

func generateRunBatPath(appName string) (string, error) {
	appName = strings.TrimLeft(appName, ".") // Remove leading periods

	if appName == "" {
		appName = "run"
	}
	return filepath.Abs(appName + ".bat")
}

func installWheels(extractDir, wheelDir string) error {
	pythonPath := filepath.Join(extractDir, "python.exe")
	pipPath := common.GetPipName(extractDir)

	wheelFiles, err := filepath.Glob(filepath.Join(wheelDir, "*.whl"))
	if err != nil {
		return fmt.Errorf("error finding wheel files: %w", err)
	}

	if len(wheelFiles) == 0 {
		fmt.Println("No wheel files found in", wheelDir)
		return nil
	}

	for _, wheelFile := range wheelFiles {
		fmt.Println("Installing:", wheelFile) // Print the wheel file being installed.
		if err := common.RunCommand(pythonPath, []string{pipPath, "install", wheelFile}); err != nil {
			return fmt.Errorf("error installing wheel %s: %w", wheelFile, err)
		}
	}

	return nil
}

func VerifyExtractionIntegrity(integrityData []byte, scriptExtractDir string) (error, bool) {

	// these will be in the form of a json string, so we need to unmarshal them
	var fileHashes []common.FileHash

	// Unmarshal JSON string to slice of FileHash objects
	err := json.Unmarshal(integrityData, &fileHashes)
	if err != nil {
		fmt.Println("Error unmarshalling JSON:", err)
		return nil, false
	}

	// get the hashes of the extracted files
	tamperedFiles, err := common.VerifyDirectoryIntegrity(scriptExtractDir, fileHashes)

	if err != nil {
		panic(err)
	}

	if len(tamperedFiles) > 0 {

		fmt.Println("Error validating integrity of extracted files.")
		fmt.Println("Warning, the following files have been modified since installation:")

		for _, file := range tamperedFiles {
			fmt.Println(" - " + file)
		}

		return nil, false

	} else {
		fmt.Println("Installation integrity validated successfully.")
	}
	return err, true
}

func ValidateExecutableHash() (exit bool) {
	myHash, err := calculateSelfHash()

	if err != nil {
		fmt.Println("Error calculating hash:", err)
		return true
	}

	if common.DoesPathExist("bootstrapped") {
		// read the hash from the file and compare it to the hash of the executable
		fileHash, err := os.ReadFile("bootstrapped")
		if err != nil {
			fmt.Println("Error reading hash file:", err)
			return true
		}

		if strings.TrimSpace(string(fileHash)) != myHash {
			fmt.Println("Error: Executable hash does not match previously accepted hash. File may have been tampered with.")

			fmt.Println("Expected:", string(fileHash))
			fmt.Println("Actual:", myHash)

			fmt.Println("Please validate the Md5 hash with the one supplied by the distributor before continuing")

			common.PressButtonToContinue("Press enter to accept the new hash and continue...")

			err = common.SaveContentsToFile("bootstrapped", myHash)
			if err != nil {
				fmt.Println("Error saving hash to file:", err)
				return true
			}

		} else {
			fmt.Println("Hashes match. File integrity validated.")
		}

	} else {

		fmt.Println("Please validate my Md5 hash before continuing")
		fmt.Println("While the hash is not a guarantee of safety, it is a good indicator of file integrity.")
		fmt.Println("You can validate my hash by running the following command in the command line:")

		var exeName string

		// check if os.Args[0] has .exe extension
		if !strings.HasSuffix(os.Args[0], ".exe") {
			exeName = os.Args[0] + ".exe"
		} else {
			exeName = os.Args[0]
		}

		fmt.Println("certutil -hashfile", exeName, "MD5")
		fmt.Println("Note: If hash values do not match, the file may have been tampered with.")

		common.PressButtonToContinue("Press enter to continue...")
	}
	return false
}

func calculateSelfHash() (string, error) {
	executablePath, err := os.Executable()
	if err != nil {
		fmt.Println("Error getting executable path:", err)
		return "", err
	}

	myHash, err := common.Md5SumFile(executablePath)

	if err != nil {
		fmt.Println("Error getting hash of executable:", err)
		return "", err
	}
	return myHash, err
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
	HashReader := attachments.Reader(common.HashmapName)
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

func ValidateAttachmentHashes(attachments *ember.Attachments) bool {

	attachmentList := attachments.List()

	hashMap, err := GetHashmap(attachments)
	if err != nil {
		return false
	}

	allHashesMatch := true

	for _, attachment := range attachmentList {
		if attachment == common.HashmapName {
			continue
		}

		attachmentReader := attachments.Reader(attachment)

		if attachmentReader == nil {
			fmt.Println("Error reading attachment:", attachment)
			return false
		}

		actualHash, hashesMatch := ValidateHash(attachmentReader, hashMap[attachment])

		expected := hashMap[attachment]

		if expected == "" {
			expected = "<NOT SET>"
		}

		if !hashesMatch {
			fmt.Println("Error validating hash for:", attachment, " -> Expected:", hashMap[attachment], "Actual:", actualHash)
			allHashesMatch = false
		}
	}

	return allHashesMatch
}
