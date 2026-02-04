package main

import (
	"bufio"
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

var overviewTimeout = 5 * time.Second
var overviewRetries = 3

var directsearchHits = make(map[string]map[string]nzbparser.NzbFile)
var directsearchCounter uint64
var overviewReaderCounter uint64
var linesCounter uint64
var startDate int64
var endDate int64
var mutex = sync.Mutex{}
var overviewLines = make(chan string, 10000)

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
	if conf.Directsearch.Step == 0 {
		conf.Directsearch.Step = 20000
	}
	if conf.Directsearch.OverviewTimeout == 0 {
		conf.Directsearch.OverviewTimeout = 5
	}
	if conf.Directsearch.OverviewRetries == 0 {
		conf.Directsearch.OverviewRetries = 3
	}
	overviewTimeout = time.Duration(conf.Directsearch.OverviewTimeout) * time.Second
	overviewRetries = conf.Directsearch.OverviewRetries
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

	var searchesCtx = context.Background()
	startDate = args.UnixDate - int64(conf.Directsearch.Hours*60*60)
	endDate = args.UnixDate + int64(60*60*conf.Directsearch.ForwardHours)
	if !args.IsTimestamp {
		endDate += 60 * 60 * 24
	}
	var firstMessageID, lastMessageID int
	var messageDate time.Time
	var err error

	// scann for first message
	Log.Info("Scanning from %s to %s", time.Unix(startDate, 0).Format("02.01.2006 15:04:05 MST"), time.Unix(endDate, 0).Format("02.01.2006 15:04:05 MST"))
	firstMessageID, messageDate, err = getFirstMessageNumberFromGroup(group, startDate, endDate, searchesCtx)
	if err != nil {
		return fmt.Errorf("error while scanning group '%s' for the first message: %v", group, err)
	}
	Log.Info("Found first message number: %v / Date: %v", FormatNumberWithApostrophe(firstMessageID), messageDate.Local().Format("02.01.2006 15:04:05 MST"))

	// scan for last message
	lastMessageID, messageDate, err = getLastMessageNumberFromGroup(group, endDate, startDate, searchesCtx)
	if err != nil {
		return fmt.Errorf("error while scanning group '%s' for the last message: %v", group, err)
	}
	if firstMessageID >= lastMessageID {
		return errors.New("no messages found within search range")
	}
	Log.Info("Found last message number:  %v / Date: %v", FormatNumberWithApostrophe(lastMessageID), messageDate.Local().Format("02.01.2006 15:04:05 MST"))

	// start searching messages
	Log.Info("Scanning messages %v to %v (%v messages in total)", FormatNumberWithApostrophe(firstMessageID), FormatNumberWithApostrophe(lastMessageID), FormatNumberWithApostrophe(lastMessageID-firstMessageID+1))

	// setup progress bar
	directsearchCounter = 0
	progressbarOptions := []progressbar.Option{
		progressbar.OptionSetDescription("   Scanning ... "),
		progressbar.OptionSetRenderBlankState(true),
		progressbar.OptionThrottle(time.Millisecond * 100),
		progressbar.OptionShowElapsedTimeOnFinish(),
		progressbar.OptionUseANSICodes(conf.Directsearch.UseANSICodes),
	}
	if conf.Directsearch.ShowCounter {
		progressbarOptions = append(progressbarOptions, progressbar.OptionShowCount())
	}
	bar := progressbar.NewOptions(lastMessageID-firstMessageID, progressbarOptions...)
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

	// initialize wait groups
	searches := sync.WaitGroup{}
	scanners := sync.WaitGroup{}

	// start line scanners
	Log.Debug("Starting line scanners.")
	for range 5 {
		scanners.Go(func() {
			lineScanner(searchesCtx, group)
		})
	}

	var first, last int
	last = firstMessageID - 1

	// start overview readers
	for last < lastMessageID {
		first = last + 1
		last = min(first+conf.Directsearch.Step, lastMessageID)
		startOverviewSearch(searchesCtx, &searches, group, first, last, 0)
	}

	searches.Wait()
	Log.Debug("All %s overview readers completed with %s messages read.", FormatNumberWithApostrophe(int(overviewReaderCounter)), FormatNumberWithApostrophe(int(directsearchCounter)))
	close(overviewLines)
	scanners.Wait()
	Log.Debug("All line scanners completed with %s lines processed.", FormatNumberWithApostrophe(int(linesCounter)))
	bar.Finish()
	fmt.Println()
	if int(maxConn) < conf.Directsearch.Connections {
		fmt.Println()
		Log.Info("%s", yellow(fmt.Sprintf("Maximum connections used: %d (configured: %d)", maxConn, conf.Directsearch.Connections)))
		Log.Info("%s", yellow("Consider increasing the 'step' setting to speed up the search."))
		fmt.Println()
	}
	return nil
}

