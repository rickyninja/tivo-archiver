package tvrage

import (
	"encoding/xml"
)

type Genre string

type Result struct {
	XMLName xml.Name `xml:"Results"`
	Shows   []Show   `xml:"show"`
}

type Show struct {
	ShowID         int     `xml:"showid"`
	Name           string  `xml:"name"`
	Link           string  `xml:"link"`
	Country        string  `xml:"country"`
	Started        int     `xml:"started"`
	Ended          int     `xml:"ended"`
	Seasons        int     `xml:"seasons"`
	Status         string  `xml:"status"`
	Classification string  `xml:"classification"`
	Genres         []Genre `xml:"genres>genre"`
}

func NewShow(showid int, name string, link string, country string, started int, ended int, status string,
	classification string, genres []Genre) *Show {
	return &Show{ShowID: showid, Name: name, Link: link, Country: country, Started: started,
		Ended: ended, Status: status, Classification: classification, Genres: genres}
}
