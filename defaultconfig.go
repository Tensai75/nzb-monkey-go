package main

import "strings"

func defaultConfig() string {
	return strings.Trim(`
[GENERAL]
# Target for handling nzb files - EXECUTE, SABNZBD, NZBGET or SYNOLOGYDLS
# Multiple targets can be separated by commas, e.g. "EXECUTE,SABNZBD"
target = "EXECUTE"
# Let the monkey choose a category. Values are: off, auto, manual
categorize = "off"
# Seconds to wait befor ending/closing the window after success
success_wait_time = 3
# Seconds to wait befor ending/closing the window after an error
error_wait_time = 10
# Write debug log to logfile.txt on windows/osx (same dir as nzb-monkey-go) or /tmp/nzb-monkey-go.log on linux
debug = false

[EXECUTE]
# Extend password to filename {{password}}
passtofile = true
# Copy password to clipboard
passtoclipboard = false
# Path to save nzb files
# Either an absolute path or a path relative to the user's home directory
nzbsavepath = "./Downloads/nzb"
# Use category subfolders
category_folder = false
# Don't execute default programm for .nzb
dontexecute = true
# Save nzb files as compressed zip files
save_as_zip = false
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
# skip SSL security checks (e.g. for self signed certificates)
skip_check = false
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
# Add compression on upload, either "none" or "zip"
compression = "none"

[NZBGET]
# NZBGet Host
host = "localhost"
# NZBGet Port
port = 6789
# Use https
ssl = false
# skip SSL security checks (e.g. for self signed certificates)
skip_check = false
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
# skip SSL security checks (e.g. for self signed certificates)
skip_check = false
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
max_missing_segments_percent = 1
# Max missing failed files
max_missing_files = 1
# Use always all Searchengines to find the best NZB but stop once a NZB with 100% completeness has been found.
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
# Enable NZBIndex Beta
nzbindex_beta =  2
# Enable NZBKing
nzbking =  3
# Enable Binsearch
binsearch =  4
# Enable nzb direct search (settings for the nzb direct search required)
directsearch = 5

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
# Number of hours to search backward from the provided date (default = 12)
hours = 12
# Number of hours to search forward from the provided date (default = 12)
forward_hours = 12
# Number of parallel scans (default = 50)
scans = 50
# Number of articles to load per scan (default = 20000)
step = 20000
# Skip direct search when using best_nzb and a good NZB file has already been found
skip = true
# Search only in the first group if several groups are provided
# (the chance to get different results in different groups is virtually 0)
first_group_only = false
`, "\n")
}