func startOverviewSearch(searchesCtx context.Context, searches *sync.WaitGroup, group string, first, last, restart int) error {
	conn, _, _, err := switchToGroup(group)
	if err != nil {
		pool.Put(conn)
		return fmt.Errorf("unable to connect to the usenet server: %v", err)
	}
	reader, err := conn.OverviewReader(first, last)
	if err != nil {
		pool.Put(conn)
		return fmt.Errorf("error retrieving message overview from the usenet server while searching in group '%s': %v", group, err)
	}
	// Wrap the reader with a larger buffer to handle long overview lines
	// Some NNTP servers can return very long overview lines (>4KB default buffer)
	largeReader := bufio.NewReaderSize(reader, 128*1024) // 128KB buffer
	searches.Go(func() {
		overviewReader(searchesCtx, searches, conn, group, largeReader, first, last, restart)
	})
	return nil
}

func switchToGroup(group string) (*nntpPool.NNTPConn, int, int, error) {
	conn, err := pool.Get(context.TODO())
	if err != nil {
		return nil, 0, 0, err
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
	conn, firstMessageNumber, lastMessageNumber, err := switchToGroup(group)
	if err != nil {
		return 0, time.Time{}, fmt.Errorf("failed to get group info: %w", err)
	}
	pool.Put(conn)

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
		Log.Info("Searching for first message on or after %s", time.Unix(targetDate, 0).Format("02.01.2006 15:04:05 MST"))
		description = "   Scanning ... "
		direction = "up"
		noResultError = "no messages found on or after the specified start date"
		boundaryError = "the newest message in the group is older than the specified start date"
	} else {
		Log.Info("Searching for last message on or before %s", time.Unix(targetDate, 0).Format("02.01.2006 15:04:05 MST"))
		description = "   Scanning ... "
		direction = "down"
		noResultError = "no messages found on or before the specified end date"
		boundaryError = "the oldest message in the group is newer than the specified end date"
	}

	bar := progressbar.NewOptions(maxIterations,
		progressbar.OptionSetDescription(description),
		progressbar.OptionSetRenderBlankState(true),
		progressbar.OptionThrottle(time.Millisecond*100),
		progressbar.OptionShowElapsedTimeOnFinish(),
		progressbar.OptionUseANSICodes(conf.Directsearch.UseANSICodes),
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
		results := []nntp.MessageOverview{}
		for i := range 3 {
			results, err = findMessageByDateOverview(overviewStart, overviewEnd, group)
			if err == nil {
				break
			} else if i == 2 {
				return 0, time.Time{}, fmt.Errorf("overview request failed after 3 attempts for range %d-%d: %w", overviewStart, overviewEnd, err)
			}
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

func findMessageByDateOverview(begin, end int, group string) ([]nntp.MessageOverview, error) {
	Log.Debug("Overview request started for range %d - %d", begin, end)
	ctx, cancel := context.WithCancel(context.Background())
	messagesChannel := make(chan []nntp.MessageOverview, 1)
	overviews := sync.WaitGroup{}
	var lastError atomic.Value
	conns := make([]*nntpPool.NNTPConn, 4)
	for i := range 4 {
		overviews.Go(func() {
			var err error
			conns[i], _, _, err = switchToGroup(group)
			if err != nil {
				select {
				case <-ctx.Done():
					return
				default:
					lastError.Store(err)
					return
				}
			}
			defer pool.Put(conns[i])
			select {
			case <-ctx.Done():
				return
			default:
			}
			messages, err := conns[i].Overview(begin, end)
			if err != nil {
				select {
				case <-ctx.Done():
					return
				default:
					lastError.Store(err)
					return
				}
			}
			select {
			case messagesChannel <- messages:
			default:
			}
		})
	}
	go func() {
		overviews.Wait()
		close(messagesChannel)
		Log.Debug("Overview request completed for range %d - %d", begin, end)
	}()
	select {
	case messageOverview, ok := <-messagesChannel:
		cancel()
		if !ok {
			if err := lastError.Load(); err != nil {
				return nil, err.(error)
			}
			return nil, fmt.Errorf("all overview requests failed")
		}
		return messageOverview, nil
	case <-time.After(overviewTimeout):
		cancel()
		for i := range 4 {
			if conns[i] != nil {
				conns[i].Close()
			}
		}
		Log.Debug("Overview request timed out for range %d - %d", begin, end)
		return nil, fmt.Errorf("overview request timed out for range %d - %d", begin, end)
	}
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

func overviewReader(ctx context.Context, searches *sync.WaitGroup, conn *nntpPool.NNTPConn, group string, reader *bufio.Reader, first, last, restart int) {
	atomic.AddUint64(&overviewReaderCounter, 1)
	for i := 1; ; i++ {
		select {
		case <-ctx.Done():
			conn.Close()
			pool.Put(conn)
			return // Error somewhere, terminate
		default: // required, otherwise it will block
		}
		lineChan := make(chan string, 1)
		errorChan := make(chan error, 1)
		go func(r *bufio.Reader) {
			line, err := r.ReadString('\n')
			if err != nil {
				errorChan <- err
				lineChan <- ""
			} else {
				lineChan <- strings.TrimSpace(line)
			}
		}(reader)
		select {
		case <-ctx.Done():
			conn.Close()
			pool.Put(conn)
			return // Error somewhere, terminate
		case line := <-lineChan:
			if line == "" {
				conn.Close()
				pool.Put(conn)
				select {
				case err := <-errorChan:
					Log.Debug("Overview reader error at line %d for range %d - %d: %v", i, first, last, err)
				default:
					Log.Debug("Overview reader error at line %d for range %d - %d: unknown error", i, first, last)
				}
				maybeRestartOverviewSearch(i, first, last, ctx, searches, group, restart)
				return
			}
			if line == "." {
				pool.Put(conn)
				return
			}
			if strings.Contains(strings.ToLower(line), strings.ToLower(args.Header)) {
				overviewLines <- line
			} else {
				atomic.AddUint64(&directsearchCounter, 1)
			}
		case <-time.After(overviewTimeout):
			conn.Close()
			pool.Put(conn)
			Log.Debug("Overview reader timeout at line %d for range %d - %d", i, first, last)
			maybeRestartOverviewSearch(i, first, last, ctx, searches, group, restart)
			return
		}
	}
}

func maybeRestartOverviewSearch(lineNumber, first, last int, ctx context.Context, searches *sync.WaitGroup, group string, restart int) {
	if first+lineNumber-1 > last {
		return
	}
	if restart >= overviewRetries {
		Log.Error("Overview reader for range %d - %d failed after %d retries.", first, last, restart)
		return
	}
	if lineNumber != 1 {
		restart = 0
	} else {
		restart++
	}
	Log.Debug("Restarting overview reader for range %d - %d", first+lineNumber-1, last)
	err := startOverviewSearch(ctx, searches, group, first+lineNumber-1, last, restart)
	if err != nil {
		Log.Error("Failed to restart overview reader: %v", err)
	}
}

func lineScanner(ctx context.Context, group string) {
	for {
		select {
		case <-ctx.Done():
			return
		case line, ok := <-overviewLines:
			if !ok {
				return
			}
			atomic.AddUint64(&linesCounter, 1)
			overview, err := nntp.ParseOverviewLine(line)
			if err != nil {
				Log.Debug("Failed to parse message line \"%s\": %v", line, err)
				continue
			}
			currentDate := overview.Date.Unix()
			if currentDate > endDate || currentDate < startDate {
				continue
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
	}
}
