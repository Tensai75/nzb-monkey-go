package main

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"regexp"
	"strconv"
	"strings"

	color "github.com/TwiN/go-color"
)

type Categories []string

func checkCategories() string {

	if conf.General.Categorize == "auto" {
		fmt.Println()
		Log.Info("Automatic checking for categories ...")
		for category, regex := range conf.Categories {
			if categoryRegexp, err := regexp.Compile("(?i)" + regex); err == nil {
				if categoryRegexp.Match([]byte(args.Title)) {
					Log.Info("Using category '%s'", category)
					return category
				}
			} else {
				Log.Warn("Error in the Regexp for '%s'", category)
			}
		}
		Log.Warn("No category did match")
		return ""
	}

	if conf.General.Categorize == "manual" && targets[conf.General.Target].getCategories != nil {
		fmt.Println()
		Log.Info("Manual category selection")
		Log.Info("Getting categories from %s ...", targets[conf.General.Target].name)
		if categories, err := targets[conf.General.Target].getCategories(); err == nil {
			if len(categories) > 0 {
				fmt.Printf("   Please select category:\n")
				for i, category := range categories {
					fmt.Printf("%s             %d - %s%s\n", color.Cyan, i+1, category, color.Reset)
				}
				fmt.Printf("%s             X - no category%s\n", color.Cyan, color.Reset)
				input := 0
				for input == 0 {
					fmt.Print("   Enter the number of the category: ")
					str := inputReader()
					if str == "x" || str == "X" {
						Log.Info("No category was selected")
						return ""
					}
					input, err = strconv.Atoi(str)
					if input > 0 && input <= len(categories) {
						Log.Info("Using category '%s'", categories[input-1])
						return categories[input-1]
					} else {
						input = 0
					}
				}
			} else {
				Log.Warn("%s returned no categories", targets[conf.General.Target].name)
			}
		} else {
			Log.Error("Unable to get categories: %s", err.Error())
		}
	}
	return ""
}

func inputReader() string {
	reader := bufio.NewReader(os.Stdin)
	input, err := reader.ReadString('\n')
	if err != nil {
		if errors.Is(err, io.EOF) { // prefered way by GoLang doc
			os.Exit(0)
		}
		Log.Warn("An error occurred while reading input. Please try again", err)
		return ""
	}
	return strings.TrimSpace(input)
}
