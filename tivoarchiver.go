package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/url"
	"os"
	"os/exec"
	"os/signal"
	"path"
	"regexp"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/rickyninja/tivo"
	"github.com/rickyninja/tvmaze"
	"gopkg.in/yaml.v2"
)

var exe string = path.Base(os.Args[0])
var meta_cachefile = "/tmp/tv-meta-cache"
var tivo_cachefile = "/tmp/tivo-cache"
var conffile = fmt.Sprintf("/etc/%s.yml", exe)
var conf *Conf
var debug bool
var (
	cache_tivo, cache_meta bool
)
var configure bool
var searchindex SearchIndex

type Conf struct {
	ArchiveDir string   `yaml:"archive_dir"`
	MAK        int      `yaml:"mak"`
	TivoHost   string   `yaml:"tivo_host"`
	SleepFor   int      `yaml:"sleep_for"`
	Region     string   `yaml:"region"`
	Extensions []string `yaml:"extensions"`
}

type SearchIndex map[string]string

func build_search_index(dir string, index SearchIndex) SearchIndex {
	files, err := ioutil.ReadDir(dir)
	if err != nil {
		log.Fatal(fmt.Sprintf("Failed to ReadDir %s: %s", dir, err.Error()))
	}

	for _, f := range files {
		if f.Name() == "." || f.Name() == ".." {
			continue
		}

		path := dir + "/" + f.Name()

		if f.Mode().IsDir() {
			build_search_index(path, index)
		} else if f.Mode().IsRegular() {
			index[f.Name()] = path
		}
	}

	return index
}

func sigTivoFile(tivo_file string) {
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, os.Interrupt, syscall.SIGTERM)
	<-sigs
	os.Remove(tivo_file)
	os.Exit(1)
}

func main() {
	flag.Parse()
	lock := lockrun()
	defer lock.Close()
	conf = load_config()

	perr := os.Chdir(conf.ArchiveDir)
	if perr != nil {
		log.Fatal("Failed to chdir: " + perr.Error())
	}

	searchindex = make(SearchIndex)
	searchindex = build_search_index(conf.ArchiveDir, searchindex)

	maze := tvmaze.New(meta_cachefile)
	maze.UseCache = cache_meta
	tc := tivo.New(conf.TivoHost, "tivo", "https", conf.MAK, tivo_cachefile)
	tc.UseCache = cache_tivo
	if debug {
		maze.Debug = true
		tc.Debug = true
	}
	param := make(map[string]string)
	param["Container"] = "/NowPlaying"
	param["Recurse"] = "Yes"

	containers := tc.QueryContainer(param)
	transCount := 0
	trans := make(chan TranStatus)

	for _, ci := range containers {
		dl, err := download(tc, maze, ci)
		if err != nil {
			fmt.Printf("Download failed: %s\n", err.Error())
			continue
		}

		if !dl.Occurred {
			continue
		}

		tivo_file := dl.File
		mpg_file := strings.TrimSuffix(tivo_file, ".tivo") + ".mpg"
		tmp_file := conf.ArchiveDir + "/" + path.Base(tivo_file)

		if debug {
			fmt.Printf("downloaded: %s\n", tmp_file)
		}

		err = tivo_decode(mpg_file, tmp_file)
		if err != nil {
			fmt.Printf("Failed to tivo_decode(%s, %s): %s\n", mpg_file, tmp_file, err.Error())
			continue
		}
		os.Remove(tmp_file)

		transCount++
		go transcode(mpg_file, trans)

		// Pause between downloads in attempt to prevent tivo network failure.
		if conf.SleepFor > 0 {
			time.Sleep(time.Duration(conf.SleepFor) * time.Second)
		}
	}

	for i := 0; i < transCount; i++ {
		status := <-trans
		reportTranStatus(status)
	}

	// log.Fatal() is preventing cache from being written.
	maze.WriteCache()
	tc.WriteCache()
}

func reportTranStatus(status TranStatus) {
	// HandBrakeCLI will fail on occasion; gui seems to always work.
	// Best effort to transcode to reduce file size, but save the file even if transcode fails.
	if status.Err == nil {
		if debug {
			fmt.Printf("Transcode passed removing %s\n", status.OrigFile)
		}
		err := os.Remove(status.OrigFile)
		if err != nil {
			fmt.Printf("%s\n", err.Error())
		}
	} else {
		if debug {
			fmt.Printf("Transcode failed: %s removing %s\n", status.Err.Error(), status.TranFile)
		}
		err := os.Remove(status.TranFile)
		if err != nil {
			fmt.Printf("%s\n", err.Error())
		}
	}
}

type Download struct {
	Occurred bool
	File     string
}

