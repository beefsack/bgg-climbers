package main

import (
	"encoding/csv"
	"fmt"
	"io"
	"log"
	"math"
	"os"
	"sort"
	"strconv"
)

const (
	SrcID = iota
	SrcName
	SrcYear
	SrcRank
	SrcAverage
	SrcBayesAverage
	SrcUsersRated
	SrcURL
	SrcThumbnail
)

type Record struct {
	ID           string
	Name         string
	Year         string
	Rank         int
	Average      string
	BayesAverage string
	UsersRated   string
	URL          string
	Thumbnail    string
}

func (r Record) RankString() string {
	if r.Rank == 0 {
		return ""
	}
	return strconv.Itoa(r.Rank)
}

type Game struct {
	Old, New   Record
	ClimbScore float64
}

func (g Game) CalcClimbScore(fallbackRank int) float64 {
	oldRank := g.Old.Rank
	if oldRank == 0 {
		oldRank = fallbackRank
	}
	newRank := g.New.Rank
	if newRank == 0 {
		newRank = fallbackRank
	}
	return math.Log(float64(oldRank)) - math.Log(float64(newRank))
}

func (g Game) ToCSVRecord(fallbackRank int) []string {
	id := g.Old.ID
	if id == "" {
		id = g.New.ID
	}
	return []string{
		id,
		g.Old.Name,
		g.Old.Year,
		g.Old.RankString(),
		g.Old.Average,
		g.Old.BayesAverage,
		g.Old.UsersRated,
		g.Old.URL,
		g.Old.Thumbnail,
		g.New.Name,
		g.New.Year,
		g.New.RankString(),
		g.New.Average,
		g.New.BayesAverage,
		g.New.UsersRated,
		g.New.URL,
		g.New.Thumbnail,
		fmt.Sprintf("%f", g.ClimbScore),
	}
}

type Games []Game

func (g Games) Len() int           { return len(g) }
func (g Games) Swap(i, j int)      { g[i], g[j] = g[j], g[i] }
func (g Games) Less(i, j int) bool { return g[i].ClimbScore < g[j].ClimbScore }

func ParseRecord(record []string) (Record, error) {
	if len(record) <= SrcThumbnail {
		return Record{}, fmt.Errorf("record too short: %#v", record)
	}
	rank, err := strconv.Atoi(record[SrcRank])
	if err != nil {
		return Record{}, fmt.Errorf("unable to parse rank '%s', %s", err)
	}
	return Record{
		ID:           record[SrcID],
		Name:         record[SrcName],
		Year:         record[SrcYear],
		Rank:         rank,
		Average:      record[SrcAverage],
		BayesAverage: record[SrcBayesAverage],
		UsersRated:   record[SrcUsersRated],
		URL:          record[SrcURL],
		Thumbnail:    record[SrcThumbnail],
	}, nil
}

func main() {
	stderr := log.New(os.Stderr, "", 0)
	if len(os.Args) != 3 {
		stderr.Fatalf("Expected two CSV file arguments to compare")
	}
	w := csv.NewWriter(os.Stdout)
	defer w.Flush()
	if err := w.Write([]string{
		"ID",
		"Name 1",
		"Year 1",
		"Rank 1",
		"Average 1",
		"Bayes average 1",
		"Users rated 1",
		"URL 1",
		"Thumbnail 1",
		"Name 2",
		"Year 2",
		"Rank 2",
		"Average 2",
		"Bayes average 2",
		"Users rated 2",
		"URL 2",
		"Thumbnail 2",
		"ln(Rank 1) - ln(Rank 2)",
	}); err != nil {
		stderr.Fatalf("Error writing header, %v", err)
	}

	games := map[string]Game{}
	maxRank := 0

	// Parse old file
	oldF, err := os.Open(os.Args[1])
	if err != nil {
		stderr.Fatalf("Unable to open '%s', %s", os.Args[1], err)
	}
	oldR := csv.NewReader(oldF)
	hasReadHeader := false
	for {
		record, err := oldR.Read()
		if err == io.EOF {
			break
		} else if err != nil {
			stderr.Fatalf("Unable to read line from '%s', %s", os.Args[1], err)
		}
		if !hasReadHeader {
			hasReadHeader = true
			continue
		}
		if len(record) <= SrcThumbnail {
			continue
		}
		g := games[record[SrcID]]
		if g.Old, err = ParseRecord(record); err != nil {
			stderr.Fatalf("Unable to parse record, %s", err)
		}
		if g.Old.Rank > maxRank {
			maxRank = g.Old.Rank
		}
		games[record[SrcID]] = g
	}
	if err := oldF.Close(); err != nil {
		stderr.Fatalf("Unable to close '%s', %s", os.Args[1], err)
	}

	// Parse new file
	newF, err := os.Open(os.Args[2])
	if err != nil {
		stderr.Fatalf("Unable to open '%s', %s", os.Args[2], err)
	}
	newR := csv.NewReader(newF)
	hasReadHeader = false
	for {
		record, err := newR.Read()
		if err == io.EOF {
			break
		} else if err != nil {
			stderr.Fatalf("Unable to read line from '%s', %s", os.Args[2], err)
		}
		if !hasReadHeader {
			hasReadHeader = true
			continue
		}
		if len(record) <= SrcThumbnail {
			continue
		}
		g := games[record[SrcID]]
		if g.New, err = ParseRecord(record); err != nil {
			stderr.Fatalf("Unable to parse record, %s", err)
		}
		if g.New.Rank > maxRank {
			maxRank = g.New.Rank
		}
		games[record[SrcID]] = g
	}
	if err := newF.Close(); err != nil {
		stderr.Fatalf("Unable to close '%s', %s", os.Args[2], err)
	}

	// Set climb score and build games slice
	gamesSlice := Games{}
	for _, g := range games {
		g.ClimbScore = g.CalcClimbScore(maxRank)
		gamesSlice = append(gamesSlice, g)
	}
	sort.Sort(sort.Reverse(gamesSlice))

	// Output
	for _, g := range gamesSlice {
		if err := w.Write(g.ToCSVRecord(maxRank + 1)); err != nil {
			stderr.Fatalf("Unable to write CSV row, %s", err)
		}
	}
}
