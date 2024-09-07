package main

import (
	"fmt"
	"html"
	"math"
	"os"
	"os/signal"
	"path/filepath"
	"sort"
	"strings"
	"syscall"
	"time"

	"github.com/Tensai75/nzbparser"
	humanize "github.com/dustin/go-humanize"
	"github.com/fatih/color"
)

type Result struct {
	SearchEngine           string
	Nzb                    *nzbparser.Nzb
	FilesMissing           int
	FilesComplete          bool
	SegmentsMissing        int
	SegmentsMissingPercent float64
	SegmentsComplete       bool
}

// global variables
var (
	appName                   = "NZB Monkey Go"
	appVersion                string
	appExec                   string
	appPath                   string
	homePath                  string
	results                   = make([]Result, 0)
	filesColor, segmentsColor func(a ...interface{}) string
	red                       = color.New(color.FgRed).SprintFunc()
	yellow                    = color.New(color.FgYellow).SprintFunc()
	green                     = color.New(color.FgGreen).SprintFunc()
	blue                      = color.New(color.FgCyan).SprintFunc()
)

func init() {

	var err error
	// set path variables
	if appExec, err = os.Executable(); err != nil {
		Log.Error("Unable to determin application path")
		exit(1)
	}
	appPath = filepath.Dir(appExec)
	if homePath, err = os.UserHomeDir(); err != nil {
		Log.Error("Unable to determin home path")
		exit(1)
	}

	// change working directory
	// important for url protocol handling (otherwise work dir will be system32 on windows)
	if err := os.Chdir(appPath); err != nil {
		Log.Error("Cannot change working directory: ", err)
		os.Exit(1)
	}

	fmt.Println()
	color.Set(color.FgHiYellow)
	Log.Info("%s %s", appName, appVersion)
	color.Unset()

	// graceful handling of manual aborts
	go func() {
		exit := make(chan os.Signal, 1)
		signal.Notify(exit, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)
		<-exit
		logClose() // clean up
		fmt.Println()
		fmt.Println()
		os.Exit(1)
	}()

}

func main() {

	parseArguments()
	setConfPath()
	checkForConfig()
	checkArguments()
	loadConfig()

	fmt.Println()
	Log.Info("Arguments provided:")
	if args.Nzblnk != "" {
		Log.Info("NZBLNK:   %s", blue(args.Nzblnk))
	}
	if args.Title != "" {
		Log.Info("Title:    %s", blue(args.Title))
	}
	if args.Header != "" {
		Log.Info("Header:   %s", blue(args.Header))
	}
	if args.Password != "" {
		Log.Info("Password: %s", blue(args.Password))
	}
	if len(args.Groups) > 0 {
		Log.Info("Groups:   %s", blue(strings.Join(args.Groups[:], ", ")))
	}
	if args.UnixDate > 0 {
		if args.IsTimestamp {
			Log.Info("Date:     %s", blue(time.Unix(args.UnixDate, 0).Format("02.01.2006 15:04:05 MST")))
		} else {
			Log.Info("Date:     %s", blue(time.Unix(args.UnixDate, 0).Format("02.01.2006")))
		}
	}
	if args.Category != "" {
		Log.Info("Category: %s", blue(args.Category))
	}

	for _, name := range conf.Searchengines {
		fmt.Println()
		Log.Info("Searching on %s ...", searchEngines[name].name)
		if err := searchEngines[name].search(searchEngines[name], searchEngines[name].name); err != nil {
			Log.Warn(err.Error())
		}
	}

	if len(results) > 0 && conf.Nzbcheck.BestNZB {
		fmt.Println()
		Log.Info("Using best NZB file found")
		sort.SliceStable(results, func(i, j int) bool {
			// Sort first by files missing
			if results[i].FilesMissing != results[j].FilesMissing {
				return results[i].FilesMissing < results[j].FilesMissing
			}
			// ... and then by segments missing
			return results[i].SegmentsMissingPercent < results[j].SegmentsMissingPercent
		})
		processFoundNzb(&results[0])
	} else {
		fmt.Println()
		Log.Error("No results found for header '%s'", args.Header)
		exit(1)
	}
}

func prettyByteSize(b int) string {
	bf := float64(b)
	for _, unit := range []string{"", "K", "M", "G", "T", "P", "E", "Z"} {
		if math.Abs(bf) < 1000.0 {
			return fmt.Sprintf("%3.2f %sB", bf, unit)
		}
		bf /= 1000.0
	}
	return fmt.Sprintf("%.1fYiB", bf)
}

