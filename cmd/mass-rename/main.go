package main

import (
	"bufio"
	"flag"
	"fmt"
	massRename "github.com/mrmelon54/mass-rename"
	"github.com/spf13/afero"
	"io"
	ioFs "io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

var (
	fileFlag    = flag.String("f", "", "Provide pre-written mapping file")
	genFlag     = flag.String("g", "", "Write a mapping file to this path")
	patternFlag = flag.String("p", "*", "*.go - pattern for Go source files")
	yesFlag     = flag.Bool("y", false, "Complete renaming after closing editor")
)

func main() {
	flag.Parse()

	if *genFlag != "" {
		create, err := os.Create(*genFlag)
		if err != nil {
			fmt.Println("[Error] Failed to write to generated mapping file:", err)
			os.Exit(1)
		}
		genWalkList(create)
		return
	}

	var mappingFile *os.File
	if *fileFlag == "" {
		tmpFile, err := os.CreateTemp("", "*.mass-rename")
		if err != nil {
			fmt.Println("[Error] Failed to create temporary file:", err)
			os.Exit(1)
		}
		defer func() {
			_ = tmpFile.Close()
			_ = os.Remove(tmpFile.Name())
		}()
		mappingFile = tmpFile

		genWalkList(mappingFile)
		openEditor(mappingFile)

		_, err = tmpFile.Seek(0, io.SeekStart)
		if err != nil {
			fmt.Println("[Error] Failed to seek to start:", err)
			os.Exit(1)
		}
	} else {
		create, err := os.Open(*fileFlag)
		if err != nil {
			fmt.Println("[Error] Failed to read mapping file:", err)
			os.Exit(1)
		}
		defer func() {
			_ = create.Close()
		}()
		mappingFile = create
	}

	// parse user rename map
	renameMap, err := massRename.ParseMassRenameMap(mappingFile)
	if err != nil {
		fmt.Println("[Error] Failed to parse map file:", err)
		os.Exit(1)
	}

	if len(renameMap) == 0 {
		fmt.Println("No files to rename")
		return
	}

	fmt.Println("Renaming:")
	for _, i := range renameMap {
		fmt.Printf("- '%s' => '%s'\n", i.Old, i.New)
	}
	fmt.Println()

	// possible ask user to continue
	if !*yesFlag {
		fmt.Printf("Do you wish to rename these files? [Y/n] ")
		line, err := bufio.NewReader(os.Stdin).ReadString('\n')
		if err != nil {
			fmt.Println("[Error] Failed to read user question:", err)
			os.Exit(1)
		}
		switch strings.ToLower(strings.TrimSpace(line)) {
		case "", "y", "yes":
		// yes
		default:
			fmt.Println("Stopping here")
			os.Exit(0)
		}
	}

	// mass rename files
	errs := massRename.MassRename(afero.NewOsFs(), renameMap)
	for i := range errs {
		if errs[i] != nil {
			if i >= len(renameMap) {
				fmt.Println("Error:", err)
			} else {
				fmt.Printf("Error '%s' => '%s': %s\n", renameMap[i].Old, renameMap[i].New, errs[i])
			}
		}
	}
	fmt.Println("Finished successfully")
}

func genWalkList(w io.StringWriter) {
	glob := make([]string, 0)
	fs := afero.NewOsFs()
	err := afero.Walk(fs, ".", func(path string, info ioFs.FileInfo, err error) error {
		if !info.IsDir() {
			if match, err := filepath.Match(*patternFlag, info.Name()); match && err == nil {
				glob = append(glob, path)
			}
		}
		return nil
	})
	if err != nil {
		fmt.Println("[Error] Failed to walk files:", err)
		os.Exit(1)
	}
	for i := range glob {
		_, err := w.WriteString(glob[i] + " => " + glob[i] + "\n")
		if err != nil {
			fmt.Println("[Error] Failed to write ")
			return
		}
	}
}

func openEditor(tmpFile *os.File) {
	cmd := exec.Command("/usr/bin/editor", tmpFile.Name())
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err := cmd.Start()
	if err != nil {
		fmt.Println("[Error] Editor failed to start:", err)
		os.Exit(1)
	}
	err = cmd.Wait()
	if err != nil {
		fmt.Println("[Error] Editor closed with an error:", err)
		os.Exit(1)
	}
}
