package tivo

import (
    "strings"
    "regexp"
    "fmt"
    "errors"
    "strconv"
    "tvrage"
)

func (c *Tivo) GetFilename(rage *tvrage.Client, detail *VideoDetail) (string, error) {
    filename := detail.Title

    if detail.IsEpisodic {
        rtv_show, err := rage.FindShow(detail.Title)
        if err != nil {
            return "", err
        }

        episodes := rage.GetEpisodes(rtv_show.ShowID)
        episode, err := c.FindRageEpisode(detail, episodes)
        if err != nil {
            fmt.Printf(err.Error())
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
            detail.EpisodeNumber = fmt.Sprintf("%d%.2d", episode.Season, episode.SeasonNum)
            filename += fmt.Sprintf(" %dx%.2d-%s", episode.Season, episode.SeasonNum, detail.EpisodeTitle)
        }
    }

    filename = strings.Replace(filename, "/", "-", -1)
    return filename, nil
}

func (c *Tivo) FindRageEpisode(detail *VideoDetail, episodes []tvrage.Episode) (tvrage.Episode, error) {
    for desperate := 0; desperate <= 3; desperate++ {
        for _, episode := range episodes {
            // normalize chars ’ vs ' etc.
            var rtb []byte
            var ttb []byte
            var tivotitle string
            var ragetitle string
            if len(episode.Title) == len(detail.EpisodeTitle) {
                rtb = []byte(episode.Title)
                ttb = []byte(detail.EpisodeTitle)
                for i := 0; i < len(rtb); i++ {
                    ord := int(rtb[i])
                    if ord < 32 || ord > 126 {
                        rtb = append(rtb[:i], rtb[i+1:]...)
                        ttb = append(ttb[:i], ttb[i+1:]...)
                    }
                }
                tivotitle = string(ttb[:])
                ragetitle = string(rtb[:])
            } else {
                tivotitle = detail.EpisodeTitle
                ragetitle = episode.Title
            }

            // As we become more desperate to find a match strip out non-word characters
            // to make a match more likely.
            if desperate >= 2 {
                re := regexp.MustCompile(`\W`)
                tivotitle = string(re.ReplaceAll([]byte(tivotitle), []byte("")))
                ragetitle = string(re.ReplaceAll([]byte(ragetitle), []byte("")))
            }

            // exact title match
            if ragetitle == tivotitle {
                return episode, nil
            // match against production code (Charmed)
            } else if detail.EpisodeNumber == episode.ProdNum {
                return episode, nil
            // exact title match if you add part_index inside parens to tivo title
            } else if detail.PartIndex > 0 && desperate == 0 {
                tt := fmt.Sprintf("%s (%d)", tivotitle, detail.PartIndex)
                if tt == ragetitle {
                    return episode, nil
                }
            // rage title contains tivo title
            } else if desperate == 1 && strings.Contains(ragetitle, tivotitle) {
                return episode, nil
            // tivo title contains rage title
            } else if desperate == 1 && strings.Contains(tivotitle, ragetitle) {
                detail.EpisodeNumber = fmt.Sprintf("%d%.2d", episode.Season, episode.SeasonNum)
                        return episode, nil
            } else if desperate == 1 {
                // try to match 'Kill Billie: Vol.2' with 'Kill Billie (2)'
                re := regexp.MustCompile(`\((\d+)\)/`)
                captures := re.FindStringSubmatch(ragetitle)

                if captures != nil {
                    rt := string(re.ReplaceAll([]byte(ragetitle), []byte("")))
                    sequel := captures[1]
                    rt = strings.TrimSpace(rt)
                    if strings.Contains(tivotitle, rt) && strings.Contains(tivotitle, sequel) {
                        return episode, nil
                    }
                } else if strings.Contains(ragetitle, " and ") && strings.Contains(tivotitle, "&") {
                    tt := strings.Replace(tivotitle, "&", "and", -1)
                    if ragetitle == tt {
                        return episode, nil
                    }
                } else if strings.Contains(ragetitle, "&") && strings.Contains(tivotitle, " and ") {
                    tt := strings.Replace(tivotitle, " and ", " & ", -1)
                    if ragetitle == tt {
                        return episode, nil
                    }
                }
            }
        }
    }

    return tvrage.Episode{}, errors.New("Failed to ID season and episode!")
}

func (c *Tivo) GetPymeta(detail *VideoDetail) string {
    pymeta := fmt.Sprintf("title: %s\n", detail.Title)
    pymeta += fmt.Sprintf("seriestitle: %s\n", detail.SeriesTitle)
    pymeta += fmt.Sprintf("isEpisode: %t\n", detail.IsEpisode)

    if detail.Description != "" {
        pymeta += fmt.Sprintf("description: %s\n", detail.Description)
    }

    for _, genre := range detail.SeriesGenres {
        pymeta += fmt.Sprintf("vProgramGenre: %s\n", genre)
    }

    for _, actor := range detail.Actors {
        pymeta += fmt.Sprintf("vActor: %s\n", actor)
    }

    for _, guest := range detail.GuestStars {
        pymeta += fmt.Sprintf("vGuestStar: %s\n", guest)
    }

    for _, director := range detail.Directors {
        pymeta += fmt.Sprintf("vDirector: %s\n", director)
    }

    for _, exec := range detail.ExecProducers {
        pymeta += fmt.Sprintf("vExecProducer: %s\n", exec)
    }

    for _, prod := range detail.Producers {
        pymeta += fmt.Sprintf("vProducer: %s\n", prod)
    }

    for _, writer := range detail.Writers {
        pymeta += fmt.Sprintf("vWriter: %s\n", writer)
    }

    for _, host := range detail.Hosts {
        pymeta += fmt.Sprintf("vHost: %s\n", host)
    }

    for _, chore := range detail.Choreographers {
        pymeta += fmt.Sprintf("vChore: %s\n", chore)
    }

    if detail.PartCount > 0 && detail.PartIndex > 0 {
        pymeta += fmt.Sprintf("partCount: %d\n", detail.PartCount)
        pymeta += fmt.Sprintf("partIndex: %d\n", detail.PartIndex)
    }

    if detail.IsEpisodic {
        if detail.EpisodeTitle != "" {
            pymeta += fmt.Sprintf("episodeTitle: %s\n", detail.EpisodeTitle)
        }

        if detail.EpisodeNumber != "" {
            pymeta += fmt.Sprintf("episodeNumber: %s\n", detail.EpisodeNumber)
        }

        if detail.OriginalAirDate != "" {
            // Alter the oad by replacing hour with partIndex. This sorts better in the tivo ui.
            oad := detail.OriginalAirDate
            pi := fmt.Sprintf("%.2d", detail.PartIndex)
            D := strings.Split(oad, "T")
            date, time := D[0], D[1]
            T := strings.Split(time, ":")
            min, sec := T[1], T[2]
            time = strings.Join([]string{pi, min, sec}, ":")
            oad = strings.Join([]string{date, time}, "T")

            pymeta += fmt.Sprintf("originalAirDate: %s\n", oad)
        }
    } else {
        pymeta += fmt.Sprintf("movieYear: %d\n", detail.MovieYear)
    }

    return pymeta
}
