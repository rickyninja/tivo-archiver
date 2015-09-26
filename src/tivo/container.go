package tivo

import (
    "encoding/xml"
)

type Container struct {
    XMLName xml.Name `xml:"TiVoContainer"`
    Items []ContainerItem `xml:"Item"`
}


type ContainerItem struct {
    // I'd like InProgess to be a bool. method wrapper?
    InProgress string `xml:"Details>InProgress"`
    ContentType string `xml:"Details>ContentType"`
    SourceFormat string `xml:"Details>SourceFormat"`
    Title string `xml:"Details>Title"`
    SourceSize int64 `xml:"Details>SourceSize"`
    Duration int `xml:"Details>Duration"`

    // This value will be in hex, convert to int on the fly.
    // <CaptureDate>0x51ED10AE</CaptureDate>
    // Does Go have a better date format type for this?
    // method wrapper?
    CaptureDate string `xml:"Details>CaptureDate"`
    EpisodeTitle string `xml:"Details>EpisodeTitle"`
    Description string `xml:"Details>Description"`
    SourceChannel string `xml:"Details>SourceChannel"`
    SourceStation string `xml:"Details>SourceStation"`

    // I'd like HighDefinition to be a bool. method wrapper?
    HighDefinition string `xml:"Details>HighDefinition"`
    ProgramID string `xml:"Details>ProgramId"`
    SeriesID string `xml:"Details>SeriesId"`
    EpisodeNumber int `xml:"Details>EpisodeNumber"`
    ByteOffset int `xml:"Details>ByteOffset"`
    ContentURL string `xml:"Links>Content>Url"`
    ContentTypeURL string `xml:"Links>Content>ContentType"`
    CustomIconURL string `xml:"Links>CustomIcon>Url"`
    VideoDetailsURL string `xml:"Links>TiVoVideoDetails>Url"`
    Detail VideoDetail
}

type Person string
type Genre string

type VideoDetailRoot struct {
    XMLName xml.Name `xml:"TvBusEnvelope"`
    Showing VideoDetail
}

type VideoDetail struct {
    XMLName xml.Name `xml:"showing"`
    IsEpisode bool `xml:"program>isEpisode"`
    IsEpisodic bool `xml:"program>series>isEpisodic"`
    Title string `xml:"program>title"`
    SeriesTitle string `xml:"program>series>seriesTitle"`

    // hook to remove tribute media blurb? method wrapper?
    Description string `xml:"program>description"`

    // is this path right?
    OriginalAirDate string `xml:"program>originalAirDate"`

    // episode Number is frequently incorrect in tivo metadata,
    // prefer the value acquired from tvrage.
    EpisodeNumber string  `xml:"program>episodeNumber"`
    EpisodeTitle string  `xml:"program>episodeTitle"`

    Time string `xml:"time"`
    MovieYear int `xml:"program>movieYear"`
    PartCount int `xml:"partCount"`
    PartIndex int `xml:"partIndex"`

    SeriesGenres []Genre `xml:"program>series>vSeriesGenre>element"`
    Actors []Person `xml:"program>vActor>element"`
    GuestStars []Person `xml:"program>vGuestStar>element"`
    Directors []Person `xml:"program>vDirector>element"`
    ExecProducers []Person `xml:"program>vExecProducer>element"`
    Producers []Person `xml:"program>vProducer>element"`
    Choreographers []Person `xml:"program>vChoreographer>element"`
    Writers []Person `xml:"program>vWriter>element"`
    Hosts []Person `xml:"program>vHost>element"`
}
