# NZB Monkey Go

A completely rewritten version in Golang of the [NZB Monkey](https://github.com/nzblnk/nzb-monkey), the reference implementation written in Python of how to handle a [NZBLNK](https://nzblnk.github.io/)-URI.
NZB Monkey Go also includes the functionality of the [NZBSearcher](https://github.com/Tensai75/nzbsearcher) so besides in the nzb search engines it can also search directly on the news server and create the NZB file from the article headers.

## Running the Monkey

Running the NZB Monkey Go is virtually identical to running the original NZB Monkey.

Differences are that in the config.txt file, 'directsearch' can be enabled as an additional search engine in the section 'SEARCHENGINES' and an additional section 'DIRECTSEARCH' is included, with additional configuration options required for the direct search on the news server to work.

See the [NZB Monkey Go Wiki](https://github.com/Tensai75/nzb-monkey-go/wiki) for further information regarding installation and configuration.

Note that directsearch requires the NZBLNK to include the news groups (parameter 'g') and an additional parameter for the date ('d') when the post was uploaded to the Usenet.
Parameter 'd' can be provided either in the format 'DD.MM.YYYY' or as an unix timestamp.

![Monkey-Gif](https://github.com/Tensai75/nzb-monkey-go/raw/main/resources/nzbmonkey-go.gif)

## Windows and Linux binaries

The binaries are available on the [release page](https://github.com/Tensai75/nzb-monkey-go/releases).

The linux binary must be chmoded to be executable before it can be started.

### macOS Support

A macOS binary is provided on the [release page](https://github.com/Tensai75/nzb-monkey-go/releases) as well, however the program cannot register the nzblnk URI protocol itself.
On the NZB Monkey github some solutions on how to register the protocol have been [discussed](https://github.com/nzblnk/nzb-monkey/issues/20) which should work for the NZB Monkey Go as well.

Please also note that the macOS binary is neither signed nor executable.

## Contribution

Feel free to send pull requests.

## Change log
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