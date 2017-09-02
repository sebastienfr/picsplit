# Picture splitter (picsplit)

## License
Apache License version 2.0.

## Description
Picture splitter is meant to process digital camera DCIM folder
in order to split contiguous files (pictures and movies) in dedicated subfolders.
When 2 pictures are taken with more than an hour between them (configuration parameter),
a new folder is created to put the most recent picture in it.
The file creation date is used as parameter to split the files.

## Create test data

     etc/mktest.sh
     make && rm -rf data && etc/mktest.sh && ls -lR data && picsplit -v -dryrun -moveraw -movemov ./data && ls -lR data

## Usage

    picsplit -v -dryrun ./data
    picsplit -v ./data

## Technology stack
1. Go is the language
2. Urfave/cli the CLI library
3. Logrus the logger

## Parameters
* -nomovemov : do not move the movie files in a separate subfolder called mov
* -nomoveraw : move the raw files in a separate subfolder called raw
* -delta : change the default (1/2h) delta time between 2 events to be split
* -dryrun : print the modification to be done without really moving the files
* -v : verbose
* -h : help

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

