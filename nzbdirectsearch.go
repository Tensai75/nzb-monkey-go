package main

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"errors"
	"fmt"
	"html"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/Tensai75/nzb-monkey-go/nzbparser"
	"github.com/Tensai75/nzb-monkey-go/subjectparser"
	progressbar "github.com/schollz/progressbar/v3"
)

var directsearchHits = make(map[string]map[string]nzbparser.NzbFile)
var directsearchCounter uint64
var startDate int64
var endDate int64
var mutex = sync.Mutex{}

func nzbdirectsearch(engine SearchEngine, name string) error {

	if len(results) > 0 && conf.Directsearch.Skip {
		Log.Info("Results already available. Skipping search based on config settings.")
		return nil
	}

	if conf.Directsearch.Username == "" || conf.Directsearch.Password == "" {
		return fmt.Errorf("No or incomplete credentials for usenet server")
	}
	if len(args.Groups) == 0 {
		return fmt.Errorf("No groups provided")
	}
	if args.UnixDate == 0 {
		return fmt.Errorf("No date provided")
	}
	if conf.Directsearch.Connections == 0 {
		conf.Directsearch.Connections = 20
	}
	if conf.Directsearch.Hours == 0 {
		conf.Directsearch.Hours = 12
	}
	if conf.Directsearch.Scans == 0 {
		conf.Directsearch.Scans = 50
	}
	if conf.Directsearch.Step == 0 {
		conf.Directsearch.Step = 20000
	}

	var searchInGroupError error

	for i, group := range args.Groups {
		if i > 0 && conf.Directsearch.First_group_only && searchInGroupError == nil {
			fmt.Println()
			Log.Info("Skipping other groups based on config settings.")
			return nil
		}
		fmt.Println()
		Log.Info("Searching in group '%s' ...", group)
		searchInGroupError = nil
		if searchInGroupError = searchInGroup(group); searchInGroupError != nil {
			Log.Error(searchInGroupError.Error())
		} else {
			if len(directsearchHits) > 0 {
				for _, hit := range directsearchHits {
					var nzb = &nzbparser.Nzb{}
					for _, files := range hit {
						nzb.Files = append(nzb.Files, files)
					}
					nzbparser.MakeUnique(nzb)
					nzbparser.ScanNzbFile(nzb)
					sort.Sort(nzb.Files)
					for id := range nzb.Files {
						sort.Sort(nzb.Files[id].Segments)
					}
					processResult(nzb, name)
				}
			} else {
				Log.Warn("No result found in group '%s'", group)
			}
		}
	}
	return nil

}

