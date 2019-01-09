# Picture splitter (picsplit)

[![Build Status](https://travis-ci.org/sebastienfr/picsplit.svg?branch=master)](https://travis-ci.org/sebastienfr/picsplit)
[![GoDoc](https://godoc.org/github.com/sebastienfr/picsplit?status.svg)](https://godoc.org/github.com/sebastienfr/picsplit)
[![Software License](http://img.shields.io/badge/license-APACHE2-blue.svg)](https://github.com/sebastienfr/picsplit/blob/master/LICENSE)

## License
Apache License version 2.0.

## Description
Picture splitter (`picsplit`) is meant to process digital camera DCIM folder
in order to split contiguous files (pictures and movies) in dedicated subfolders.
When 2 pictures are taken with more than an hour between them (configuration parameter),
a new folder is created (folder pattern YYYY - MMDD - hhmm) to put the most recent picture in it.
The file creation date is used as parameter to split the files.

Supported extension are the following :

- Image : JPG, JPEG,
- Raw : NEF, NRW, CR2, CRW
- Movie : MOV, AVI, MP4

## Technology stack

1. [Go](https://golang.org) is the language
2. [Urfave/cli](https://github.com/urfave/cli) the CLI library
3. [Logrus](https://github.com/sirupsen/logrus) the logger

## CLI Parameters

* `-nomvmov` : do not move movies in a separate `mov` folder
* `-nomvraw` : do not move raw files in a separate `raw` folder
* `-delta` : change the default (1h) delta time between 2 files to be split
* `-dryrun` : print the modification to be done without really moving the files
* `-v` : verbose
* `-h` : help

## Results

Effects of **picsplit** are the following :

```
data
├── PHOTO_01.JPG
├── PHOTO_02.JPG
├── PHOTO_03.CR2
├── PHOTO_03.JPG
├── PHOTO_04.JPG
├── PHOTO_04.MOV
├── PHOTO_04.NEF
├── PHOTO_04.test
└── TEST
```

to

```
data
├── 2019 - 0216 - 0900
│   └── PHOTO_01.JPG
├── 2019 - 0216 - 1000
│   └── PHOTO_02.JPG
├── 2019 - 0216 - 1100
│   ├── PHOTO_03.JPG
│   └── raw
│       └── PHOTO_03.CR2
├── 2019 - 0216 - 1200
│   ├── PHOTO_04.JPG
│   ├── mov
│   │   └── PHOTO_04.MOV
│   └── raw
│       └── PHOTO_04.NEF
├── PHOTO_04.test
└── TEST
```

## Build and test

Makefile is used to build `picsplit`

     make
     
Create a populated test folder

     etc/mktest.sh

## Usage

    picsplit -v -dryrun ./data
    picsplit -v ./data
    picsplit -v -nomvmov -nomvraw ./data

## Roadmap

### First release

- [X] configurable delta
- [X] move movies
- [X] move raw
- [X] dry run mode

### Next releases

- [ ] merge folder command (case split too much)
- [ ] add an option to read dating data from EXIF instead of file dates
- [ ] add a console GUI

