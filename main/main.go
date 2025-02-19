package main

import (
	_ "embed"
	"fmt"
	"github.com/maja42/ember"
	"io"
	"lukasolson.net/common"
	"os"
)

func main() {

	defer func() {
		// Check if a panic occurred.
		code := 0

		if r := recover(); r != nil {
			fmt.Println("Panic:", r)
			code = 1
		}

		// Pause if launched from Explorer.
		if isLaunchedFromExplorer() {
			fmt.Print("Press any key to continue...")
			var input string
			// Wait for user input before exiting.
			fmt.Scanln(&input)

			os.Exit(code)
		}
	}()

	embedded, err := checkIfEmbedded()
	if err != nil {
		fmt.Println("Error checking if embedded:", err)
		return
	}

	// Play the installer theme if it exists.
	if common.ThemeMusicSupport {
		_ = playInstallerTheme()
	}

	if embedded {
		err := bootstrap()
		if err != nil {
			panic(err)
		}
	} else {
		PrintHeader() // Only print the header if we're in creator mode.

		fmt.Println("No scripts have been embedded. Running in creator mode.")
		err = createInstaller()
		if err != nil {
			panic(err)
		}
	}

}

func checkIfEmbedded() (bool, error) {

	attachments, err := ember.Open()
	if err != nil {
		fmt.Println("Error opening attachments:", err)
		return false, err
	}
	defer attachments.Close()

	attachmentList := attachments.List()

	// If there are no attachments or only theme.wav, we're in creator mode.

	if len(attachmentList) == 0 || (common.ThemeMusicSupport && len(attachmentList) == 1 && attachmentList[0] == "theme.wav") {
		return false, nil
	} else {
		return true, nil
	}

}

func playInstallerTheme() error {
	attachments, err := ember.Open()
	if err != nil {
		fmt.Println("Error opening attachments:", err)
		return err
	}
	defer attachments.Close()

	// Attempt to directly open "theme.wav"
	audioFile := attachments.Reader("theme.wav")
	if audioFile == nil {
		return fmt.Errorf("theme.wav not found in attachments")
	}

	// Load the file into a byte array.
	audioBytes, err := io.ReadAll(audioFile)
	if err != nil {
		return fmt.Errorf("failed to read audio file: %w", err)
	}

	// Create a new slice and copy the audio data into it.
	copiedAudio := make([]byte, len(audioBytes))
	copy(copiedAudio, audioBytes)

	// Store the data globally so that it stays alive.
	themeWavData = copiedAudio

	// Start playback in a separate goroutine.
	go func(data []byte) {
		if err := playWavFromByteArray(data); err != nil {
			fmt.Println("Error playing theme:", err)
		}
		// Optionally, once playback is complete, clear the global variable.
		// themeWavData = nil
	}(themeWavData)

	return nil
}

func PrintHeader() {
	header := `
 _____        ______      
|  ___|       | ___ \     
| |____  _____| |_/ /   _ 
|  __\ \/ / _ \  __/ | | |
| |___>  <  __/ |  | |_| |
\____/_/\_\___\_|   \__, |
                     __/ |
                    |___/ 
`
	fmt.Println(header)
}
