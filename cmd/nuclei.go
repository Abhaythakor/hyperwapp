package cmd

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/Abhaythakor/hyperwapp/detect"
	"github.com/Abhaythakor/hyperwapp/model"
	"github.com/Abhaythakor/hyperwapp/util"
	"github.com/spf13/cobra"
)

var nucleiCmd = &cobra.Command{
	Use:   "nuclei [output_file.jsonl]",
	Short: "Post-process HyperWapp output to generate Nuclei tags",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		filePath := args[0]
		file, err := os.Open(filePath)
		if err != nil {
			util.Fatal("Could not open file: %v", err)
		}
		defer file.Close()

		tagMap := make(map[string]struct{})
		var allTags []string

		scanner := bufio.NewScanner(file)
		for scanner.Scan() {
			var d model.Detection
			if err := json.Unmarshal(scanner.Bytes(), &d); err != nil {
				continue // Skip malformed lines
			}

			tag := detect.MapToNucleiTag(d.Technology)
			if tag != "" {
				if _, exists := tagMap[tag]; !exists {
					tagMap[tag] = struct{}{}
					allTags = append(allTags, tag)
				}
			}
		}

		if len(allTags) == 0 {
			util.Info("No technologies with Nuclei tags found in %s", filePath)
			return
		}

		color := util.NewColorizer(!disableColor)
		fmt.Printf("\n[+] %s: %s\n", color.Cyan("Discovered Nuclei Tags"), strings.Join(allTags, ", "))
		fmt.Printf("[>] %s: nuclei -l targets.txt -tags %s\n\n", color.Yellow("Run Nuclei"), strings.Join(allTags, ","))
	},
}

func init() {
	rootCmd.AddCommand(nucleiCmd)
}
