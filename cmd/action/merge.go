// Copyright 2021-present The Atlas Authors. All rights reserved.
// This source code is licensed under the Apache 2.0 license found
// in the LICENSE file in the root directory of this source tree.

package action

import (
	"errors"
	"fmt"
	"github.com/spf13/cobra"
	"io"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"strings"
)

const (
	defaultDefinitionPartialFileDirectory = "atlas"
	aggregateSchemaDefinitionFile         = "atlas.hcl"
	atlasPartialDefinitionFileExtension   = ".atlas.hcl"
)

var (
	MergeFlags struct {
		Directory         string
		Walk              bool
		IgnoreDirectories []string
		OutputAs          string
	}
	// MergeCmd represents the fmt command.
	MergeCmd = &cobra.Command{
		Use:   "merge [path ...]",
		Short: "Merges Atlas HCL partial definition files, if they exist in the current directory",
		Long: "`atlas schema merge`" + ` merges all ".hcl" files under the given file directory. 
If no directory is provided, the merge tool will default to looking in a root "atlas" directory
for all partial definition files.

After running, the command will re-create the parent aggregate definition file, titled "atlas.hcl" if no
file name is given.
`,
		Run: CmdMergeRun,
		Example: `
atlas schema merge # walks all files in the current directory and merges all atlas files"
atlas schema merge -o my_atlas_file.hcl # walks all files in the current directory and merges all atlas files into the provided filename"
atlas schema merge -w # explicitly walks all files in the current directory and merges all atlas files"
atlas schema merge -p path/to/atlas/files # merges all atlas files in the provided directory"
atlas schema merge -p path/to/atlas/files -w # walks the directory and merges all atlas files"
atlas schema merge -w -i directory/to/ignore # explicitly walks all files in the current direct ignoring the provided directory"
atlas schema merge -w -i directory/to/ignore another/directory/to/ignore # explicitly walks all files in the current direct ignoring all provided directories"`,
	}
	ErrNoBeginningSlash = errors.New("directory path must not start with beginning slash")
)

func init() {
	schemaCmd.AddCommand(MergeCmd)
	MergeCmd.Flags().StringVarP(&MergeFlags.Directory, "directory", "d", "", "Provides the optional path where partial definition files are stored")
	MergeCmd.Flags().BoolVarP(&MergeFlags.Walk, "walk", "w", false, "Tells the merge process to walk the directory for atlas files")
	MergeCmd.Flags().StringArrayVarP(&MergeFlags.IgnoreDirectories, "ignore-directories", "i", []string{""}, "Provides optional directories to ignore while walking")
	MergeCmd.Flags().StringVarP(&MergeFlags.OutputAs, "output", "o", "", "Provides an optional file output path, defaults to atlas.hcl")
}

// CmdMergeRun merges all HCL files in a given directory using canonical HCL formatting
// rules.
func CmdMergeRun(cmd *cobra.Command, args []string) {
	cwd, err := os.Getwd()
	cobra.CheckErr(err)

	atlasFilePath := cwd

	if len(MergeFlags.Directory) > 0 {
		if strings.Index(MergeFlags.Directory, "/") == 0 {
			cobra.CheckErr(ErrNoBeginningSlash)
		}

		atlasFilePath = fmt.Sprintf("%s%c%s", cwd, os.PathSeparator, MergeFlags.Directory)
	}

	if MergeFlags.Walk {
		walkAtlasFiles(cwd, MergeFlags.OutputAs, MergeFlags.IgnoreDirectories)
	} else {
		handleFileMergeFromDirectory(cwd, atlasFilePath, MergeFlags.OutputAs)
	}
}

func combineFiles(cwd string, filesToMerge []string) {
	schemaDefinitionFilePath := fmt.Sprintf("%s%c%s", cwd, os.PathSeparator, aggregateSchemaDefinitionFile)

	if err := os.Remove(schemaDefinitionFilePath); err != nil && !os.IsNotExist(err) {
		cobra.CheckErr(err)
	}

	rehydratedFile, err := os.Create(schemaDefinitionFilePath)

	defer func(rehydratedFile *os.File) {
		if err := rehydratedFile.Close(); err != nil {
			log.Fatal(err)
		}
	}(rehydratedFile)

	cobra.CheckErr(err)

	for _, filePath := range filesToMerge {
		mergeFileIntoSchemaDefinition(filePath, rehydratedFile)
	}
}

func combineFilesFromAtlasPath(path, cwd string) {
	atlasFiles, err := os.ReadDir(path)
	cobra.CheckErr(err)

	var filesToMerge []string

	for _, file := range atlasFiles {
		filePath := fmt.Sprintf("%s%c%s", path, os.PathSeparator, file.Name())
		filesToMerge = append(filesToMerge, filePath)
	}

	combineFiles(cwd, filesToMerge)
}

func handleWalkedDir(path string, d fs.DirEntry, err error, filesToMerge []string) error {
	cobra.CheckErr(err)

	if !d.IsDir() {
		if strings.Contains(d.Name(), atlasPartialDefinitionFileExtension) {
			// TODO: verify valid hcl file
			filesToMerge = append(filesToMerge, path)
		}
	}

	return nil
}

func walkAtlasFiles(pathToWalk, outputAs string, ignoredDirectories []string) {
	var filesToMerge []string

	err := filepath.WalkDir(pathToWalk, func(path string, d fs.DirEntry, err error) error {
		return handleWalkedDir(path, d, err, filesToMerge)
	})

	log.Printf("files to merge: %d", len(filesToMerge))

	cobra.CheckErr(err)
}

// handleFileMergeFromDirectory batches the task to merge all atlas HCL DDL files together.
func handleFileMergeFromDirectory(cwd, path string, outputAs string) {
	combineFilesFromAtlasPath(path, cwd)
}

// mergeFileIntoSchemaDefinition inspects the definition file partial and appends
// to the atlas definition file defined in the root of the project.
func mergeFileIntoSchemaDefinition(filePath string, schemaDefinitionFile *os.File) {
	definitionFilePartial, err := os.Open(filePath)
	cobra.CheckErr(err)

	defer func(definitionFilePartial *os.File) {
		if err := definitionFilePartial.Close(); err != nil {
			log.Fatal(err)
		}
	}(definitionFilePartial)

	_, err = io.Copy(schemaDefinitionFile, definitionFilePartial)
	cobra.CheckErr(err)

	_, err = schemaDefinitionFile.Write([]byte("\n"))
	cobra.CheckErr(err)
}
