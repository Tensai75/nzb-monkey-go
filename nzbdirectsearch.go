package main

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"errors"
	"fmt"
	"html"
	"sort"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/Tensai75/nntp"
	"github.com/Tensai75/nntpPool"
	"github.com/Tensai75/nzbparser"
	"github.com/Tensai75/subjectparser"
	progressbar "github.com/schollz/progressbar/v3"
)

const overviewStep = 1000

var directsearchHits = make(map[string]map[string]nzbparser.NzbFile)
var directsearchCounter uint64
var startDate int64
var endDate int64
var mutex = sync.Mutex{}

type messageResult struct {
	MessageNumber int
	Date          time.Time
}

func nzbdirectsearch(engine SearchEngine, name string) error {

	if len(results) > 0 && conf.Directsearch.Skip {
		Log.Info("Results already available. Skipping search based on config settings.")
		return nil
	}

	if conf.Directsearch.Username == "" || conf.Directsearch.Password == "" {
		return errors.New("no or incomplete credentials for usenet server")
	}
	if len(args.Groups) == 0 {
		return errors.New("no groups provided")
	}
	if args.UnixDate == 0 {
		return errors.New("no date provided")
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

	if err := initNntpPool(); err != nil {
		return err
	} else {
		defer pool.Close()
	}

	for i, group := range args.Groups {
		if i > 0 && conf.Directsearch.FirstGroupOnly && searchInGroupError == nil {
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
	startDate = args.UnixDate - int64(conf.Directsearch.Hours*60*60)
	endDate = args.UnixDate + int64(60*60*conf.Directsearch.ForwardHours)
	if !args.IsTimestamp {
		endDate += 60 * 60 * 24
	}
	var currentMessageID, lastMessageID int
	var messageDate time.Time
	var err error
	Log.Info("Scanning from %s to %s", time.Unix(startDate, 0).Format("02.01.2006 15:04:05 MST"), time.Unix(endDate, 0).Format("02.01.2006 15:04:05 MST"))
	currentMessageID, messageDate, err = getFirstMessageNumberFromGroup(group, startDate, endDate, searchesCtx)
	if err != nil {
		return fmt.Errorf("error while scanning group '%s' for the first message: %v", group, err)
	}
	Log.Info("Found first message number: %v / Date: %v", FormatNumberWithApostrophe(currentMessageID), messageDate.Local().Format("02.01.2006 15:04:05 MST"))
	lastMessageID, messageDate, err = getLastMessageNumberFromGroup(group, endDate, startDate, searchesCtx)
	if err != nil {
		return fmt.Errorf("error while scanning group '%s' for the last message: %v", group, err)
	}
	if currentMessageID >= lastMessageID {
		return errors.New("no messages found within search range")
	}
	Log.Info("Found last message number: %v / Date: %v", FormatNumberWithApostrophe(lastMessageID), messageDate.Local().Format("02.01.2006 15:04:05 MST"))
	Log.Info("Scanning messages %v to %v (%v messages in total)", FormatNumberWithApostrophe(currentMessageID), FormatNumberWithApostrophe(lastMessageID), FormatNumberWithApostrophe(lastMessageID-currentMessageID+1))
	directsearchCounter = 0
	bar := progressbar.NewOptions(lastMessageID-currentMessageID,
		progressbar.OptionSetDescription("   Scanning messages ...                "),
		progressbar.OptionSetRenderBlankState(true),
		progressbar.OptionThrottle(time.Millisecond*100),
		progressbar.OptionShowElapsedTimeOnFinish(),
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
	conn, firstMessageID, lastMessageID, err := switchToGroup(group, ctx)
	if err != nil {
		return err
	}
	defer pool.Put(conn)
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
	if err != nil {
		return fmt.Errorf("error retrieving message overview from the usenet server while searching in group '%s': %v", group, err)
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
		if strings.Contains(strings.ToLower(subject), strings.ToLower(args.Header)) {
			if subject, err := subjectparser.Parse(subject); err == nil {
				var date int64
				if date = overview.Date.Unix(); date < 0 {
					date = 0
				}
				poster := strings.ToValidUTF8(overview.From, "")
				// make hashes
				headerHash := GetMD5Hash(subject.Header + poster + strconv.Itoa(subject.TotalFiles))
				fileHash := GetMD5Hash(headerHash + subject.Filename + strconv.Itoa(subject.File) + strconv.Itoa(subject.TotalSegments))
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
					file.Groups = append(file.Groups, html.EscapeString(group))
				}
				if subject.Segment == 1 {
					file.Subject = subject.Subject
				}
				if int(date) > file.Date {
					file.Date = int(date)
				}
				file.Segments = append(file.Segments, nzbparser.NzbSegment{
					Number: subject.Segment,
					Id:     html.EscapeString(strings.Trim(overview.MessageId, "<>")),
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

func switchToGroup(group string, ctx context.Context) (*nntpPool.NNTPConn, int, int, error) {
	conn, err := pool.Get(ctx)
	if err != nil {
		return nil, 0, 0, fmt.Errorf("unable to connect to the usenet server: %v", err)
	}
	_, firstMessageID, lastMessageID, err := conn.Group(group)
	if err != nil {
		return nil, 0, 0, fmt.Errorf("unable to retrieve group information for group '%s' from the usenet server: %v", group, err)
	}
	return conn, firstMessageID, lastMessageID, nil
}

func GetMD5Hash(text string) string {
	hasher := md5.New()
	hasher.Write([]byte(text))
	return hex.EncodeToString(hasher.Sum(nil))
}

func getFirstMessageNumberFromGroup(group string, startDate int64, endDate int64, ctx context.Context) (int, time.Time, error) {
	message, date, err := findMessageByDate(group, startDate, true, ctx)
	if err != nil {
		return 0, time.Time{}, err
	}
	if date.Unix() > endDate {
		return 0, time.Time{}, errors.New("the oldest message in the group is newer than the specified end date")
	}
	return message, date, nil
}

func getLastMessageNumberFromGroup(group string, endDate int64, startDate int64, ctx context.Context) (int, time.Time, error) {
	message, date, err := findMessageByDate(group, endDate, false, ctx)
	if err != nil {
		return 0, time.Time{}, err
	}
	if date.Unix() < startDate {
		return 0, time.Time{}, errors.New("the newest message in the group is older than the specified start date")
	}
	return message, date, nil
}

// Common binary search function for finding messages by date
func findMessageByDate(group string, targetDate int64, searchForFirst bool, ctx context.Context) (int, time.Time, error) {
	conn, err := pool.Get(ctx)
	if err != nil {
		return 0, time.Time{}, fmt.Errorf("failed to get connection: %w", err)
	}
	defer pool.Put(conn)

	_, firstMessageNumber, lastMessageNumber, err := conn.Group(group)
	if err != nil {
		return 0, time.Time{}, fmt.Errorf("failed to get group info: %w", err)
	}

	// Check if group is empty
	if firstMessageNumber >= lastMessageNumber {
		return 0, time.Time{}, fmt.Errorf("group '%s' appears to be empty", group)
	}

	low := firstMessageNumber
	high := lastMessageNumber

	// Calculate estimated maximum iterations for binary search
	totalSearchSpace := lastMessageNumber - firstMessageNumber + 1
	maxIterations := calcMaxIterations(totalSearchSpace)

	var description string
	var direction string
	var noResultError string
	var boundaryError string

	if searchForFirst {
		description = "   Scanning for first message number ..."
		direction = "up"
		noResultError = "no messages found on or after the specified start date"
		boundaryError = "the newest message in the group is older than the specified start date"
	} else {
		description = "   Scanning for last message number ...  "
		direction = "down"
		noResultError = "no messages found on or before the specified end date"
		boundaryError = "the oldest message in the group is newer than the specified end date"
	}

	bar := progressbar.NewOptions(maxIterations,
		progressbar.OptionSetDescription(description),
		progressbar.OptionSetRenderBlankState(true),
		progressbar.OptionThrottle(time.Millisecond*100),
		progressbar.OptionShowElapsedTimeOnFinish(),
	)
	defer func() {
		bar.Finish()
		fmt.Println()
	}()

	var lastStep = false
	var lastResult messageResult
	iterationCount := 0

	// Binary search for the target message
	for low <= high {
		iterationCount++

		// Check for context cancellation
		select {
		case <-ctx.Done():
			return 0, time.Time{}, ctx.Err()
		default:
		}

		// Ensure boundaries are within valid message range
		if low < firstMessageNumber {
			low = firstMessageNumber
		}
		if high > lastMessageNumber {
			high = lastMessageNumber
		}
		if low > high {
			break
		}

		// Calculate the mid point
		mid := low + (high-low)/2

		// Calculate the overview range as +/- overviewStep/2 around mid
		overviewStart := mid - (overviewStep / 2)
		overviewEnd := mid + (overviewStep / 2)

		// Ensure overviewStart and overviewEnd are within boundaries
		if overviewStart <= low {
			overviewStart = low
			lastStep = true
		}
		if overviewEnd >= high {
			overviewEnd = high
			lastStep = true
		}

		// Request overview for the calculated range
		results, err := conn.Overview(overviewStart, overviewEnd)
		if err != nil {
			return 0, time.Time{}, fmt.Errorf("overview request failed for range %d-%d: %w", overviewStart, overviewEnd, err)
		}

		// Handle empty results - messages might be deleted in this range
		if len(results) == 0 {
			// No messages available in this range
			// If this was the last step, we can't go further
			if lastStep {
				if (searchForFirst && overviewEnd == high) || (!searchForFirst && overviewStart == low) {
					return 0, time.Time{}, errors.New(boundaryError)
				} else {
					if lastResult != (messageResult{}) {
						return lastResult.MessageNumber, lastResult.Date, nil
					} else {
						return 0, time.Time{}, errors.New(noResultError)
					}
				}
			}
			// Update search bounds based on direction
			if direction == "up" {
				low = overviewEnd + 1
			} else {
				high = overviewStart - 1
			}
			bar.Set(iterationCount)
			continue
		}

		// Save appropriate result for potential use in last step
		if searchForFirst {
			lastResult = messageResult{
				MessageNumber: results[0].MessageNumber,
				Date:          results[0].Date,
			}
		} else {
			lastResult = messageResult{
				MessageNumber: results[len(results)-1].MessageNumber,
				Date:          results[len(results)-1].Date,
			}
		}

		// If this is the last step, scan the results directly
		if lastStep {
			result, found := scanResultsForTarget(results, targetDate, searchForFirst)
			if found {
				return result.MessageNumber, result.Date, nil
			}

			if (searchForFirst && overviewEnd == high) || (!searchForFirst && overviewStart == low) {
				return 0, time.Time{}, errors.New(boundaryError)
			} else {
				if searchForFirst {
					return results[0].MessageNumber, results[0].Date, nil
				} else {
					return results[len(results)-1].MessageNumber, results[len(results)-1].Date, nil
				}
			}
		}

		// Check only first and last message to determine search direction
		firstResult := results[0]
		lastResult := results[len(results)-1]

		// Update search bounds based on comparison with target date
		if searchForFirst {
			if targetDate < firstResult.Date.Unix() {
				high = firstResult.MessageNumber - 1
				direction = "down"
				bar.Set(iterationCount)
				continue
			}
			if targetDate > lastResult.Date.Unix() {
				low = lastResult.MessageNumber + 1
				direction = "up"
				bar.Set(iterationCount)
				continue
			}
		} else {
			if targetDate > lastResult.Date.Unix() {
				low = lastResult.MessageNumber + 1
				direction = "up"
				bar.Set(iterationCount)
				continue
			}
			if targetDate < firstResult.Date.Unix() {
				high = firstResult.MessageNumber - 1
				direction = "down"
				bar.Set(iterationCount)
				continue
			}
		}

		// Target date is between first and last message - scan the range directly
		result, found := scanResultsForTarget(results, targetDate, searchForFirst)
		if found {
			return result.MessageNumber, result.Date, nil
		}

		// Fallback - continue search
		Log.Warn("Unexpected condition encountered during search; continuing binary search.")
		if direction == "up" {
			low = overviewEnd + 1
		} else {
			high = overviewStart - 1
		}
		bar.Set(iterationCount)
	}

	return 0, time.Time{}, errors.New(noResultError)
}

// Helper function to scan results for target date
func scanResultsForTarget(results []nntp.MessageOverview, targetDate int64, searchForFirst bool) (nntp.MessageOverview, bool) {
	if searchForFirst {
		// Scan forward for first message >= targetDate
		for _, result := range results {
			if result.Date.Unix() >= targetDate {
				return result, true
			}
		}
	} else {
		// Scan backward for last message <= targetDate
		for i := len(results) - 1; i >= 0; i-- {
			result := results[i]
			if result.Date.Unix() <= targetDate {
				return result, true
			}
		}
	}
	return nntp.MessageOverview{}, false
}

// Calculate maximum possible iterations for binary search
func calcMaxIterations(searchSpace int) int {
	if searchSpace <= 0 {
		return 1
	}
	// Binary search worst case is log2(n)
	maxIterations := 0
	for searchSpace > overviewStep { // The window size
		searchSpace = searchSpace / 2
		maxIterations++
	}
	return maxIterations
}

func FormatNumberWithApostrophe(n int) string {
	str := strconv.Itoa(n)

	// Handle negative numbers
	negative := ""
	if str[0] == '-' {
		negative = "-"
		str = str[1:]
	}

	// Process from right to left
	var parts []string
	for len(str) > 3 {
		parts = append([]string{str[len(str)-3:]}, parts...)
		str = str[:len(str)-3]
	}
	if len(str) > 0 {
		parts = append([]string{str}, parts...)
	}

	return negative + strings.Join(parts, "'")
}
