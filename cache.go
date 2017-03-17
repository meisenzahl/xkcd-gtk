package main

import (
	"encoding/json"
	"errors"
	"github.com/rkoesters/xkcd"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
)

func getComicPath(n int) string {
	return filepath.Join(CacheDir(), strconv.Itoa(n))
}

func getComicInfoPath(n int) string {
	return filepath.Join(getComicPath(n), "info")
}

func getComicImagePath(n int) string {
	return filepath.Join(getComicPath(n), "image")
}

// GetComicInfo always returns a valid *xkcd.Comic that can be used, and
// err will be set if any errors were encountered, however these errors
// can be ignored safely.
func GetComicInfo(n int) (*xkcd.Comic, error) {
	infoPath := getComicInfoPath(n)

	// First, check if we have the file.
	_, err := os.Stat(infoPath)
	if os.IsNotExist(err) {
		err = downloadComicInfo(n)
		if err == xkcd.ErrNotFound {
			return &xkcd.Comic{
				Num:       n,
				Title:     "Comic Not Found",
				SafeTitle: "Comic Not Found",
			}, err
		} else if err != nil {
			return &xkcd.Comic{
				Num:       n,
				Title:     "Couldn't Get Comic",
				SafeTitle: "Couldn't Get Comic",
			}, err
		}
	} else if err != nil {
		return &xkcd.Comic{
			Num:       n,
			Title:     "I guess we can't access our cache",
			SafeTitle: "I guess we can't access our cache",
		}, err
	}

	f, err := os.Open(infoPath)
	if err != nil {
		return &xkcd.Comic{
			Num:       n,
			Title:     "Error trying to read comic info from cache",
			SafeTitle: "Error trying to read comic info from cache",
		}, err
	}
	defer f.Close()
	c, err := xkcd.New(f)
	if err != nil {
		return &xkcd.Comic{
			Num:       n,
			Title:     "I guess the cached comic info is invalid",
			SafeTitle: "I guess the cached comic info is invalid",
		}, err
	}
	return c, nil
}

var cachedNewestComic *xkcd.Comic

var (
	// ErrCache means that there was an error while trying to access the
	// local cache.
	ErrCache = errors.New("error accessing local xkcd cache")
	// ErrOffline means that there was an error trying to access the
	// xkcd server.
	ErrOffline = errors.New("error accessing xkcd server")
)

// GetNewestComicInfo always returns a valid *xkcd.Comic that appears to
// be newest, and err will be set if any errors were encountered,
// however these errors can be ignored safely.
func GetNewestComicInfo() (*xkcd.Comic, error) {
	var err error
	if cachedNewestComic == nil {
		cachedNewestComic, err = xkcd.GetCurrent()
		if err != nil {
			newestAvaliable := &xkcd.Comic{
				Num:   0,
				Title: "Connect to the internet to download some comics!",
			}

			d, err := os.Open(CacheDir())
			if err != nil {
				return newestAvaliable, ErrCache
			}
			defer d.Close()

			cachedirs, err := d.Readdirnames(0)
			if err != nil {
				return newestAvaliable, ErrCache
			}

			for _, f := range cachedirs {
				comicID, err := strconv.Atoi(f)
				if err != nil {
					continue
				}
				comic, err := GetComicInfo(comicID)
				if err != nil {
					continue
				}
				if comicID > newestAvaliable.Num {
					newestAvaliable = comic
				}
			}
			return newestAvaliable, ErrOffline
		}
	}
	return cachedNewestComic, nil
}

func downloadComicInfo(n int) error {
	comic, err := xkcd.Get(n)
	if err != nil {
		return err
	}

	err = os.MkdirAll(getComicPath(n), 0777)
	if err != nil {
		return err
	}

	f, err := os.Create(getComicInfoPath(n))
	if err != nil {
		return err
	}
	defer f.Close()

	e := json.NewEncoder(f)
	err = e.Encode(comic)
	if err != nil {
		return err
	}

	// Now add the new file to the searchIndex.
	return searchIndex.Index(strconv.Itoa(comic.Num), comic)
}

// DownloadComicImage tries to add a comic image to our local cache. Any
// errors are indicated by err.
func DownloadComicImage(n int) error {
	c, err := GetComicInfo(n)
	if err != nil {
		return err
	}

	resp, err := http.Get(c.Img)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	f, err := os.Create(getComicImagePath(n))
	if err != nil {
		return err
	}
	defer f.Close()

	_, err = io.Copy(f, resp.Body)
	if err != nil {
		return err
	}
	return nil
}