func searchInGroup(group string) error {
	var searchesWG sync.WaitGroup
	var searchesGuard = make(chan struct{}, conf.Directsearch.Scans)
	defer close(searchesGuard)
	var searchesErrorChannel = make(chan error, 1)
	defer close(searchesErrorChannel)
	var searchesCtx, searchesCancel = context.WithCancel(context.Background())
	defer searchesCancel() // Make sure it's called to release resources even if no errors
	var step = conf.Directsearch.Step
	var interval = conf.Directsearch.Hours * 60 * 60
	if !args.IsTimestamp {
		interval += 60 * 60 * 24
	}
	startDate = args.UnixDate - int64(interval)
	endDate = args.UnixDate + int64(60*60*conf.Directsearch.Forward_hours)
	var currentMessageID int
	conn, firstMessageID, lastMessageID, err := switchToGroup(group)
	defer conn.close()
	if err != nil {
		return err
	}
	Log.Info("Scanning from %s to %s", time.Unix(startDate, 0).Format("02.01.2006 15:04:05"), time.Unix(endDate, 0).Format("02.01.2006 15:04:05"))
	currentMessageID, _, err = scanForDate(conn, firstMessageID, lastMessageID, -interval, true, "   Scanning for first message ID ...")
	if err != nil {
		return fmt.Errorf("Error while scanning group '%s' for the first message: %v\n", group, err)
	}
	lastMessageID, _, err = scanForDate(conn, firstMessageID, lastMessageID, 0, false, "   Scanning for last message ID ... ")
	if err != nil {
		return fmt.Errorf("Error while scanning group '%s' for the last message: %v\n", group, err)
	}
	if currentMessageID >= lastMessageID {
		return errors.New("no messages found within search range")
	}
	conn.close()
	directsearchCounter = 0
	bar := progressbar.NewOptions(lastMessageID-currentMessageID,
		progressbar.OptionSetDescription("   Scanning messages ...            "),
		progressbar.OptionSetRenderBlankState(true),
		progressbar.OptionThrottle(time.Millisecond*100),
	)
	go func(bar *progressbar.ProgressBar, ctx context.Context) {
		for {
			select {
			case <-ctx.Done():
				return
			default:
				bar.Set(int(atomic.LoadUint64(&directsearchCounter)))
			}
		}
	}(bar, searchesCtx)
	for currentMessageID <= lastMessageID {
		var lastMessage int
		if currentMessageID+step > lastMessageID {
			lastMessage = lastMessageID
		} else {
			lastMessage = currentMessageID + step
		}
		searchesGuard <- struct{}{} // will block if guard channel is already filled
		searchesWG.Add(1)
		go func(ctx context.Context, currentMessageID int, lastMessage int, group string) {
			defer func() {
				searchesWG.Done()
				<-searchesGuard
			}()
			if err := searchMessages(ctx, currentMessageID, lastMessage, group); err != nil {
				select {
				case searchesErrorChannel <- err:
				default:
				}
				searchesCancel()
				return

			}
		}(searchesCtx, currentMessageID, lastMessage, group)
		// update currentMessageID for next request
		currentMessageID = lastMessage + 1
	}
	searchesWG.Wait()
	if searchesCtx.Err() != nil {
		fmt.Println()
		return <-searchesErrorChannel
	}
	searchesCancel()
	bar.Finish()
	fmt.Println()
	return nil
}

func searchMessages(ctx context.Context, firstMessage int, lastMessage int, group string) error {
	select {
	case <-ctx.Done():
		return nil // Error somewhere, terminate
	default: // required, otherwise it will block
	}
	conn, firstMessageID, lastMessageID, err := switchToGroup(group)
	defer conn.close()
	if err != nil {
		return err
	}
	if firstMessage < firstMessageID {
		firstMessage = firstMessageID
	}
	if lastMessage > lastMessageID {
		lastMessage = lastMessageID
	}
	select {
	case <-ctx.Done():
		return nil // Error somewhere, terminate
	default: // required, otherwise it will block
	}
	results, err := conn.Overview(firstMessage, lastMessage)
	conn.close()
	if err != nil {
		return fmt.Errorf("Error retrieving message overview from the usenet server while searching in group '%s': %v\n", group, err)
	}
	for _, overview := range results {
		select {
		case <-ctx.Done():
			return nil // Error somewhere, terminate
		default: // required, otherwise it will block
		}
		currentDate := overview.Date.Unix()
		if currentDate >= endDate {
			return nil
		}
		subject := html.UnescapeString(strings.ToValidUTF8(overview.Subject, ""))
		searchPattern := regexp.MustCompile("(?i)" + regexp.QuoteMeta(args.Header))
		if match := searchPattern.Match([]byte(subject)); match {
			if subject, err := subjectparser.Parse(subject); err == nil {
				var date int64
				if date = overview.Date.Unix(); date < 0 {
					date = 0
				}
				poster := strings.ToValidUTF8(overview.From, "")
				// make hashes
				headerHash := GetMD5Hash(subject.Header + poster + strconv.Itoa(subject.TotalFiles))
				fileHash := GetMD5Hash(headerHash + subject.Filename + strconv.Itoa(subject.TotalSegments))
				mutex.Lock()
				if _, ok := directsearchHits[headerHash]; !ok {
					directsearchHits[headerHash] = make(map[string]nzbparser.NzbFile)
				}
				if _, ok := directsearchHits[headerHash][fileHash]; !ok {
					directsearchHits[headerHash][fileHash] = nzbparser.NzbFile{
						Groups:       []string{group},
						Subject:      subject.Subject,
						Poster:       poster,
						Number:       subject.File,
						Filename:     subject.Filename,
						Basefilename: subject.Basefilename,
					}
				}
				file := directsearchHits[headerHash][fileHash]
				if file.Groups[len(file.Groups)-1] != group {
					file.Groups = append(file.Groups, group)
				}
				if subject.Segment == 1 {
					file.Subject = subject.Subject
				}
				if int(date) > file.Date {
					file.Date = int(date)
				}
				file.Segments = append(file.Segments, nzbparser.NzbSegment{
					Number: subject.Segment,
					Id:     strings.Trim(overview.MessageId, "<>"),
					Bytes:  overview.Bytes,
				})
				directsearchHits[headerHash][fileHash] = file
				mutex.Unlock()
			}
		}
		atomic.AddUint64(&directsearchCounter, 1)
	}
	return nil
}

