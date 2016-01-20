package tivo

import (
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"tvmaze"
)

func (c *Tivo) GetFilename(maze *tvmaze.Client, detail *VideoDetail) (string, error) {
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

		episode, err := c.FindEpisode(detail, episodes)
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

func (c *Tivo) FindEpisode(detail *VideoDetail, episodes []tvmaze.Episode) (tvmaze.Episode, error) {
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