func download(tc *tivo.Tivo, maze *tvmaze.Client, ci tivo.ContainerItem) (Download, error) {
	if ci.InProgress == "Yes" {
		return Download{false, ""}, nil
	}

	// These are the recordings that need to be downloaded.
	if !strings.Contains(ci.CustomIconURL, "save-until-i-delete-recording") {
		return Download{false, ""}, nil
	}

	detail := tc.GetDetail(ci)
	ci.Detail = detail
	// Mr. Robot not matching.  http://services.tvrage.com/feeds/episode_list.php?sid=42422
	filename, err := getFilename(tc, maze, &detail)
	if err != nil {
		return Download{false, ""}, err
	}

	tivofilename := filename + ".tivo"

	if alreadyDownloaded(filename) {
		return Download{false, ""}, nil
	}

	uri, err := url.Parse(ci.ContentURL)
	if err != nil {
		return Download{false, ""}, err
	}

	if debug {
		fmt.Printf("downloading %s\n", tivofilename)
	}

	go sigTivoFile(tivofilename)
	resp := tc.Go(uri)
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return Download{false, ""}, errors.New(resp.Status)
	}

	out, err := os.Create(tivofilename)
	if err != nil {
		os.Remove(tivofilename)
		return Download{false, ""}, err
	}
	defer out.Close()

	_, err = io.Copy(out, resp.Body)
	if err != nil {
		os.Remove(tivofilename)
		return Download{false, ""}, err
	}

	dest := conf.ArchiveDir + "/"
	if detail.IsEpisodic {
		slice := []string{"tv", detail.Title}
		dest += strings.Join(slice, "/")
	} else {
		dest += "movies"
	}

	err = os.MkdirAll(dest, 0755)
	if err != nil {
		return Download{false, ""}, err
	}

	return Download{true, dest + "/" + tivofilename}, nil
}

func alreadyDownloaded(filename string) bool {
	// Match any configured file extension (m4v, mpg, etc.).
	for _, ext := range conf.Extensions {
		found := searchindex[filename+"."+ext]
		if found != "" {
			if debug {
				fmt.Printf("already downloaded %s\n", path.Base(found))
			}
			return true
		} else {
			if debug {
				fmt.Printf("Failed to find %s.%s\n", filename, ext)
			}
		}
	}
	return false
}

type TranStatus struct {
	OrigFile string
	TranFile string
	Err      error
}

func transcode(file string, trans chan<- TranStatus) {
	mp4 := strings.TrimSuffix(file, ".mpg") + ".m4v"
	cmd := exec.Command("/usr/bin/HandBrakeCLI", "-i", file, "-o", mp4,
		"--audio", "1", "--aencoder", "copy:aac", "--audio-fallback", "faac",
		"--audio-copy-mask", "aac", "--preset=Universal")

	stderr, err := cmd.StderrPipe()
	if err != nil {
		trans <- TranStatus{file, mp4, err}
		return
	}

	err = cmd.Start()
	if err != nil {
		trans <- TranStatus{file, mp4, err}
		return
	}

	bytes, _ := ioutil.ReadAll(stderr)
	msg := strings.TrimSpace(string(bytes))

	err = cmd.Wait()
	if err != nil {
		trans <- TranStatus{file, mp4, errors.New(msg)}
		return
	}
	trans <- TranStatus{file, mp4, nil}
}

func tivo_decode(mpgfilename, tivofilename string) error {
	cmd := exec.Command("/usr/local/bin/tivodecode", "--mak", strconv.Itoa(conf.MAK), "--out", mpgfilename, tivofilename)
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return err
	}

	err = cmd.Start()
	if err != nil {
		return err
	}

	bytes, _ := ioutil.ReadAll(stderr)
	msg := strings.TrimSpace(string(bytes))

	err = cmd.Wait()
	if err != nil {
		return errors.New(msg)
	}
	return nil
}

func init() {
	flag.BoolVar(&debug, "debug", false, "--debug")
	flag.BoolVar(&cache_tivo, "cache-tivo", false, "Use cache of tivo api query results.")
	flag.BoolVar(&cache_meta, "cache-meta", true, "Use cache of tv meta data api results.")
	flag.BoolVar(&configure, "configure", false, "--configure")
}

func lockrun() *os.File {
	lockfile := fmt.Sprintf("/tmp/%s.pid", path.Base(os.Args[0]))
	fd, err := os.OpenFile(lockfile, os.O_RDWR|os.O_CREATE, 0666)
	if err != nil {
		log.Fatal(fmt.Sprintf("Failed to open %s: %s", lockfile, err.Error()))
	}

	_, err = fd.Write([]byte(fmt.Sprintf("%d", os.Getpid())))
	if err != nil {
		log.Fatal("Failed to write pid to lockfile: " + err.Error())
	}

	err = syscall.Flock(int(fd.Fd()), syscall.LOCK_EX|syscall.LOCK_NB)
	if err != nil {
		log.Fatal("Failed to obtain lock: " + err.Error())
	}
	return fd
}

func load_config() *Conf {
	conf := &Conf{}
	yamldata, err := ioutil.ReadFile(conffile)
	if err != nil {
		log.Fatal("Failed to open yaml config: " + err.Error())
	}

	err = yaml.Unmarshal([]byte(yamldata), conf)
	if err != nil {
		log.Fatal("Failed to load yaml config: " + err.Error())
	}

	return conf
}