func scanForDate(conn *safeConn, firstMessageID int, lastMessageID int, interval int, first bool, text string) (int, time.Time, error) {
	bar := progressbar.NewOptions(lastMessageID-firstMessageID,
		progressbar.OptionSetDescription(text),
		progressbar.OptionSetRenderBlankState(true),
		progressbar.OptionThrottle(time.Millisecond*100),
	)
	defer func() {
		bar.Finish()
		fmt.Println()
	}()
	currentMessageID := firstMessageID
	endMessageID := lastMessageID
	scanStep := endMessageID - currentMessageID
	for currentMessageID <= endMessageID {
		step := 0
		if currentMessageID == firstMessageID {
			step = 2000
		}
		if scanStep < 1000 {
			results, err := conn.Overview(currentMessageID-1000, currentMessageID+1000)
			if err != nil {
				return 0, time.Time{}, err
			}
			for _, overview := range results {
				bar.Add(1)
				if overview.Date.Unix() > endDate+int64(interval) {
					return overview.MessageNumber, overview.Date, nil
				}
			}
			return results[len(results)-1].MessageNumber, results[len(results)-1].Date, nil
		} else {
			results, err := conn.Overview(currentMessageID, currentMessageID+step)
			if err != nil {
				return 0, time.Time{}, err
			}
			if len(results) == 0 {
				return 0, time.Time{}, fmt.Errorf("Overview results are empty")
			}
			overview := results[0]
			currentDate := overview.Date.Unix()
			scanStep = scanStep / 2
			if first && currentMessageID == firstMessageID && currentDate > endDate+int64(interval) {
				return overview.MessageNumber, overview.Date, nil
			} else if !first && currentMessageID == firstMessageID && currentDate > endDate+int64(interval) {
				return 0, time.Time{}, fmt.Errorf("post date is older than oldest message of this group")
			}
			if currentDate < endDate+int64(interval) {
				currentMessageID = currentMessageID + scanStep
			}
			if currentDate > endDate+int64(interval) {
				currentMessageID = currentMessageID - scanStep
			}
		}
		bar.Add(scanStep)

	}
	return 0, time.Time{}, fmt.Errorf("no messages found within search range")
}

func switchToGroup(group string) (*safeConn, int, int, error) {
	conn, err := ConnectNNTP()
	if err != nil {
		return nil, 0, 0, fmt.Errorf("Error connecting to the usenet server: %v", err)
	}
	_, firstMessageID, lastMessageID, err := conn.Group(group)
	if err != nil {
		return nil, 0, 0, fmt.Errorf("Error retrieving group information for group '%s' from the usenet server: %v\n", group, err)
	}
	return conn, firstMessageID, lastMessageID, nil
}

func GetMD5Hash(text string) string {
	hasher := md5.New()
	hasher.Write([]byte(text))
	return hex.EncodeToString(hasher.Sum(nil))
}