func processResult(nzb *nzbparser.Nzb, name string) {
	result := Result{
		SearchEngine:           name,
		Nzb:                    nzb,
		FilesMissing:           nzb.TotalFiles - nzb.Files.Len(),
		FilesComplete:          nzb.TotalFiles-nzb.Files.Len() <= conf.Nzbcheck.MaxMissingFiles,
		SegmentsMissing:        nzb.TotalSegments - nzb.Segments,
		SegmentsMissingPercent: float64(float64(nzb.TotalSegments-nzb.Segments) / float64(nzb.TotalSegments) * 100),
		SegmentsComplete:       float64(float64(nzb.TotalSegments-nzb.Segments)/float64(nzb.TotalSegments)*100) <= conf.Nzbcheck.MaxMissingSegmentsPercent,
	}
	if result.FilesComplete {
		filesColor = green
	} else {
		filesColor = red
	}
	if result.SegmentsComplete {
		segmentsColor = green
	} else {
		segmentsColor = red
	}
	Log.Info("Found:    %s", green(fmt.Sprintf("%s (%s)", result.Nzb.Files[0].Subject, humanize.Bytes(uint64(result.Nzb.Bytes)))))
	Log.Info("Files:    %s", filesColor(fmt.Sprintf("%d/%d (Missing files: %d)", result.Nzb.Files.Len(), result.Nzb.TotalFiles, result.FilesMissing)))
	Log.Info("Segments: %s", segmentsColor(fmt.Sprintf("%d/%d (Missing segments: %f %%)", result.Nzb.Segments, result.Nzb.TotalSegments, result.SegmentsMissingPercent)))

	if !conf.Nzbcheck.SkipFailed || (result.FilesComplete && result.SegmentsComplete) {
		if !conf.Nzbcheck.BestNZB || (result.FilesMissing == 0 && result.SegmentsMissing == 0) {
			processFoundNzb(&result)
		} else {
			results = append(results, result)
		}
	} else {
		Log.Warn("NZB file is skipped because it is incomplete!")
	}
}

func processFoundNzb(nzb *Result) {
	Log.Info("Using NZB file from %s", nzb.SearchEngine)
	if !nzb.FilesComplete || !nzb.SegmentsComplete {
		Log.Warn("NZB file is probably incomplete!")
	}
	var category = checkCategories()
	nzb.Nzb.Comment = fmt.Sprintf("Downloaded from %s with %s %s", nzb.SearchEngine, appName, appVersion)
	if nzb.Nzb.Meta == nil {
		nzb.Nzb.Meta = make(map[string]string)
	}
	nzb.Nzb.Meta["title"] = html.EscapeString(args.Title)
	if args.Password != "" {
		nzb.Nzb.Meta["password"] = html.EscapeString(args.Password)
	}
	var err error
	var nzbfile string
	var hasError bool
	if nzbfile, err = nzbparser.WriteString(nzb.Nzb); err == nil {
		for _, target := range conf.General.Targets {
			if err = targets[target].push(nzbfile, category); err != nil {
				Log.Error(err.Error())
				hasError = true
			}
		}
	} else {
		Log.Error(err.Error())
		hasError = true
	}
	if hasError {
		exit(1)
	} else {
		exit(0)
	}
}

// always use exit function to terminate
// cmd window will stay open for the configured time if the program was startet outside a cmd window
func exit(exitCode int) {

	if conf.General.Success_wait_time == 0 {
		conf.General.Success_wait_time = 3
	}
	if conf.General.Error_wait_time == 0 {
		conf.General.Error_wait_time = 10
	}
	wait_time := int(math.Abs(float64(conf.General.Success_wait_time)))
	if exitCode > 0 {
		wait_time = int(math.Abs(float64(conf.General.Error_wait_time)))
	}

	logClose() // clean up

	// pause before ending the program
	fmt.Println()
	for i := wait_time; i >= 0; i-- {
		fmt.Print("\033[G\033[K") // move the cursor left and clear the line
		fmt.Printf("   Ending program in %d seconds %s", i, strings.Repeat(".", wait_time-i))
		if i > 0 {
			time.Sleep(1 * time.Second)
		}
	}
	fmt.Println()
	fmt.Println()
	os.Exit(exitCode)
}
