# SemanticMerge external parser for PHP

## Usage of `semantic-php`:

```
Usage:
  ./semantic-php [options] shell {flagfile}
	shell interactive mode
  ./semantic-php [options] {source}
	parse single file
Options:
  -debug
    	log extra parse info to stderr
  -php version
    	parse using php version (5 or 7) (default 7)
  -proto version
    	SemanticMerge protocol version (1 or 2) (default 2)
```

An example diffing two files with SemanticMerge using PHP 5:

```batchfile
set parser=%gopath%\bin\semantic-php.exe -php 5
semanticmergetool.exe -ep=%parser% src.php dst.php
```

## Install from source (Windows)

First fetch and install Go from <https://golang.org/dl/> and git from
<https://git-scm.com/download> if you don't already have them.
Then

```
go get github.com/imuli/semantic-php
```

will install `semantic-php.exe` into your Go binary path above.

## Binaries

...will be released soon.

