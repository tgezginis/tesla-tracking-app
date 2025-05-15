package utils

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
)


func PlaySound(filename string) error {
	
	absPath, err := filepath.Abs(filename)
	if err != nil {
		return fmt.Errorf("failed to get absolute path: %w", err)
	}

	
	if _, err := os.Stat(absPath); os.IsNotExist(err) {
		return fmt.Errorf("sound file not found: %s", absPath)
	}

	
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		
		cmd = exec.Command("afplay", absPath)
	case "windows":
		
		cmd = exec.Command("cmd", "/C", "start", absPath)
	case "linux":
		
		
		if _, err := exec.LookPath("mpg123"); err == nil {
			cmd = exec.Command("mpg123", "-q", absPath)
		} else if _, err := exec.LookPath("aplay"); err == nil {
			
			cmd = exec.Command("aplay", absPath)
		} else {
			return fmt.Errorf("no suitable audio player found on Linux")
		}
	default:
		return fmt.Errorf("unsupported platform: %s", runtime.GOOS)
	}

	
	go func() {
		if err := cmd.Run(); err != nil {
			fmt.Printf("Error playing sound: %v\n", err)
		}
	}()

	return nil
} 