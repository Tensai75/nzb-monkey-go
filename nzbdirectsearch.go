package main

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"time"

	"github.com/Tensai75/fslock"
	"github.com/Tensai75/nntpDirectSearch"
	"github.com/Tensai75/nzbparser"
	progressbar "github.com/schollz/progressbar/v3"
)

var directSearch *nntpDirectSearch.DirectSearch
var err error
var startDate int64
var endDate int64
var searchInGroupError error
var totalPeakMessagesPerSecond uint64
var totalPeakBytesPerSecond uint64

var FormatNumberWithApostrophe = func(n uint) string {
	return nntpDirectSearch.FormatNumberWithApostrophe(n)
}

func nzbdirectsearch(engine SearchEngine, name string) error {

	if len(results) > 0 && conf.Directsearch.Skip {
		Log.Info("Results already available. Skipping search based on config settings.")
		return nil
	}

	// validate config and arguments
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

	// set start and end date for search
	startDate = args.UnixDate - int64(conf.Directsearch.Hours*60*60)
	endDate = args.UnixDate + int64(60*60*conf.Directsearch.ForwardHours)
	if !args.IsTimestamp {
		endDate += 60 * 60 * 24
	}

	// acquire lock if configured to allow only one instance of the direct search
	if conf.Directsearch.OneInstanceOnly {
		lock, err := acquireLock()
		if err != nil {
			return fmt.Errorf("failed to acquire lock: %v", err)
		}
		defer lock.Unlock()
	}

	// initialize nntp pool
	if err := initNntpPool(); err != nil {
		return err
	} else {
		defer pool.Close()
	}

	// initialize direct search
	directSearchCtx, directSearchCtxCancel := context.WithCancel(context.Background())
	defer directSearchCtxCancel()
	directSearch, err = nntpDirectSearch.New(pool, directSearchCtx)
	if err != nil {
		return err
	}
	directSearchConfig := nntpDirectSearch.DirectSearchConfig{
		Connections:     uint(conf.Directsearch.Connections),
		Step:            uint(conf.Directsearch.Step),
		OverviewRetries: uint(conf.Directsearch.OverviewRetries),
		OverviewTimeout: uint(conf.Directsearch.OverviewTimeout),
	}
	err := directSearch.SetConfig(directSearchConfig)
	if err != nil {
		return fmt.Errorf("failed to set direct search config: %v", err)
	}

	// start debug log listener
	go func() {
		for {
			select {
			case <-directSearchCtx.Done():
				return
			case debugLog := <-directSearch.Log:
				Log.Debug("NNTPDirectSearch: %s", debugLog)
			}
		}
	}()

	// iterate over groups
	for i, group := range args.Groups {
		if i > 0 && conf.Directsearch.FirstGroupOnly && searchInGroupError == nil {
			Log.Info("Skipping other groups based on config settings.")
			return nil
		}
		Log.Info("Searching in group '%s' ...", group)
		searchInGroupError = nil

		nzbFiles, searchInGroupError := searchInGroup(group)
		if searchInGroupError != nil {
			Log.Error(searchInGroupError.Error())
			continue
		}
		if len(nzbFiles) == 0 {
			Log.Warn("No result found in group '%s'", group)
			continue
		}
		for _, nzb := range nzbFiles {
			nzbparser.MakeUnique(nzb)
			nzbparser.ScanNzbFile(nzb)
			sort.Sort(nzb.Files)
			for id := range nzb.Files {
				sort.Sort(nzb.Files[id].Segments)
			}
			processResult(nzb, name)
		}
	}
	return nil
}

