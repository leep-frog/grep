package main

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/leep-frog/command/sourcerer"
	"github.com/leep-frog/grep"
)

func main() {
	// cmd := exec.Command(`powershell.exe`, `-NoProfile`, `C:\Users\gleep\Desktop\Coding\go\src\grep\cmd\here.ps1`)
	// cmd := exec.Command(`powershell.exe`, `-NoProfile`, `echo`, "hello guy")
	cmd := exec.Command(`git`, "status")
	cmd.Stdout = os.Stdout
	if err := cmd.Run(); err != nil {
		fmt.Println("NOPE:", err)
	}
	return
	os.Exit(sourcerer.Source([]sourcerer.CLI{
		grep.HistoryCLI(),
		grep.FilenameCLI(),
		grep.RecursiveCLI(),
		grep.StdinCLI(),
	}))
}
