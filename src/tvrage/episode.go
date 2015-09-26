package tvrage

import (
    "encoding/xml"
)

type EpisodeList struct {
    XMLName xml.Name `xml:"Show"`
    Name string `xml:"name"`
    Seasons []Season `xml:"Episodelist>Season"`
    //TotalSeasons int `xml:"totalseasons"` // why does this panic?
}

type Season struct {
    No int `xml:"no,attr"`
    Episodes []Episode `xml:"episode"`
}

type Episode struct {
    // This doesn't reset to 1 at the start of each season.
    EpNum int `xml:"epnum"`

    // This is the episode number for the current season.  This one resets to 1 each season.
    SeasonNum int `xml:"seasonnum"`

    // This isn't a tvrage attribute, but I'm adding it as a convenience.  This attribute
    // is what I expected the seasonnum to be.
    // error: xml: Season>no chain not valid with attr flag2015/05/24 18:58:32 52 <nil>
    Season int

    // This is the production code.  It's poorly named since it can contain non-numbers.
    ProdNum string `xml:"prodnum"`

    AirDate string `xml:"airdate"`

    // This is a link to the tvrage site with info on the episode.
    Link string `xml:"link"`

    // This is the episode's title.
    Title string `xml:"title"`
}

func NewEpisode(season int, epnum int, seasonnum int, prodnum string, airdate string, link string, title string) *Episode {
    return &Episode{
        Season: season,
        EpNum: epnum,
        SeasonNum: seasonnum,
        ProdNum: prodnum,
        AirDate: airdate,
        Link: link,
        Title: title,
    }
}
