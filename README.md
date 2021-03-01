# Brutus

A tool for brute-forcing cli tool arguments using file-based wordlists.

# Installation

```
go get github.com/retro-engineer/brutus
go install ~/go/src/github.com/retro-engineer/brutus
```

# Usage

```
$ ~/go/bin/brutus
Usage: ~/go/bin/brutus [OPTION] COMMANDS

Options:
  -file:     array of files to read contents from
  -shell:    execution shell (default: /bin/sh)
  -shellarg: execution shell argument array (default: -c)
  -success:  strings that indicate success
  -failure:  strings that indicate failure. These have precedense over success strings.
  -fuzz:     the identifier that is replaced by the file contents.
             The curly braces are replaced by '', '2', '3', etc. (default: FUZ{}Z)
  -threads:  the number of concurrent workers (default: 10)
  -tries:    print the tried combinations
  -positive: treat an absense of failure as a success
  -progress: print progress
  -verbose:  verbose output

Examples:
  SMB   user/pass bruteforce:  ~/go/bin/brutus -file users.txt -file passwords.txt -success IP smbmap -u FUZZ -p FUZ2Z -H 10.10.10.179
  MySQL user/pass bruteforce:  ~/go/bin/brutus -file users.txt -file passwords.txt -failure 'Access denied' mysql -u\'FUZZ\' -p\'FUZ2Z\' --host=127.0.0.1
  SSH   user/pass bruteforce:  ~/go/bin/brutus -file users.txt -file passwords.txt -success Success ssh-test 10.10.10.179 22 FUZZ 'FUZ2Z'
```

# Optional helpers
```
GOBIN=~/go/bin go install ~/go/src/github.com/retro-engineer/brutus/helpers/ssh-test.go
```