func getFilename(tc *tivo.Tivo, maze *tvmaze.Client, detail *tivo.VideoDetail) (string, error) {
	filename := detail.Title

	if detail.IsEpisodic {
		show, err := maze.FindShow(detail.Title)
		if err != nil {
			return "", err
		}

		episodes, err := maze.GetEpisodes(show.ID)
		if err != nil {
			return "", err
		}

		episode, err := findEpisode(detail, episodes)
		if err != nil {
			fmt.Println(err.Error())
			// Some shows have several candidates in the tvrage api, and no data
			// in the tivo to disambiguate the candidates (Being Human for example).
			// If the episode_number is all digits, it's hopefully accurate.
			re := regexp.MustCompile(`^(\d{1,})(\d{2})$`)
			captures := re.FindStringSubmatch(detail.EpisodeNumber)
			if captures != nil {
				s, serr := strconv.Atoi(captures[1])
				ep, eperr := strconv.Atoi(captures[2])
				if serr == nil && eperr == nil {
					filename += fmt.Sprintf(" %dx%.2d-%s", s, ep, detail.EpisodeTitle)
				} else if serr != nil {
					return "", errors.New("string to int conversion failed for serr: " + serr.Error())
				} else if serr != nil {
					return "", errors.New("string to int conversion failed for eperr: " + eperr.Error())
				}
			}
		} else {
			detail.EpisodeNumber = fmt.Sprintf("%d%.2d", episode.Season, episode.Number)
			filename += fmt.Sprintf(" %dx%.2d-%s", episode.Season, episode.Number, detail.EpisodeTitle)
		}
	}

	filename = strings.Replace(filename, "/", "-", -1)
	return filename, nil
}

func findEpisode(detail *tivo.VideoDetail, episodes []tvmaze.Episode) (tvmaze.Episode, error) {
	for desperate := 0; desperate <= 3; desperate++ {
		for _, episode := range episodes {
			// normalize chars ’ vs ' etc.
			var mtb []byte
			var ttb []byte
			var tivotitle string
			var mazetitle string
			if len(episode.Name) == len(detail.EpisodeTitle) {
				mtb = []byte(episode.Name)
				ttb = []byte(detail.EpisodeTitle)
				for i := 0; i < len(mtb); i++ {
					ord := int(mtb[i])
					if ord < 32 || ord > 126 {
						mtb = append(mtb[:i], mtb[i+1:]...)
						ttb = append(ttb[:i], ttb[i+1:]...)
					}
				}
				tivotitle = string(ttb[:])
				mazetitle = string(mtb[:])
			} else {
				tivotitle = detail.EpisodeTitle
				mazetitle = episode.Name
			}

			// As we become more desperate to find a match strip out non-word characters
			// to make a match more likely.
			if desperate >= 2 {
				re := regexp.MustCompile(`\W`)
				tivotitle = string(re.ReplaceAll([]byte(tivotitle), []byte("")))
				mazetitle = string(re.ReplaceAll([]byte(mazetitle), []byte("")))
			}

			// exact title match
			if strings.ToLower(mazetitle) == strings.ToLower(tivotitle) {
				return episode, nil
				// exact title match if you add part_index inside parens to tivo title
			} else if detail.PartIndex > 0 && desperate == 0 {
				tt := fmt.Sprintf("%s (%d)", tivotitle, detail.PartIndex)
				if tt == mazetitle {
					return episode, nil
				}
				// rage title contains tivo title
			} else if desperate == 1 && strings.Contains(mazetitle, tivotitle) {
				return episode, nil
				// tivo title contains rage title
			} else if desperate == 1 && strings.Contains(tivotitle, mazetitle) {
				detail.EpisodeNumber = fmt.Sprintf("%d%.2d", episode.Season, episode.Number)
				return episode, nil
			} else if desperate == 1 {
				// try to match 'Kill Billie: Vol.2' with 'Kill Billie (2)'
				re := regexp.MustCompile(`\((\d+)\)/`)
				captures := re.FindStringSubmatch(mazetitle)

				if captures != nil {
					mt := string(re.ReplaceAll([]byte(mazetitle), []byte("")))
					sequel := captures[1]
					mt = strings.TrimSpace(mt)
					if strings.Contains(tivotitle, mt) && strings.Contains(tivotitle, sequel) {
						return episode, nil
					}
				} else if strings.Contains(mazetitle, " and ") && strings.Contains(tivotitle, "&") {
					tt := strings.Replace(tivotitle, "&", "and", -1)
					if mazetitle == tt {
						return episode, nil
					}
				} else if strings.Contains(mazetitle, "&") && strings.Contains(tivotitle, " and ") {
					tt := strings.Replace(tivotitle, " and ", " & ", -1)
					if mazetitle == tt {
						return episode, nil
					}
				}
			}
		}
	}

	return tvmaze.Episode{}, errors.New("Failed to ID season and episode!")
}
