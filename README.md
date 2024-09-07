[![Release Workflow](https://github.com/Tensai75/nzb-monkey-go/actions/workflows/build_and_publish.yml/badge.svg?event=release)](https://github.com/Tensai75/nzb-monkey-go/actions/workflows/build_and_publish.yml)
[![Latest Release)](https://img.shields.io/github/v/release/Tensai75/nzb-monkey-go?logo=github)](https://github.com/Tensai75/nzb-monkey-go/releases/latest)

# NZB Monkey Go

A completely rewritten version in Golang of the [NZB Monkey](https://github.com/nzblnk/nzb-monkey), the reference implementation written in Python of how to handle a [NZBLNK](https://nzblnk.github.io/)-URI.
NZB Monkey Go also includes the functionality of the [NZBSearcher](https://github.com/Tensai75/nzbsearcher) so besides in the nzb search engines it can also search directly on the news server and create the NZB file from the article headers.

## Running the Monkey

Running the NZB Monkey Go is virtually identical to running the original NZB Monkey.

Differences are that in the configuration file, 'directsearch' can be enabled as an additional search engine in the section 'SEARCHENGINES' and an additional section 'DIRECTSEARCH' is included, with additional configuration options required for the direct search on the news server to work.

See the [NZB Monkey Go Wiki](https://github.com/Tensai75/nzb-monkey-go/wiki) for further information regarding installation and configuration.

Note that directsearch requires the NZBLNK to include the news groups (parameter 'g') and an additional parameter for the date ('d') when the post was uploaded to the Usenet.
Parameter 'd' can be provided either in the format 'DD.MM.YYYY' or as an unix timestamp.

![Monkey-Gif](https://github.com/Tensai75/nzb-monkey-go/raw/main/resources/nzbmonkey-go.gif)

## Windows and Linux binaries

The binaries are available on the [release page](https://github.com/Tensai75/nzb-monkey-go/releases).

For Arch Linux / Manjao user a AUR package is kindly provided by [nzb-tuxxx](https://github.com/nzb-tuxxx): https://aur.archlinux.org/packages/nzb-monkey-go-bin

### macOS Support

The macOS binaries are provided on the [release page](https://github.com/Tensai75/nzb-monkey-go/releases) as well, however the program cannot register the nzblnk URI protocol itself.
On the NZB Monkey github some solutions on how to register the protocol have been [discussed](https://github.com/nzblnk/nzb-monkey/issues/20) which should work for the NZB Monkey Go as well.

Please also note that the macOS binaries are not signed.

## Contribution

Feel free to send pull requests.

## Change log
#### v0.1.16
- fix: don't panic on null values in json responses
- new feature: zip compression support for NZB file uploads to SABnzbd (thanks @BearsWithPasta)

#### v0.1.15
- adding new search engine nzbindex_beta

#### v0.1.14
- clean search string for NZBKing & url encode search string (should fix some issues with NZBKing)
- use cyan instead of blue
- use nntpPool instead of nntp
- use strings.Contains instead of regexp for directsearch
- update subjectparser to v0.1.1
- fix: actually set defaults as given in config comments (thanks @wilriker)
- complete rewrite of the start and end message search functions

#### v0.1.13
- fix for directsearch not working on some usenet servers which actually return the correct error response for unknown commands
- better color output support for windows
- update of all dependencies
- 32bit builds removed from build script

#### v0.1.12
- new option to have several targets, seperated by commas
- new option for category subfolders when using EXECUTE
- update of cleanup routine for better error handling and taking into account the category subfolders
- fix for success messages not written to the log file

#### v0.1.11
- update of binsearch.info urls (new API)
- removal of binsearch alternative servers (no longer available)
- fix for panic if an NZB file with no file entries is returned
- fix for panic if no searchengine is set in the configuration file
- new behavior: don't exit in case of unknown values for the searchengines

#### v0.1.10
- fix for wrong error "start date of search range is newer than latest message" [fixes https://github.com/Tensai75/nzb-monkey-go/issues/20]
- new behavior: the program stops searching after a 100% complete NZB file has been found if best_nzb = true (Thanks @[wilriker](https://github.com/wilriker)). This makes perfectly sense because the first NZB who was found to be 100% complete would be used anyway also if all further search engines were searched as well.

#### v0.1.9
- fix for Basic Auth not working after code refactoring

#### v0.1.8
- fix for wrong category assignment [fixes https://github.com/Tensai75/nzb-monkey-go/issues/18]
- fix for the "not well-formed (invalid token)" error when the inner XML text contains special HTML characters
- fix for the wrong search interval and other problems when searching for first and last message number
- additional output of information during scanning (first/last found message number and message number scan range)
- addition of the time zone when displaying date and time
- minor changes

#### v0.1.7
- fix for panic upon connection errors and various problems with connection pool (Thanks @[wilriker](https://github.com/wilriker)) [fixes https://github.com/Tensai75/nzb-monkey-go/issues/14]
- minor code refactoring and clean-ups (Thanks @[wilriker](https://github.com/wilriker))
- addition of a GitHub Action for automatic builds for several architectures, including debian packages for linux and darwin builds (Thanks @[reloxx13](https://github.com/reloxx13))

#### v0.1.6
- possible bug fix for the occasional "too many connections" errors
- bug fix for the progress bar not progressing for further groups after first group was scanned
- improved indexing of message subjects (improved hashing of file names for better differentiation)

#### v0.1.5
- Linux version: the config file is moved to `~/.config/nzb-monkey-go.conf` for better compatibility with packed distribution (an existing config.txt file in the application directory will be automatically moved to the new location during the first execution)
- Linux version: if debug is enabled, the log file is now created as `/tmp/nzb-monkey-go.log` for better compatibility with packed distribution
- All versions: the `nzbsavepath` in the `[EXECUTE]` section of the config file must now be an absolute path or a relative path to the user's home directory
- All versions: default path for `nzbsavepath` in the `[EXECUTE]` section of the config file changed to `./Downloads/nzb`.
- some minor bug fixes and typo fixes (thanks @nzb-tuxxx)
- executables are now named `nzb-monkey-go(.exe)` (previously `nzbmonkey-go(.exe)`)

#### v0.1.4
- Fix: progress bar not updating when scanning for messages

#### v0.1.3
- Fix: in some circumstances when more the one group was specified, only the first group was searched
- The user is now informed if a NZB file is skipped due to incompleteness
- NZBLNKs are now checked whether several groups are specified within a single 'g' parameter (to account for malformed NZBLNKs, e.g. where groups are separated by a space or a comma)
- Addition of 'forward_hours' config parameter for direct search (to account for that unix timestamps sometimes are the start time of the upload and not the end time)
- Addition of 'debug' also as a config file parameter (if set to true the cmd line output will be saved to logfile.txt)
- Addition of 'first_group_only' config file parameter to search first group only during direct search (based on how messages are stored on news servers, the chance to get different results in different groups is virtually zero)

#### v0.1.2
- Fix: Move arguments and config functions to main() to avoid blocking of SIGINT
- Graceful handling of manual aborts

#### v0.1.1
- Fix: ignore case if checking for categories (THX @shyper)

#### v0.1.0
- New flag '--register' to force registering the NZBLNK

#### v0.1.0-rc1
- Make SSL check skip a configurable option

#### v0.1.0-beta4
- Fix panic on NNTP connection errors and empty connectionGuard channel if no connection was established.

#### v0.1.0-beta3
- directsearch: new configuration parameter "skip" (default true). If set to true directsearch is skipped if a valid NZB file was already found.
- Fix for the message scan getting stuck on errors.
- Fix for incorrect parsing of the configuration file for values containing a "#" character.
- Reduced default threshold for missing files and segments.

#### v0.1.0-beta2
- Waiting time before exiting the programme is now configurable
- directsearch: Change of the configurable time span for the search backwards from days to hours.

  If the parameter date is specified as Unix timestamp, the search starts from date minus the configured number of hours and goes up to date plus 1 hour (as buffer).
  If the parameter date is specified as DD.MM.YYYY, the search will cover the whole day plus the configured number of hours of the previous day.

- Bug fixes and elimination of race conditions

#### v0.1.0-beta1
- first public release
