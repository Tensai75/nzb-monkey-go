package main

import (
	"fmt"
	"math"
	"os"
	"os/signal"
	"path/filepath"
	"sort"
	"strings"
	"syscall"
	"time"

	"github.com/Tensai75/nzb-monkey-go/nzbparser"
	color "github.com/TwiN/go-color"
	humanize "github.com/dustin/go-humanize"
)

type Result struct {
	SearchEngine     string
	Nzb              *nzbparser.Nzb
	FilesMissing     int
	FilesComplete    bool
	SegmentsMissing  float64
	SegmentsComplete bool
}

// global variables
var (
	appName    = "NZB Monkey Go"
	appVersion string
	appExec    string
	appPath    string
	confFile   = "config.txt"
	results    = make([]Result, 0)
)

func init() {

	// set apPath variable
	appExec, _ = os.Executable()
	appPath = filepath.Dir(appExec)

	// change working directory
	// important for url protocol handling (otherwise work dir will be system32 on windows)
	os.Chdir(appPath)

	fmt.Println()
	Log.Info("%s%s %s%s", color.Yellow, appName, appVersion, color.Reset)

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
	checkForConfig()
	checkArguments()
	loadConfig()

	fmt.Println()
	Log.Info("Arguments provided:")
	if args.Nzblnk != "" {
		Log.Info("NZBLNK:   %s%s%s", color.Blue, args.Nzblnk, color.Reset)
	}
	if args.Title != "" {
		Log.Info("Title:    %s%s%s", color.Blue, args.Title, color.Reset)
	}
	if args.Header != "" {
		Log.Info("Header:   %s%s%s", color.Blue, args.Header, color.Reset)
	}
	if args.Password != "" {
		Log.Info("Password: %s%s%s", color.Blue, args.Password, color.Reset)
	}
	if len(args.Groups) > 0 {
		Log.Info("Groups:   %s%s%s", color.Blue, strings.Join(args.Groups[:], ", "), color.Reset)
	}
	if args.UnixDate > 0 {
		Log.Info("Date:     %s%s%s", color.Blue, time.Unix(args.UnixDate, 0).Format("02.01.2006 15:04:05"), color.Reset)
	}
	if args.Category != "" {
		Log.Info("Category: %s%s%s", color.Blue, args.Category, color.Reset)
	}

	for _, name := range conf.Searchengines {
		fmt.Println()
		Log.Info("Searching on %s ...", searchEngines[name].name)
		if nzb, err := searchEngines[name].search(searchEngines[name]); nzb != nil {
			result := Result{
				SearchEngine:     name,
				Nzb:              nzb,
				FilesMissing:     nzb.TotalFiles - nzb.Files.Len(),
				FilesComplete:    nzb.TotalFiles-nzb.Files.Len() <= conf.Nzbcheck.Max_missing_files,
				SegmentsMissing:  float64(float64(nzb.TotalSegments-nzb.Segments) / float64(nzb.TotalSegments) * 100),
				SegmentsComplete: float64(float64(nzb.TotalSegments-nzb.Segments)/float64(nzb.TotalSegments)*100) <= conf.Nzbcheck.Max_missing_segments_percent,
			}
			var filesColor string
			if result.FilesComplete {
				filesColor = color.Green
			} else {
				filesColor = color.Red
			}
			var segmentsColor string
			if result.SegmentsComplete {
				segmentsColor = color.Green
			} else {
				segmentsColor = color.Red
			}
			Log.Info("Found:    %s%s (%s)%s", color.Green, result.Nzb.Files[0].Subject, humanize.Bytes(uint64(result.Nzb.Bytes)), color.Reset)
			Log.Info("Files:    %s%d/%d (Missing files: %d)%s", filesColor, result.Nzb.Files.Len(), result.Nzb.TotalFiles, result.FilesMissing, color.Reset)
			Log.Info("Segments: %s%d/%d (Missing segments: %f %%)%s", segmentsColor, result.Nzb.Segments, result.Nzb.TotalSegments, result.SegmentsMissing, color.Reset)

			if !conf.Nzbcheck.Best_nzb && (!conf.Nzbcheck.Skip_failed || (result.FilesComplete && result.SegmentsComplete)) {
				processFoundNzb(&result)
			} else {
				if !conf.Nzbcheck.Skip_failed || (result.FilesComplete && result.SegmentsComplete) {
					results = append(results, result)
				}
			}
		} else if err != nil {
			Log.Warn(err.Error())
		}
	}

	if len(results) > 0 && conf.Nzbcheck.Best_nzb {
		fmt.Println()
		Log.Info("Using best NZB file found")
		sort.SliceStable(results, func(i, j int) bool {
			return results[i].FilesMissing < results[j].FilesMissing
		})
		sort.SliceStable(results, func(i, j int) bool {
			return results[i].SegmentsMissing < results[j].SegmentsMissing
		})
		processFoundNzb(&results[0])
	} else {
		fmt.Println()
		Log.Error("No results found for header '%s'", args.Header)
		exit(1)
	}

}

func processFoundNzb(nzb *Result) {
	Log.Info("Using NZB file from %s", searchEngines[nzb.SearchEngine].name)
	if !(nzb.FilesComplete && nzb.SegmentsComplete) {
		Log.Warn("NZB file is probably incomplete!")
	}
	var category = checkCategories()
	nzb.Nzb.Comment = fmt.Sprintf("Downloaded from %s with %s %s", searchEngines[nzb.SearchEngine].name, appName, appVersion)
	if nzb.Nzb.Meta == nil {
		nzb.Nzb.Meta = make(map[string]string)
	}
	nzb.Nzb.Meta["title"] = fmt.Sprintf("%s", args.Title)
	if args.Password != "" {
		nzb.Nzb.Meta["password"] = fmt.Sprintf("%s", args.Password)
	}
	var err error
	var nzbfile string
	if nzbfile, err = nzbparser.WriteString(nzb.Nzb); err == nil {
		if err = targets[conf.General.Target].push(nzbfile, category); err == nil {
			exit(0)
		}
	}
	Log.Error(err.Error())
	exit(1)
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
