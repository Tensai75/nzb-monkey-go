package main

import (
	"bufio"
	"bytes"
	"fmt"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	parser "github.com/alexflint/go-arg"
)

// arguments structure
type Args struct {
	Nzblnk      string   `arg:"positional" help:"a qualified NZBLNK URI (nzblnk://?h=...)"`
	Header      string   `arg:"-s,--subject" help:"the header/subject to search for"`
	Title       string   `arg:"-t,--title" help:"the title/tag for the NZB file"`
	Password    string   `arg:"-p,--password" help:"the password to extract the download"`
	Groups      []string `arg:"-g,--group" help:"the group(s) to search in (several groups seperated with space)"`
	Date        string   `arg:"-d,--date" help:"the date the upload was posted to Usenet (either in the format DD.MM.YYYY or as a Unix timestamp)"`
	Category    string   `arg:"-c,--category" help:"the category to use for the target (if supportet by the target)"`
	UnixDate    int64    `arg:"-"` // will hold the parsed Unix timestamp
	IsTimestamp bool     `arg:"-"` // will indicate if exact timestamp was passed as date
	Config      string   `arg:"--config" help:"path to the config file"`
	Debug       bool     `arg:"--debug" help:"logs output to log file"`
	Register    bool     `arg:"--register" help:"register the NZBLNK protocol"`
}

// version information
func (Args) Version() string {
	return " "
}

// additional description
func (Args) Epilogue() string {
	return "   Parameters that are passed as arguments have precedence over the parameters of the NZBLNK.\n\n   For more information visit github.com/Tensai75/nzb-monkey-go\n"
}

// global arguments variable
var args struct {
	Args
}

// parser variable
var argParser *parser.Parser

func parseArguments() {

	parserConfig := parser.Config{
		IgnoreEnv: true,
	}

	// parse flags
	argParser, _ = parser.NewParser(parserConfig, &args)
	if err := parser.Parse(&args); err != nil {
		if err.Error() == "help requested by user" {
			writeHelp(argParser)
			fmt.Println(args.Epilogue())
			exit(0)
		} else if err.Error() == "version requested by user" {
			fmt.Println(args.Version())
			exit(0)
		}
		writeUsage(argParser)
		Log.Error(err.Error())
		exit(1)
	}

}

func checkArguments() {

	if args.Register {
		fmt.Println()
		Log.Info("Registering the 'nzblnk' URL protocol ...")
		registerProtocol()
		exit(0)
	}

	if args.Header == "" && args.Nzblnk == "" {
		writeUsage(argParser)
		Log.Error("You must provide either --subject or a NZBLNK URI")
		exit(1)
	}

	// parse nzblink if provided
	isNzblnk := false
	if args.Nzblnk != "" {
		if nzblnk, err := url.Parse(args.Nzblnk); err == nil {
			if query, err := url.ParseQuery(nzblnk.RawQuery); err == nil {
				if h := query.Get("h"); h != "" && args.Header == "" {
					args.Header = strings.TrimSpace(h)
				} else {
					writeUsage(argParser)
					Log.Error("Invalid NZBLNK URI: missing 'h' parameter")
					exit(1)
				}
				if t := query.Get("t"); t != "" && args.Title == "" {
					args.Title = strings.TrimSpace(t)
				}
				if p := query.Get("p"); p != "" && args.Password == "" {
					args.Password = strings.TrimSpace(p)
				}
				if query.Get("g") != "" && args.Groups == nil {
					for _, group := range query["g"] {
						if strings.Contains(group, ",") || strings.Contains(group, ";") || strings.Contains(group, " ") {
							divider := regexp.MustCompile(` *[,; ] *`)
							for _, splitGroup := range divider.Split(group, -1) {
								args.Groups = append(args.Groups, strings.TrimSpace(splitGroup))
							}
						} else {
							args.Groups = append(args.Groups, strings.TrimSpace(group))
						}
					}
				}
				if d := query.Get("d"); d != "" && args.Date == "" {
					isNzblnk = true
					args.Date = strings.TrimSpace(d)
				}
			}
		}
	}

	// date argument needs to be parsed because it can have two formats
	if args.Date != "" {
		dateRegexDate := regexp.MustCompile(`^[0-3]\d\.[0-1]\d\.(?:19|20)\d\d$`)
		dateRegexTimestamp := regexp.MustCompile(`^[1-9]\d{9}$`)
		var parseError error
		if match := dateRegexDate.FindStringIndex(args.Date); match != nil {
			var date time.Time
			zone, _ := time.Now().Zone()
			if date, parseError = time.Parse("02.01.2006 MST", fmt.Sprintf("%s %s", args.Date, zone)); parseError == nil {
				args.UnixDate = date.Unix()
			}
		} else if match := dateRegexTimestamp.FindStringIndex(args.Date); match != nil {
			args.UnixDate, parseError = strconv.ParseInt(args.Date, 10, 64)
			args.IsTimestamp = true
		} else {
			parseError = fmt.Errorf("ERROR")
		}
		if parseError != nil || args.UnixDate == 0 {
			if isNzblnk {
				Log.Warn("Invalid NZBLNK URI: invalid input for parameter 'd'")
			} else {
				writeUsage(argParser)
				Log.Error("Invalid input for --date")
				exit(1)
			}
		}
	}

	// set title to header if empty
	if args.Title == "" {
		args.Title = args.Header
	}

	// replace a.b. in groups
	for i, group := range args.Groups {
		args.Groups[i] = strings.Replace(group, "a.b.", "alt.binaries.", 1)
	}

}

func writeUsage(parser *parser.Parser) {
	var buf bytes.Buffer
	parser.WriteUsage(&buf)
	scanner := bufio.NewScanner(&buf)
	for scanner.Scan() {
		fmt.Println("   " + scanner.Text())
	}

}

func writeHelp(parser *parser.Parser) {
	var buf bytes.Buffer
	parser.WriteHelp(&buf)
	scanner := bufio.NewScanner(&buf)
	for scanner.Scan() {
		fmt.Println("   " + scanner.Text())
	}

}
