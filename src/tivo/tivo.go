package tivo

import (
    "net/http"
    "net/http/cookiejar"
    "net/url"
    "io/ioutil"
    cache "github.com/robfig/go-cache"
    "os"
    "fmt"
    "log"
    "time"
    "encoding/xml"
    "crypto/md5"
    "crypto/tls"
    "regexp"
    "strings"
    "strconv"
)

type Tivo struct {
    Debug bool
    BaseURI string
    MAK int
    Realm string
    Login string
    Host string
    Protocol string
    UseCache bool
    Cache *cache.Cache
    CacheFile string
    *http.Client
}

func New(host string, login string, protocol string, mak int, cachefile string) *Tivo {
    c := cache.New(time.Second * 3600, time.Minute * 60)
    if _, err := os.Stat(cachefile); err == nil {
        err := c.LoadFile(cachefile)
        if err != nil {
            log.Fatal("Failed to load cache from file: " + cachefile)
        }
    }

    // downloads get interrupted when the timeout is too short
    timeout := time.Duration(0 * time.Second)
    tr := &http.Transport{
        TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
    }

    jar, err := cookiejar.New(&cookiejar.Options{})
    if err != nil {
        log.Fatal(err.Error())
    }

    client := &http.Client{
        Timeout: timeout,
        Transport: tr,
        Jar: jar,
    }

    baseuri := fmt.Sprintf("%s://%s/TiVoConnect", protocol, host)
    return &Tivo{
        false,
        baseuri,
        mak,
        "TiVo DVR",
        login,
        host,
        protocol,
        false,
        c,
        cachefile,
        client,
    }
}

func (c *Tivo) FetchData(uri *url.URL) []byte {
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

func (c *Tivo) QueryContainer(param map[string]string) []ContainerItem {
    param["Command"] = "QueryContainer"
    uri := c.GetURI(param)

    xmldata := c.FetchData(uri)
    containers := c.GetContainerFromXML(xmldata)
    return containers
}

func (c *Tivo) GetContainerFromXML(xmldata []byte) []ContainerItem {
    container := &Container{}
    err := xml.Unmarshal(xmldata, container)
    if err != nil {
        log.Fatal(fmt.Printf("error: %v", err))
    }
    return container.Items
}

func (c *Tivo) GetDetail(ci ContainerItem) VideoDetail {
    uri, err := url.Parse(ci.VideoDetailsURL)
    if err != nil {
        log.Fatal("Failed to parse url: " + err.Error())
    }

    xmldata := c.FetchData(uri)
    detail := c.GetDetailFromXML(xmldata)
    detail = c.ScrubDetail(detail)
    return detail
}

func (c *Tivo) ScrubDetail(detail VideoDetail) VideoDetail {
    detail.Description = strings.TrimSuffix(detail.Description, " Copyright Tribune Media Services, Inc.")
    return detail
}

func (c *Tivo) GetDetailFromXML(xmldata []byte) VideoDetail {
    root := &VideoDetailRoot{}
    err := xml.Unmarshal(xmldata, root)
    if err != nil {
        log.Fatal(fmt.Printf("error: %v", err))
    }
    return root.Showing
}

func (c *Tivo) DigestAuth(response *http.Response) *http.Response {
    var err error
    var request *http.Request

    request, err = http.NewRequest("GET", response.Request.URL.String(), nil)
    if err != nil {
        log.Fatal(err)
    }

    digest := fmt.Sprintf(`Digest username="%s", `, c.Login)

    var re *regexp.Regexp
    var result []string

    re, err = regexp.Compile(`Digest realm="([^"]+)"`)
    result = re.FindStringSubmatch(response.Header.Get("Www-Authenticate"))
    realm := result[1]
    digest += fmt.Sprintf(`realm="%s", `, realm)

    re, err = regexp.Compile(`nonce="([^"]+)"`)
    result = re.FindStringSubmatch(response.Header.Get("Www-Authenticate"))
    nonce := result[1]
    digest += fmt.Sprintf(`nonce="%s", `, nonce)

    re, err = regexp.Compile(`qop="([^"]+)"`)
    result = re.FindStringSubmatch(response.Header.Get("Www-Authenticate"))
    qop := result[1]
    digest += fmt.Sprintf(`qop=%s, `, qop)

    digest += `algorithm="MD5", `

    digest += fmt.Sprintf(`uri="%s", `, response.Request.URL.RequestURI())

    nc := fmt.Sprintf("%08X", 1)
    digest += fmt.Sprintf(`nc="%s", `, nc)

    cnonce := fmt.Sprintf("%8x", time.Now().Unix())
    digest += fmt.Sprintf(`cnonce="%s", `, cnonce)

    HA1 := c.HA1(c.Login, realm, strconv.Itoa(c.MAK))
    HA2 := c.HA2(response.Request.Method, response.Request.URL.RequestURI())

    resp := c.GetDigestResponse(HA1, HA2, nonce, nc, cnonce, qop)
    digest += fmt.Sprintf(`response="%s"`, resp)


    request.Header.Add("Authorization", digest)


    //response.Write(os.Stdout)
    //fmt.Println("")
    //request.Write(os.Stdout)

    authresponse, err := c.Do(request)
    if err != nil {
        log.Fatal(err)
    }
    return authresponse
}

func (c *Tivo) GetDigestResponse(ha1, ha2, nonce, nc, cnonce, qop string) string {
    s := []string{ha1, nonce, nc, cnonce, qop, ha2}
    return h(strings.Join(s, ":"))
}

func (c *Tivo) HA2(method string, uri string) string {
    s := []string{method, uri}
    return h(strings.Join(s, ":"))
}

func (c *Tivo) HA1(username string, realm string, password string) string {
    s := []string{username, realm, password}
    return h(strings.Join(s, ":"))
}

func h(str string) string {
    sum := md5.Sum([]byte(str))
    return fmt.Sprintf("%x", sum)
}

func (c *Tivo) GetURI(param map[string]string) *url.URL {
    uri, err := url.Parse(c.BaseURI)
    if err != nil {
        log.Fatal(err)
    }

    p := url.Values{}
    for k, v := range param {
        p.Add(k, v)
    }
    uri.RawQuery = p.Encode()
    return uri
}

func (c *Tivo) Go(uri *url.URL) *http.Response {
    request, err := http.NewRequest("GET", uri.String(), nil)
    if err != nil {
        log.Fatal(err)
    }

    response, err := c.Do(request)
    if err != nil {
        log.Fatal(err)
    }

    response = c.CheckAuth(response)

    return response
}

// Can the Digest auth be hooked into the transport?
func (c *Tivo) CheckAuth(response *http.Response) *http.Response {
    if response.StatusCode == 200 {
        return response
    } else if response.StatusCode == 401 {
        for i := 1; i < 10; i++ {
            authresponse := c.DigestAuth(response)
            if authresponse.StatusCode == 200 {
                return authresponse
            } else if authresponse.StatusCode == 503 {
                log.Print(fmt.Sprintf("%s : attempt %d/10, retrying\n", authresponse.Status, i))
                time.Sleep(2 * time.Second)
                continue
            } else {
                log.Fatal("Failed to authenticate to the tivo: " + authresponse.Status)
            }
        }
    }

    return response
}

func (c *Tivo) WriteCache() {
    err := c.Cache.SaveFile(c.CacheFile)
    if err != nil {
        log.Println(err.Error())
    }
}