func searchInGroup(group string) ([]*nzbparser.Nzb, error) {

	// switch to group
	err = directSearch.SwitchToGroup(group)
	if err != nil {
		return nil, fmt.Errorf("failed to switch to group '%s': %v", group, err)
	}

	// scan for first and last message
	Log.Info("Scanning for first and last message from %s to %s", time.Unix(startDate, 0).Format("02.01.2006 15:04:05 MST"), time.Unix(endDate, 0).Format("02.01.2006 15:04:05 MST"))
	boundaries, err := scanForBoundaries()
	if err != nil {
		return nil, fmt.Errorf("failed to scan for first and last message: %v", err)
	}
	Log.Info("Found first message ID: %s - Date: %s", FormatNumberWithApostrophe(boundaries.FirstMessage.MessageID), boundaries.FirstMessage.Date.Local().Format("02.01.2006 15:04:05 MST"))
	Log.Info("Found last message ID:  %s - Date: %s", FormatNumberWithApostrophe(boundaries.LastMessage.MessageID), boundaries.LastMessage.Date.Local().Format("02.01.2006 15:04:05 MST"))

	// start peak rate measurement
	Log.Debug("Starting peak rate measurement.")
	ticker := time.NewTicker(1000 * time.Millisecond)
	totalSearchRange := boundaries.LastMessage.MessageID - boundaries.FirstMessage.MessageID + 1
	Log.Debug("Total search range: %s messages", FormatNumberWithApostrophe(totalSearchRange))
	peakMessagesPerSecond := uint64(0)
	peakBytesPerSecond := uint64(0)
	go measurePeakRates(ticker, &peakMessagesPerSecond, &peakBytesPerSecond)

	// scan messages for header
	Log.Info("Scanning messages %v to %v (%v messages in total)", FormatNumberWithApostrophe(boundaries.FirstMessage.MessageID), FormatNumberWithApostrophe(boundaries.LastMessage.MessageID), FormatNumberWithApostrophe(totalSearchRange))
	startTime := time.Now()
	nzbFiles, err := scanForHeader(boundaries)
	if err != nil {
		return nil, fmt.Errorf("failed to scan messages: %v", err)
	}

	// display rates
	ticker.Stop()
	// calculate duration
	duration := time.Since(startTime)
	formattedDuration := fmt.Sprintf("%02dm %02ds %03dms", int(duration/time.Minute), int((duration%time.Minute)/time.Second), int((duration%time.Second)/time.Millisecond))
	// get lines read
	linesRead := directSearch.GetLinesRead()
	formattedLinesRead := FormatNumberWithApostrophe(uint(linesRead))
	// calculate average messages per second
	averageMessagesPerSecond := uint64((float64(linesRead) / float64(duration.Milliseconds()) * 1000))
	formattedAverageMessagesPerSecond := FormatNumberWithApostrophe(uint(averageMessagesPerSecond))
	// calculate average Mbit/s
	averageBytesPerSecond := uint64((float64(directSearch.GetBytesRead()) / float64(duration.Milliseconds())) * 1000)
	averageMbitPerSecond := float64(averageBytesPerSecond) * 8 / 1000000
	// calculate peake messages per second
	var formatedPeakMessagesPerSecond string
	if peakMessagesPerSecond < averageMessagesPerSecond {
		formatedPeakMessagesPerSecond = formattedAverageMessagesPerSecond
	} else {
		formatedPeakMessagesPerSecond = FormatNumberWithApostrophe(uint(peakMessagesPerSecond))
	}
	// calculate peak Mbit/s
	var peakMbitPerSecond float64
	if peakBytesPerSecond < averageBytesPerSecond {
		peakMbitPerSecond = float64(averageBytesPerSecond) * 8 / 1000000
	} else {
		peakMbitPerSecond = float64(peakBytesPerSecond) * 8 / 1000000
	}
	Log.Info("Scan completed in %s with %s messages processed", formattedDuration, formattedLinesRead)
	Log.Info("Average rate: %s messages/s / %.2f Mbit/s", formattedAverageMessagesPerSecond, averageMbitPerSecond)
	Log.Info("Peak rate:    %s messages/s / %.2f Mbit/s", formatedPeakMessagesPerSecond, peakMbitPerSecond)
	fmt.Println()

	return nzbFiles, nil
}

func scanForBoundaries() (nntpDirectSearch.BoundariesScannerResult, error) {

	maxIterations := directSearch.MaxBoundariesScannerIterations

	// setup progress bar
	bar := progressbar.NewOptions(int(maxIterations),
		progressbar.OptionSetDescription("   Scanning ... "),
		progressbar.OptionSetRenderBlankState(true),
		progressbar.OptionThrottle(time.Millisecond*100),
		progressbar.OptionShowElapsedTimeOnFinish(),
		progressbar.OptionUseANSICodes(conf.Directsearch.UseANSICodes),
	)
	iterationFunc := func() {
		bar.Add(1)
	}

	// scan for boundaries
	var boundaries nntpDirectSearch.BoundariesScannerResult
	boundaries, err = directSearch.BoundariesScanner(time.Unix(startDate, 0), time.Unix(endDate, 0), iterationFunc)
	bar.Finish()
	fmt.Println()
	if err != nil {
		return boundaries, err
	}

	return boundaries, nil
}

func scanForHeader(boundaries nntpDirectSearch.BoundariesScannerResult) ([]*nzbparser.Nzb, error) {

	firstMessageID := boundaries.FirstMessage.MessageID
	lastMessageID := boundaries.LastMessage.MessageID
	maxIterations := int(lastMessageID - firstMessageID + 1)

	// setup progress bar
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
	bar := progressbar.NewOptions(maxIterations, progressbarOptions...)
	iterationFunc := func() {
		bar.Add(1)
	}

	// scan for header
	var nzbFiles []*nzbparser.Nzb
	nzbFiles, err = directSearch.MessageScanner(args.Header, firstMessageID, lastMessageID, iterationFunc)
	bar.ChangeMax64(int64(directSearch.GetLinesRead()))
	bar.Finish()
	fmt.Println()
	if err != nil {
		return nzbFiles, err
	}

	return nzbFiles, nil
}

func measurePeakRates(ticker *time.Ticker, peakMessagesPerSecond *uint64, peakBytesPerSecond *uint64) {
	var lastMessages uint64
	var lastBytes uint64
	for range ticker.C {
		currentMessages := directSearch.GetLinesRead()
		currentBytes := directSearch.GetBytesRead()
		messagesThisSecond := currentMessages - lastMessages
		bytesThisSecond := currentBytes - lastBytes
		if messagesThisSecond > *peakMessagesPerSecond {
			*peakMessagesPerSecond = messagesThisSecond
			if totalPeakMessagesPerSecond < *peakMessagesPerSecond {
				totalPeakMessagesPerSecond = *peakMessagesPerSecond
			}
		}
		if bytesThisSecond > *peakBytesPerSecond {
			*peakBytesPerSecond = bytesThisSecond
			if totalPeakBytesPerSecond < *peakBytesPerSecond {
				totalPeakBytesPerSecond = *peakBytesPerSecond
			}
		}
		lastMessages = currentMessages
		lastBytes = currentBytes
	}
}

func acquireLock() (*fslock.Lock, error) {
	lockFilePath := fmt.Sprintf("%s/directSearch.lock", tempPath)
	lock := fslock.New(lockFilePath)
	err := lock.TryLock()
	if err == nil {
		return lock, nil
	}
	if !errors.Is(err, fslock.ErrLocked) {
		return nil, err
	}
	Log.Warn("Another instance of the direct search is already running. Waiting for lock to be released...")
	err = lock.Lock()
	if err != nil {
		return nil, err
	}
	return lock, nil
}
