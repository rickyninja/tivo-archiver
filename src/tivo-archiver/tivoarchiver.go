package main

import (
    "flag"
    "fmt"
    "gopkg.in/yaml.v2"
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
    "tivo"
    "tvrage"
)

var exe string = path.Base(os.Args[0])
var ragecachefile = "/tmp/go-tvrage-cache"  // combined cachefile?
var conffile = fmt.Sprintf("/etc/%s.yml", exe)
var conf *Conf
var debug bool
var use_cache bool
var configure bool
var searchindex SearchIndex

type Conf struct {
    ArchiveDir string `yaml:"archive_dir"`
    MAK int `yaml:"mak"`
    TivoDecode int `yaml:"tivo_decode"`
    TivoHost string `yaml:"tivo_host"`
    SleepFor int `yaml:"sleep_for"`
    Region string `yaml:"region"`
    Extensions []string `yaml:"extensions"`
}

type SearchIndex map[string]string

type DownloadStatus struct {
    filename string
    downloaded bool
}

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

func main() {
    flag.Parse()
    lockrun()
    conf = load_config()

    perr := os.Chdir(conf.ArchiveDir)
    if perr != nil {
        log.Fatal("Failed to chdir: " + perr.Error())
    }

    searchindex = make(SearchIndex)
    searchindex = build_search_index(conf.ArchiveDir, searchindex)

    rage := tvrage.New(ragecachefile)
    tc := tivo.New(conf.TivoHost, "tivo", "https", conf.MAK, "go-tivo-cache")
    tc.UseCache = use_cache
    if debug {
        rage.Debug = true
        tc.Debug = true
    }
    param := make(map[string]string)
    param["Container"] = "/NowPlaying"
    param["Recurse"] = "Yes"

    ch := make(chan DownloadStatus)
    containers := tc.QueryContainer(param)
    for _, ci := range containers {
        download(tc, rage, ci, ch)
    }

    for i := 1; i <= len(containers); i++ {
        status := <-ch
        if status.downloaded && debug {
            fmt.Printf("downloaded: %s\n", status.filename)
        }
    }

    // log.Fatal() is preventing cache from being written.
    rage.WriteCache()
    tc.WriteCache()
}

func download(tc *tivo.Tivo, rage *tvrage.Client, ci tivo.ContainerItem, ch chan DownloadStatus) {
    status := DownloadStatus{}
    if ci.InProgress == "Yes" {
        go func() { ch <- status }()
        return
    }

    // These are the recordings that need to be downloaded.
    if ! strings.Contains(ci.CustomIconURL, "save-until-i-delete-recording") {
        go func() { ch <- status }()
        return
    }

    detail := tc.GetDetail(ci)
    ci.Detail = detail
    // Mr. Robot not matching.  http://services.tvrage.com/feeds/episode_list.php?sid=42422
    filename, err := tc.GetFilename(rage, &detail)
    if err != nil {
        log.Print("Failed to get tivo filename: " + err.Error())
        go func() { ch <- status }()
        return
    }

    tivofilename := filename + ".tivo"
    mpgfilename := filename + ".mpg"
    pymeta := tc.GetPymeta(&detail)
    pymetafile := fmt.Sprintf("%s/.meta/%s.txt", conf.ArchiveDir, mpgfilename) // default

    // Match any configured file extension (m4v, mpg, etc.).
    for _, ext := range conf.Extensions {
        found := searchindex[filename + "." + ext]
        if found != "" {
            pymetafile = fmt.Sprintf("%s/.meta/%s.txt", conf.ArchiveDir, path.Base(found))
            write_pymeta(pymeta, pymetafile)
            if debug {
                fmt.Printf("already downloaded %s\n", path.Base(found))
            }
            set_orig_air_date(detail, found)
            go func() { ch <- status }()
            return
        } else {
            if debug {
                fmt.Printf("Failed to find %s.%s\n", filename, ext)
            }
        }
    }

    write_pymeta(pymeta, pymetafile)
    if debug {
        fmt.Printf("downloading %s\n", mpgfilename)
    }
    uri, err := url.Parse(ci.ContentURL)
    if err != nil {
        log.Fatal(err)
    }

    resp := tc.Go(uri)
    defer resp.Body.Close()
    if resp.StatusCode != 200 {
        log.Println(resp.Status)
        go func() { ch <- status }()
        return
    }

    sigs := make(chan os.Signal, 1)
    signal.Notify(sigs, os.Interrupt, syscall.SIGTERM)
    go func() {
        <-sigs
        os.Remove(tivofilename)
        os.Exit(1)
    }()

    out, err := os.Create(tivofilename)
    if err != nil {
        log.Println(err.Error())
        os.Remove(tivofilename)
        go func() { ch <- status }()
        return
    }
    defer out.Close()

    _, err = io.Copy(out, resp.Body)
    if err != nil {
        log.Println(err.Error())
        os.Remove(tivofilename)
        go func() { ch <- status }()
        return
    }
    status.downloaded = true

    dest := conf.ArchiveDir + "/"
    if detail.IsEpisodic {
        season := 0
        re := regexp.MustCompile(`^(\d+)\d{2}$`)
        s := re.FindStringSubmatch(detail.EpisodeNumber)
        if s != nil {
            season, err = strconv.Atoi(s[1])
        }
        slice := []string{"tv", detail.Title, fmt.Sprintf("season%d", season)}
        dest += strings.Join(slice, "/")
    } else {
        dest += "movies"
    }

    err = os.MkdirAll(dest, 0755)
    if err != nil {
        log.Println(err.Error())
        go func() { ch <- status }()
        return
    }

    fi, err := os.Lstat(dest + "/.meta")
    if err != nil || fi.Mode() & os.ModeSymlink != os.ModeSymlink {
        err := os.Symlink(conf.ArchiveDir + "/.meta", dest + "/.meta")
        if err != nil {
            fmt.Printf("Failed to create symlink at %s/.meta\n", dest)
        }
    }

    if conf.TivoDecode == 1 {
        err := tivo_decode(mpgfilename, tivofilename)
        if err != nil {
            log.Fatal("Failed to decode tivo file to mpeg: " + err.Error())
        } else {
            os.Remove(tivofilename)
        }

        err = os.Rename(mpgfilename, dest + "/" + mpgfilename)
        if err != nil {
            log.Fatal(err.Error());
        }

        file := dest + "/" + mpgfilename

        go func() {
            file = transcode(file)
            status.filename = file
            set_orig_air_date(detail, file)
            ch <- status
        }()
    } else {
        absfile := dest + "/" + tivofilename
        err = os.Rename(tivofilename, absfile)
        if err != nil {
            log.Fatal(fmt.Sprintf("Failed to move %s to $dest: %s", mpgfilename, err.Error()));
        }
        set_orig_air_date(detail, absfile)
        status.filename = absfile
        go func() { ch <- status }()
    }

    // Pause between downloads in attempt to prevent tivo network failure.
    if conf.SleepFor > 0 {
        //time.Sleep(time.Duration(conf.SleepFor) * time.Second)
    }
}

