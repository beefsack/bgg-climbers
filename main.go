package main

import (
	"encoding/csv"
	"fmt"
	"io"
	"log"
	"os"
	"path"
	"sort"
	"strconv"
	"strings"
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

const (
	RankWidth         = 4
	AverageWidth      = 5
	BayesAverageWidth = AverageWidth
	UsersRatedWidth   = 6
	EmptyStr          = "N/A"
	MaxDisplaced      = 1
)

var (
	RankFormat         = fmt.Sprintf("%%%dd", RankWidth)
	AverageFormat      = fmt.Sprintf("%%%ds", AverageWidth)
	BayesAverageFormat = AverageFormat
	UsersRatedFormat   = fmt.Sprintf("%%%ds", UsersRatedWidth)
	UnrankedText       = fmt.Sprintf(fmt.Sprintf("%%%ds", RankWidth), EmptyStr)
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

func (r Record) RankDescription() string {
	if r.Rank == 0 {
		return UnrankedText
	}
	return fmt.Sprintf(RankFormat, r.Rank)
}

func (r Record) Description() string {
	return fmt.Sprintf(
		fmt.Sprintf("%%s  %s  %s  %s", AverageFormat, BayesAverageFormat, UsersRatedFormat),
		r.RankDescription(),
		StrOrNA(r.Average),
		StrOrNA(r.BayesAverage),
		StrOrNA(r.UsersRated),
	)
}

func FileTitle(s string) string {
	return path.Base(s[:len(s)-len(path.Ext(s))])
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
	return float64(oldRank) / float64(newRank)
}

func MaxLen(a, b string) int {
	l := len(a)
	if bl := len(b); bl > l {
		l = bl
	}
	return l
}

func StrOrNA(s string) string {
	if s == "" {
		return EmptyStr
	}
	return s
}

func (g Game) RatingString() string {
	return fmt.Sprintf(
		"[size=18][b][COLOR=#009900]↗ %s[/COLOR][/b][/size]",
		g.RatingPercString(),
	)
}

func (g Game) RatingPercString() string {
	return fmt.Sprintf("%.2f%%", (g.ClimbScore-1)*100)
}

func DisplacedString(displaced []Game) string {
	if len(displaced) == 0 {
		return ""
	}

	links := []string{}
	for i, d := range displaced {
		if i == MaxDisplaced {
			break
		}
		links = append(links, DisplacedStringLink(d))
	}

	othersStr := ""
	if l := len(displaced); l > MaxDisplaced {
		others := l - MaxDisplaced
		suffix := ""
		if others > 1 {
			suffix = "s"
		}
		othersStr = fmt.Sprintf(
			" and %d other%s",
			others,
			suffix,
		)
	}

	return fmt.Sprintf(
		"Displaced %s%s",
		strings.Join(links, ", "),
		othersStr,
	)
}

func DisplacedStringLink(displaced Game) string {
	return fmt.Sprintf(
		"[thing=%s][/thing] (%d -> %d)",
		displaced.New.ID,
		displaced.Old.Rank,
		displaced.New.Rank,
	)
}

func (g Game) Description(oldTitle, newTitle string, displaced []Game) string {
	titleLen := MaxLen(oldTitle, newTitle)
	return fmt.Sprintf(
		fmt.Sprintf(`%%s
[c]
[BGCOLOR=#000000][COLOR=#FFFFFF][b]%%%ds  %%%ds  %%%ds  %%%ds  %%%ds[/b][/COLOR][/BGCOLOR]
[BGCOLOR=#D8D8D8]%%%ds  %%s[/BGCOLOR]
%%%ds  %%s
[/c]
%%s`, titleLen, RankWidth, AverageWidth, BayesAverageWidth, UsersRatedWidth, titleLen, titleLen),
		g.RatingString(),
		"",
		"Rank",
		"Avg",
		"Bay",
		"#Rtg",
		oldTitle,
		g.Old.Description(),
		newTitle,
		g.New.Description(),
		DisplacedString(displaced),
	)
}

func (g Game) ToCSVRecord(oldTitle, newTitle string, displaced []Game) []string {
	id := g.Old.ID
	if id == "" {
		id = g.New.ID
	}
	name := g.New.Name
	if name == "" {
		name = g.Old.Name
	}
	return []string{
		id,
		name,
		g.Description(oldTitle, newTitle, displaced),
		fmt.Sprintf("%f", g.ClimbScore),
		g.Old.RankString(),
		g.New.RankString(),
		g.Old.Name,
		g.Old.Year,
		g.Old.Average,
		g.Old.BayesAverage,
		g.Old.UsersRated,
		g.Old.URL,
		g.Old.Thumbnail,
		g.New.Name,
		g.New.Year,
		g.New.Average,
		g.New.BayesAverage,
		g.New.UsersRated,
		g.New.URL,
		g.New.Thumbnail,
	}
}

type GamesByClimbScore []Game

func (g GamesByClimbScore) Len() int           { return len(g) }
func (g GamesByClimbScore) Swap(i, j int)      { g[i], g[j] = g[j], g[i] }
func (g GamesByClimbScore) Less(i, j int) bool { return g[i].ClimbScore < g[j].ClimbScore }

type GamesByUsersRated []Game

func (g GamesByUsersRated) Len() int      { return len(g) }
func (g GamesByUsersRated) Swap(i, j int) { g[i], g[j] = g[j], g[i] }
func (g GamesByUsersRated) Less(i, j int) bool {
	iur, _ := strconv.Atoi(g[i].New.UsersRated)
	jur, _ := strconv.Atoi(g[j].New.UsersRated)
	return iur < jur
}

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
		"Name",
		"Description",
		"Rank 1 / Rank 2",
		"Rank 1",
		"Rank 2",
		"Name 1",
		"Year 1",
		"Average 1",
		"Bayes average 1",
		"Users rated 1",
		"URL 1",
		"Thumbnail 1",
		"Name 2",
		"Year 2",
		"Average 2",
		"Bayes average 2",
		"Users rated 2",
		"URL 2",
		"Thumbnail 2",
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
	gamesSlice := GamesByClimbScore{}
	gamesByOldRank := map[int]Game{}
	for _, g := range games {
		g.ClimbScore = g.CalcClimbScore(maxRank / 2)
		gamesSlice = append(gamesSlice, g)
		if g.Old.Rank != 0 {
			gamesByOldRank[g.Old.Rank] = g
		}
	}
	sort.Sort(sort.Reverse(gamesSlice))

	// Output
	oldTitle := FileTitle(os.Args[1])
	newTitle := FileTitle(os.Args[2])
	for _, g := range gamesSlice {
		if g.Old.Rank == 0 || g.New.Rank == 0 {
			// Ignore games which gained or lost their rank
			continue
		}
		if ur, err := strconv.Atoi(g.New.UsersRated); err != nil || ur < 100 {
			continue
		}
		// Find out if it displaced any games.
		displaced := []Game{}
		for r := g.New.Rank + 1; r <= g.Old.Rank; r++ {
			og := gamesByOldRank[r]
			if og.New.Rank != 0 && og.New.Rank > g.New.Rank && og.New.Rank > og.Old.Rank {
				displaced = append(displaced, og)
			}
		}
		sort.Sort(sort.Reverse(GamesByUsersRated(displaced)))
		if err := w.Write(g.ToCSVRecord(oldTitle, newTitle, displaced)); err != nil {
			stderr.Fatalf("Unable to write CSV row, %s", err)
		}
	}
}
