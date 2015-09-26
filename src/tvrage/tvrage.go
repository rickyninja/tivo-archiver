package tvrage

import (
    "encoding/xml"
    "log"
    "io/ioutil"
    "net/http"
    "net/url"
    "strings"
    "fmt"
    cache "github.com/robfig/go-cache"
    "time"
    "os"
    "errors"
    "strconv"
)

type Client struct {
    Debug bool
    BaseURI string
    Region string
    Cache *cache.Cache
    CacheFile string
    *http.Client
}

func New(cachefile string) *Client {
    c := cache.New(time.Minute * 60 * 24 * 7, time.Minute * 60)
    if _, err := os.Stat(cachefile); err == nil {
        err := c.LoadFile(cachefile)
        if err != nil {
            log.Fatal("Failed to load cache from file: " + cachefile)
        }
    }

    timeout := time.Duration(180 * time.Second)
    client := &http.Client{
        Timeout: timeout,
    }
    return &Client{false, "http://services.tvrage.com", "", c, cachefile, client}
}

func (c *Client) WriteCache() {
    err := c.Cache.SaveFile(c.CacheFile)
    if err != nil {
        log.Println(err.Error())
    }
}

func (c *Client) FindShow(showname string) (Show, error) {
    shows := c.GetShow(showname)
    // Lost Girl is listed as CA country in tvrage, and so my configured region of US
    // will not match.  Lost Girl has only aired in CA though, so you can get a string
    // equality match on the 2nd retry (the old behavior).
    for retry := 0; retry <= 1; retry++ {
        for _, show := range shows {
            if c.Region != "" && retry < 1 {
                if c.Region == show.Country && strings.HasPrefix(show.Name, showname) {
                    return show, nil
                }
            } else {
                if strings.ToLower(show.Name) == strings.ToLower(showname) {
                    return show, nil
                } else if retry > 0 && strings.HasPrefix(show.Name, showname) {
                    return show, nil
                }
            }
        }
    }

    return Show{}, errors.New("Failed to match show in tvrage!")
}

func (c *Client) GetShow(show string) []Show {
    route := "/feeds/search.php"
    uri, err := url.Parse(c.BaseURI + route)
    if err != nil {
        log.Fatal(err)
    }

    query := url.Values{}
    query.Add("show", show)
    uri.RawQuery = query.Encode()

    xmldata := c.FetchData(uri)
    shows := c.GetShowsFromXML(xmldata)
    return shows
}

func (c *Client) GetShowsFromXML(xmldata []byte) []Show {
    result := Result{}
    err := xml.Unmarshal(xmldata, &result)
    if err != nil {
        log.Fatal(fmt.Printf("error: %s", string(xmldata)))
    }
    return result.Shows
}

func (c *Client) GetEpisodes(showid int) []Episode {
    route := "/feeds/episode_list.php"
    uri, err := url.Parse(c.BaseURI + route)
    if err != nil {
        log.Fatal(err)
    }

    var showstr string
    showstr = strconv.Itoa(showid)
    query := url.Values{}
    query.Add("sid", showstr)
    uri.RawQuery = query.Encode()

    xmldata := c.FetchData(uri)
    episodes := c.GetEpisodesFromXML(xmldata)
    return episodes
}

func (c *Client) GetEpisodesFromXML(xmldata []byte) []Episode {
    elist := EpisodeList{}
    err := xml.Unmarshal(xmldata, &elist)
    if err != nil {
        log.Fatal(fmt.Printf("error: %v", err))
    }

    episodes := make([]Episode, 0, 1)
    for _, s := range elist.Seasons {
        for _, e := range s.Episodes {
            e.Season = s.No
            episodes = append(episodes, e)
        }
    }

    return episodes
}

func (c *Client) FetchData(uri *url.URL) []byte {
    data, found := c.Cache.Get(uri.String())

    if ! found {
        if c.Debug {
            log.Print("cache miss: " + uri.String() + "\n")
        }
        response := c.Go(uri)
        if response.StatusCode == 200 {
            var err error
            data, err = ioutil.ReadAll(response.Body)
            if err != nil {
                log.Fatal(err)
            }
            c.Cache.Set(uri.String(), data, 0)
        }
    } else {
        if c.Debug {
            fmt.Println("cache hit: " + uri.String() + "\n")
        }
    }

    return data.([]byte)
}

func (c *Client) Go(uri *url.URL) *http.Response {
    request, err := http.NewRequest("GET", uri.String(), nil)
    if err != nil {
        log.Fatal(err)
    }

    response, err := c.Do(request)
    if err != nil {
        log.Fatal(err)
    }

    return response
}