func set_orig_air_date(detail tivo.VideoDetail, file string) {
    if ! detail.IsEpisodic {
        return
    }

    _, err := os.Stat(file)
    if err != nil {
        fmt.Printf("Failed to stat %s: %s\n", file, err.Error())
    }

    mtime, err := time.Parse("2006-01-02T15:04:05Z", detail.OriginalAirDate)
    if err != nil {
        fmt.Printf("Failed parse OAD: %s\n", err.Error())
        return
    }

    err = os.Chtimes(file, mtime, mtime)
    if err != nil {
        fmt.Printf("Failed to update timestamp for %s: %s\n", file, err.Error())
    }
}

func transcode(file string) string {
    mp4 := strings.TrimSuffix(file, ".mpg") + ".m4v"
    cmd := exec.Command("/usr/bin/HandBrakeCLI", "-i", file, "-o", mp4,
        "--audio", "1", "--aencoder", "copy:aac", "--audio-fallback", "faac",
        "--audio-copy-mask", "aac", "--preset=Universal")

    // HandBrakeCLI will fail on occasion; gui seems to always work.
    // Best effort to transcode to reduce file size, but save the file even if transcode fails.
    err := cmd.Run()
    if err == nil {
        if debug {
            fmt.Printf("Transcode passed removing %s\n", file)
        }
        err := os.Remove(file)
        if err != nil {
            fmt.Printf("%s\n", err.Error())
        }
        return mp4
    } else {
        if debug {
            fmt.Printf("Transcode failed: %s removing %s\n", err.Error(), mp4)
        }
        err := os.Remove(mp4)
        if err != nil {
            fmt.Printf("%s\n", err.Error())
        }
        return file
    }
}

func tivo_decode(mpgfilename, tivofilename string) error {
    cmd := exec.Command("/usr/local/bin/tivodecode", "--mak", strconv.Itoa(conf.MAK), "--out", mpgfilename, tivofilename)
    err := cmd.Run()
    return err
}

func write_pymeta(pymeta, file string) {
    fd, err := os.Create(file)
    if err != nil {
        log.Fatal(fmt.Sprintf("Failed to open %s: %s", file, err.Error()))
    }
    defer fd.Close()

    _, err = fd.Write([]byte(pymeta))
    if err != nil {
        log.Fatal("Failed to write pymeta file: " + err.Error())
    }
}

func init() {
    flag.BoolVar(&debug, "debug", false, "--debug")
    flag.BoolVar(&use_cache, "usecache", false, "--usecache")
    flag.BoolVar(&configure, "configure", false, "--configure")
}

func lockrun() {
    lockfile := fmt.Sprintf("/tmp/%s.pid", path.Base(os.Args[0]))
    fd, err := os.OpenFile(lockfile, os.O_RDWR | os.O_CREATE, 0666)
    if err != nil {
        log.Fatal(fmt.Sprintf("Failed to open %s: %s", lockfile, err.Error()))
    }

    _, err = fd.Write([]byte(fmt.Sprintf("%d", os.Getpid())))
    if err != nil {
        log.Fatal("Failed to write pid to lockfile: " + err.Error())
    }

    err = syscall.Flock(int(fd.Fd()), syscall.LOCK_EX | syscall.LOCK_NB)
    if err != nil {
        log.Fatal("Failed to obtain lock: " + err.Error())
    }
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

