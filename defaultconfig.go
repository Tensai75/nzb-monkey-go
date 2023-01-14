package main

import "strings"

func defaultConfig() string {
	return strings.Trim(`
[GENERAL]
# Target for handling nzb files - EXECUTE, SABNZBD, NZBGET or SYNOLOGYDLS
target = "EXECUTE"
# Let the monkey choose a category. Values are: off, auto, manual
categorize = "off"
# seconds to wait befor ending/closing the window after success
success_wait_time = 3
# seconds to wait befor ending/closing the window after an error
error_wait_time = 10

[EXECUTE]
# Extend password to filename {{password}}
passtofile = true
# Copy password to clipboard
passtoclipboard = false
# Path to save nzb files
nzbsavepath = "./nzb"
# Don't execute default programm for .nzb
dontexecute = true
# Delete old NZB files from nzbsavepath
clean_up_enable = false
# NZB files older than x days will be deleted
clean_up_max_age = 2

[SABNZBD]
# SABnzbd Hostname
host = "localhost"
# SABnzbd Port
port = 8080
# Use https
ssl = false
# NZB Key
nzbkey = ""
# Basic Auth Username
basicauth_username = ""
# Basic Auth Password
basicauth_password = ""
# Basepath
basepath = ""
# Category
category = ""
# Add the nzb paused to the queue
addpaused = false

[NZBGET]
# NZBGet Host
host = "localhost"
# NZBGet Port
port = 6789
# Use https
ssl = false
# NZBGet Username
user = ""
# NZBGet Password
pass = ""
# Basepath
basepath = ""
# NZBGet Category
category = ""
# Add the nzb paused to the queue
addpaused = false

[SYNOLOGYDLS]
# Downloadstation Host
host = "localhost"
# Downloadstation Port
port = 5000
# Use https
ssl = false
# Downloadstation Username
user = ""
# Downloadstation Password
pass = ""
# Basepath
basepath = ""

[NZBCheck]
# Don't skip failed nzb
skip_failed = true
# Max missing failed segments
max_missing_segments_percent = 2
# Max missing failed files
max_missing_files = 2
# Use always all Searchengines to find the best NZB
best_nzb = true

[CATEGORIZER]
# Place your category and you regex here
# Please uncomment the following lines
# series = "(s\d+e\d+|s\d+ complete)"
# movies = "(x264|xvid|bluray|720p|1080p|untouched)"

[SEARCHENGINES]
# Set values between 0-9
# 0 = disabled; 1-9 = enabled; 1-9 are also the order in which the search engines are used
# More than 1 server with the same order number is allowed
# Enable NZBIndex
nzbindex =  1
# Enable NZBKing
nzbking =  2
# Enable Binsearch
binsearch =  3
# Enable Binsearch - Alternative Server
binsearch_alternative = 3
# Enable nzb direct search (settings for the nzb direct search required)
directsearch = 4

# Settings for the nzb direct search
[DIRECTSEARCH]
# Your usenet server host name
host = "news-eu.newshosting.com"
# Your usenet server port number
port = 119
# Use SSL
ssl = false
# Your usenet account username
username = ""
# Your usenet account password
password = ""
# Maximum allowed connections to your usenet server (default = 20)
connections = 20
# Number of days to search back from the provided date (default = 2)
days = 2
# Number of parallel scans (default = 50)
scans = 50
# Number of articles to load per scan (default = 20000)
Step = 20000   
`, "\n")
}
