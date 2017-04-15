# Picture splitter (picsplit)

## License
Apache License version 2.0.

## Description
Picture splitter is meant to process digital camera DCIM folder in order to split contiguous files (pictures and movies) in dedicated subfolders. When 2 pictures are taken with more than half an hour between them (configuration parameter), a new folder is created to put the most recent picture in. The file creation date is used as an input to split the files.

## Technology stack
1. Go is the language
2. Urfave/cli the CLI library
3. Logrus the logger

## Parameters
* -nomov : do not move the movie files in a separate subfolder called mov
* -raw : move the raw files in a separate subfolder called raw
* -reject : move the raw files not having an associated JPEG file in a subfolder called reject
* -delta : change the default (1/2h) delta time between 2 events to be split
* -confirm : ask for user confirmation before executing an action (folder and subfolder creation)
* -dryrun : print the modification to be done without really moving the files
* -v : verbose
* -h : help

## Roadmap
### 1st release
nomov, raw and delta
### 2nd release
confirm and reject
### 3rd release
dryrun mode
### extra for later releases
add an option to read dating data from EXIF instead of file dates.
add a console HMI
