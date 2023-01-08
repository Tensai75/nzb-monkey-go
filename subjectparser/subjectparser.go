package subjectparser

import (
	"regexp"
	"strconv"
	"strings"
)

type Subject struct {
	Subject       string // full subject
	Header        string // header (remaining text before the filename)
	Filename      string // filename with extension (in the subject it is usually between quotes)
	Basefilename  string // filename without extension(s)
	File          int    // number of the file in the file set (=X in [X/Y])
	TotalFiles    int    // number of total files in the file set (=Y in [X/Y])
	Segment       int    // number of the segment of this file (=X in (X/Y))
	TotalSegments int    // number of total segments for this file (=Y in (X/Y))
}

func Parse(s string) (Subject, error) {

	// new subject structure
	subject := Subject{
		Subject: strings.TrimSpace(s),
	}

	// remainder of the subject string after each parsing step
	var remainder string

	// first we search for file and segment numbers, [X/Y] and (X/Y)
	// we assume that file numbers may or may not be between angle or square brackets and are divided by a slash
	// we assume however that segment numbers are always between round brackets and divided by a slash (Y/X)
	// we also assume that max. two number pairs are present (segment numbers or segment and file numbers)
	// we further assume that if both number pairs are present the last one are the segment numbers,
	// and that no relevant info apperars in the subject after them (usualy only "yEnc" and the size is stated after the segment numbers)
	r := regexp.MustCompile(`(?i)(?:(?P<remainder>.*?) *(?:(?P<files>(?:"?\[|[<[]? *)(?P<file>\d+) */ *(?P<totalfiles>\d+) *(?:\]"?|[>\]])?)|(?P<segments>"?\((?P<segment>\d+) */ *(?P<totalsegments>\d+)\)"?))|.*$)`)
	matches := findAllNamedMatches(r, subject.Subject)
	if matches != nil {
		// check the matches from back to start
		for counter := len(matches) - 1; counter >= 0; counter-- {
			if subject.File == 0 && subject.Segment == 0 {
				// if neither segments nor files are assigned yet, we just assign
				if matches[counter]["files"] != "" {
					subject.File, _ = strconv.Atoi(matches[counter]["file"])
					subject.TotalFiles, _ = strconv.Atoi(matches[counter]["totalfiles"])
				} else if matches[counter]["segments"] != "" { // we have either [files] or (segments) in a match but not both
					subject.Segment, _ = strconv.Atoi(matches[counter]["segment"])
					subject.TotalSegments, _ = strconv.Atoi(matches[counter]["totalsegments"])
				}
			} else if subject.TotalFiles == 0 || subject.TotalSegments == 0 {
				// one of them is already assigned, we do some checks
				if matches[counter]["files"] != "" {
					// check if we have already assigned the file numbers
					// if yes, probably both, files and segments used square brackets
					// in this case we assume that segment numbers are at the end
					// and assigne the previous "file" numbers to the segments
					if subject.TotalFiles != 0 {
						subject.Segment = subject.File
						subject.TotalSegments = subject.TotalFiles
					}
					// check if we have already assigned the segment numbers
					// if not, probably only segment numbers are present but use square brackets
					// in this case we assign the "file" numbers to the segments
					if subject.TotalSegments == 0 {
						subject.Segment, _ = strconv.Atoi(matches[counter]["file"])
						subject.TotalSegments, _ = strconv.Atoi(matches[counter]["totalfiles"])
					} else {
						subject.File, _ = strconv.Atoi(matches[counter]["file"])
						subject.TotalFiles, _ = strconv.Atoi(matches[counter]["totalfiles"])
					}
				} else if matches[counter]["segments"] != "" { // we have either [files] or (segments) in a match but not both
					// check if we have already assigned the segment numbers
					// if yes, probably both, files and segments used round brackets
					// in this case we assume that segment numbers are at the end
					// and assigne these "segment" numbers to the files
					if subject.TotalSegments != 0 {
						subject.File, _ = strconv.Atoi(matches[counter]["segment"])
						subject.TotalFiles, _ = strconv.Atoi(matches[counter]["totalsegments"])
					} else {
						subject.Segment, _ = strconv.Atoi(matches[counter]["segment"])
						subject.TotalSegments, _ = strconv.Atoi(matches[counter]["totalsegments"])
					}
				}
			}
		}
		// combine the remainders of the matches for further parsing
		for counter := 0; counter <= len(matches)-1; counter++ {
			remainder = strings.TrimSpace(remainder + " " + strings.TrimSpace(matches[counter]["remainder"]))
		}
	} else {
		// if we had no match we use the full subject for further parsing
		remainder = subject.Subject
	}
	// check if we have found some segment numbers
	if subject.TotalSegments == 0 {
		// if no segment numbers were found we assume it was a single segment file
		subject.Segment = 1
		subject.TotalSegments = 1
	}
	// check if we have found some file numbers
	if subject.TotalFiles == 0 {
		// if no file numbers were found we first try some edge cases like "x of y"
		r := regexp.MustCompile(`(?i)(?:(?P<remainder1>.*?) *(?P<files>(?:\[|[<[]? *(?:file|datei)?) *(?P<file>\d+) *(?:of|von) *(?P<totalfiles>\d+) *(?:\]|[>\]])?)(?P<remainder2>.*$))`)
		matches := findAllNamedMatches(r, remainder)
		if matches != nil && matches[0]["files"] != "" {
			subject.File, _ = strconv.Atoi(matches[0]["file"])
			subject.TotalFiles, _ = strconv.Atoi(matches[0]["totalfiles"])
			remainder = strings.TrimSpace(strings.TrimSpace(matches[0]["remainder1"]) + " " + strings.TrimSpace(matches[0]["remainder2"]))
		} else {
			// if still nothing was found, we assume it is a single file post
			subject.File = 1
			subject.TotalFiles = 1
		}
	}

	// now search for the header and the file name
	// we first assume that the filename is between quotes and may or may not end with an extension
	// we also assume that there is no more relevant information after the filename
	// everything before the filename is considered to be the header
	r = regexp.MustCompile(`(?i)^(?P<header>.*?)?[- ]*"+(?P<filename>(?P<basefilename>.*?)(?:\.(?P<extension>(?:vol\d+\+\d+\.par2?|part\d+\.[^ "\.]*|[^ "\.]*\.\d+|[^ "\.]*)))?)"+`)
	matches = findAllNamedMatches(r, remainder)
	if matches != nil {
		subject.Header = strings.Trim(matches[0]["header"], " -")
		subject.Filename = strings.Trim(matches[0]["filename"], " -")
		subject.Basefilename = strings.Trim(matches[0]["basefilename"], " -")
	} else {
		// if no filename was found between quotes, we assume the whole remaining subject is the filename and we only search for an extension
		r = regexp.MustCompile(`(?i)^(?P<filename>(?P<basefilename>.*?)\.(?P<extension>(?:vol\d+\+\d+\.par2?|part\d+\.[^ "\.]*|[^ "\.]*\.\d+|[^ "\.]*))(?:[" ]|$))`)
		matches = findAllNamedMatches(r, remainder)
		if matches != nil {
			subject.Filename = strings.Trim(matches[0]["filename"], " -")
			subject.Basefilename = strings.Trim(matches[0]["basefilename"], " -")
		} else {
			// if no filename with extension was found and it is a single file post, we take everything as the (base)fileame
			if subject.TotalFiles == 1 {
				subject.Filename = strings.Trim(remainder, " -")
				subject.Basefilename = strings.Trim(remainder, " -")
			}
		}
	}

	// if header is empty use the basefilename as header
	if subject.Header == "" && subject.Basefilename != "" {
		subject.Header = subject.Basefilename
	}

	return subject, nil

}

// helper function for easier handling of named sub matches
func findAllNamedMatches(regex *regexp.Regexp, str string) map[int]map[string]string {
	matches := regex.FindAllStringSubmatch(str, -1)
	if matches == nil {
		return nil
	}
	results := make(map[int]map[string]string)
	for i, match := range matches {
		results[i] = make(map[string]string)
		for y, value := range match {
			if regex.SubexpNames()[y] != "" {
				results[i][regex.SubexpNames()[y]] = value
			}
		}
	}
	return results
}
