package handler

import (
	"github.com/sirupsen/logrus"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const (
	// movie extension
	mov = ".mov"
	avi = ".avi"
	mp4 = ".mp4"

	// picture extension
	jpg  = ".jpg"
	jpeg = ".jpeg"

	// raw picture extension
	nef = ".nef"
	nrw = ".nrw"
	crw = ".crw"
	cr2 = ".cr2"
	rw2 = ".rw2"

	maxFiles          = 100
	folderNamePattern = "2006 - 0102 - 1504"
	movFolder         = "mov"
	rawFolder         = "raw"
)

var (
	// movieExtension the list of movie file extension
	movieExtension = map[string]bool{
		mov: true,
		avi: true,
		mp4: true,
	}

	// rawFileExtension the list of the raw file extension
	rawFileExtension = map[string]bool{
		nef: true,
		nrw: true,
		crw: true,
		cr2: true,
		rw2: true,
	}

	// jpegExtension the list of the JPG file extension
	jpegExtension = map[string]bool{
		jpg:  true,
		jpeg: true,
	}

	// split directories cache
	directories map[string]string
)

// Split is the main function the moves files to dated folders according to delta duration parameter
func Split(basedir string, delta time.Duration, noMoveMov, noMoveRaw, dryRun bool) error {
	// make directories cache
	directories = make(map[string]string)

	// list existing directories
	err := listDirectories(basedir)
	if err != nil {
		return err
	}

	// process files in dir
	err = processFiles(basedir, delta, noMoveMov, noMoveRaw, dryRun)
	if err != nil {
		return err
	}

	return nil
}

func listDirectories(basedir string) error {
	file, err := os.Open(basedir)
	if err != nil {
		return err
	}
	defer file.Close()

	fos, err := file.Readdir(maxFiles)
	for ; err == nil; fos, err = file.Readdir(maxFiles) {
		for _, fi := range fos {
			if fi.IsDir() {
				_, err := time.Parse(folderNamePattern, fi.Name())
				if err != nil {
					logrus.Debugf("ignoring non date formatted folder : %s", fi.Name())
					continue
				}
				logrus.Debugf("found folder : %s, date : %s", fi.Name(), fi.ModTime().String())
				directories[fi.Name()] = fi.Name()
			}
		}
	}

	return nil
}

func processFiles(basedir string, delta time.Duration, noMoveMov, noMoveRaw, dryRun bool) error {
	// open folder to process
	file, err := os.Open(basedir)
	if err != nil {
		return err
	}
	defer file.Close()

	// list by batch of MaxFiles the files in the folder
	fos, err := file.Readdir(maxFiles)
	if err != nil {
		return err
	}

	for ; err == nil; fos, err = file.Readdir(maxFiles) {
		// check for errors
		if err != nil && err != io.EOF {
			return err
		}

		// for each file move it to the corresponding folder
		for _, fi := range fos {

			// only process files
			if fi.IsDir() {
				continue
			}

			logrus.Debugf("processing file : %s, date : %s", fi.Name(), fi.ModTime().String())

			// case file is a picture
			if isPicture(fi) {
				logrus.Debugf("is picture : %s, date : %s", fi.Name(), fi.ModTime().String())
				// find suitable datedFolder for file
				datedFolder, err := findOrCreateDatedFolder(basedir, fi, delta, dryRun)
				if err != nil {
					return err
				}

				// special raw file file
				destMovieDir := datedFolder
				if isRaw(fi) && !noMoveRaw {
					// create the mov directory
					baseMovieDir := filepath.Join(basedir, datedFolder)
					rawDir, err := findOrCreateFolder(baseMovieDir, rawFolder, dryRun)
					if err != nil {
						return err
					}
					destMovieDir = filepath.Join(datedFolder, rawDir)
				}

				// move file in suitable folder
				err = moveFile(basedir, fi.Name(), destMovieDir, dryRun)
				if err != nil {
					return err
				}

				// case file is a movie
			} else if isMovie(fi) {
				logrus.Debugf("is movie : %s, date : %s", fi.Name(), fi.ModTime().String())

				// find suitable datedFolder for file
				datedFolder, err := findOrCreateDatedFolder(basedir, fi, delta, dryRun)
				if err != nil {
					return err
				}

				destMovieDir := datedFolder

				// if moving mov in a separate folder
				if !noMoveMov {
					// create the mov directory
					baseMovieDir := filepath.Join(basedir, datedFolder)
					movieDir, err := findOrCreateFolder(baseMovieDir, movFolder, dryRun)
					if err != nil {
						return err
					}
					destMovieDir = filepath.Join(datedFolder, movieDir)
				}

				// move file in the right folder
				err = moveFile(basedir, fi.Name(), destMovieDir, dryRun)
				if err != nil {
					return err
				}
			} else {
				logrus.Debugf("%s is unknown extension file", fi.Name())
				continue
			}
		}
	}

	return nil
}

func findOrCreateDatedFolder(basedir string, file os.FileInfo, delta time.Duration, dryRun bool) (string, error) {
	roundedDate := file.ModTime().Round(delta).Format(folderNamePattern)

	if dryRun {
		return roundedDate, nil
	}

	f, ok := directories[roundedDate]
	if ok {
		logrus.Debugf("found suitable folder : %s", roundedDate)
		return f, nil
	}

	dirCreate := filepath.Join(basedir, roundedDate)
	logrus.Debugf("create suitable folder : %s", roundedDate)

	err := os.Mkdir(dirCreate, os.ModePerm)
	if err != nil {
		return "", err
	}

	fi, err := os.Stat(dirCreate)
	if err != nil {
		return "", err
	}

	directories[roundedDate] = fi.Name()
	return fi.Name(), nil
}

func findOrCreateFolder(basedir, name string, dryRun bool) (string, error) {
	dirCreate := filepath.Join(basedir, name)

	logrus.Debugf("find or create folder : %s", dirCreate)

	if dryRun {
		return name, nil
	}

	fi, err := os.Stat(dirCreate)
	if err != nil {
		if os.IsNotExist(err) {
			err := os.Mkdir(dirCreate, os.ModePerm)
			if err != nil {
				return "", err
			}

			fi, err = os.Stat(dirCreate)
			if err != nil {
				return "", err
			}

			return fi.Name(), nil
		}

		return "", err
	}
	return fi.Name(), nil
}

func moveFile(basedir, src, dest string, dryRun bool) error {
	srcPath := filepath.Join(basedir, src)
	dstPath := filepath.Join(basedir, dest, src)
	if !dryRun {
		logrus.Warnf("move file : %v, to %v", srcPath, dstPath)
		return os.Rename(srcPath, dstPath)
	}

	logrus.Warnf("move file : %v, to %v", srcPath, dstPath)
	return nil
}

func isMovie(file os.FileInfo) bool {
	ext := strings.ToLower(filepath.Ext(file.Name()))
	isMovie := movieExtension[ext]
	return isMovie
}

func isPicture(file os.FileInfo) bool {
	ext := strings.ToLower(filepath.Ext(file.Name()))
	isPicture := jpegExtension[ext] || rawFileExtension[ext]
	return isPicture
}

func isRaw(file os.FileInfo) bool {
	ext := strings.ToLower(filepath.Ext(file.Name()))
	isRaw := rawFileExtension[ext]
	return isRaw
}
