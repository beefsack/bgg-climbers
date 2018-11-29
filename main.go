package main

import (
	"encoding/csv"
	"flag"
	"fmt"
	"io"
	"log"
	"math"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
)

// Source file field offsets.
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

// FileDateFormat is the format of the date in the file name.
const FileDateFormat = "2006-01-02"

// Formatting constants.
const (
	RankWidth         = 5
	AverageWidth      = 5
	NewAverageWidth   = AverageWidth
	BayesAverageWidth = 6
	UsersRatedWidth   = 6
	ChangeWidth       = 8
	ColorTagWidth     = 23
	EmptyStr          = "N/A"
)

// Formatting variables.
var (
	RankFormat         = fmt.Sprintf("%%%ds", RankWidth)
	AverageFormat      = fmt.Sprintf("%%%ds", AverageWidth)
	NewAverageFormat   = fmt.Sprintf("%%%ds", NewAverageWidth)
	BayesAverageFormat = fmt.Sprintf("%%%d.3f", BayesAverageWidth)
	UsersRatedFormat   = fmt.Sprintf("%%%ds", UsersRatedWidth)
	ChangeFormat       = fmt.Sprintf("%%%ds", ChangeWidth+ColorTagWidth)
	UnrankedText       = fmt.Sprintf(fmt.Sprintf("%%%ds", RankWidth), EmptyStr)
	TitleLen           = len(FileDateFormat)
)

// File is a parsed file.
type File struct {
	Path    string
	Date    time.Time
	MaxRank int
	Records []Record
}

// ParseFileDate parses the date of the file from the filename.
func ParseFileDate(p string) (time.Time, error) {
	return time.Parse(FileDateFormat, strings.TrimSuffix(path.Base(p), ".csv"))
}

// ParseFile parses a bgg-ranking-historicals file.
func ParseFile(p string) (File, error) {
	f := File{
		Path: p,
	}

	date, err := ParseFileDate(p)
	if err != nil {
		return f, fmt.Errorf("Unable parse date from '%s', %s", p, err)
	}
	f.Date = date

	handle, err := os.Open(p)
	if err != nil {
		return f, fmt.Errorf("Unable to open '%s', %s", p, err)
	}
	defer handle.Close()

	csvReader := csv.NewReader(handle)
	hasReadHeader := false
	for {
		record, err := csvReader.Read()
		if err == io.EOF {
			break
		} else if err != nil {
			return f, fmt.Errorf("Unable to read line from '%s', %s", p, err)
		}
		if !hasReadHeader {
			hasReadHeader = true
			continue
		}

		if len(record) <= SrcThumbnail {
			continue
		}

		parsedRecord, err := ParseRecord(record)
		if err != nil {
			return f, fmt.Errorf("Unable to parse record, %s", err)
		}

		if parsedRecord.Rank > f.MaxRank {
			f.MaxRank = parsedRecord.Rank
		}

		f.Records = append(f.Records, parsedRecord)
	}

	return f, nil
}

// Record is a record in a bgg-ranking-historicals file.
type Record struct {
	ID           string
	Name         string
	Year         string
	Rank         int
	Average      string
	BayesAverage float64
	UsersRated   string
	URL          string
	Thumbnail    string
}

// RankString converts a BGG rank to a string, using an empty string to
// represent no rank.
func (r Record) RankString() string {
	if r.Rank == 0 {
		return ""
	}
	return strconv.Itoa(r.Rank)
}

// Description outputs the record rank details.
func (r Record) Description() string {
	return fmt.Sprintf(
		fmt.Sprintf("%%s  %s  %s  %s  %v  %s", AverageFormat, NewAverageFormat, BayesAverageFormat, UsersRatedFormat, ChangeFormat),
		StrOrNA(r.RankString()),
		StrOrNA(r.Average),
		r.BayesAverage,
		StrOrNA(r.UsersRated),
	)
}

// A Game is a collection of GameRecords
type Game struct {
	Records []GameRecord
}

// ClimbScore is a ratio of rank movement in this game's most recent period.
func (g Game) ClimbScore() float64 {
	return ClimbScore(g.Records[1].BayesAverage, g.Records[0].BayesAverage)
}

// ClimbScore is a ratio of rank movement.
func ClimbScore(oldBayes, newBayes float64) float64 {
	return newBayes - oldBayes
}

func NewRatingAverage(oldRecord, newRecord Record) float64 {
	oldRatings, _ := strconv.ParseFloat(oldRecord.UsersRated, 64)
	oldAverage, _ := strconv.ParseFloat(oldRecord.Average, 64)
	newRatings, _ := strconv.ParseFloat(newRecord.UsersRated, 64)
	newAverage, _ := strconv.ParseFloat(newRecord.Average, 64)
	if oldAverage == newAverage || oldRatings == newRatings {
		return newAverage
	}
	return math.Max(
		0,
		math.Min(
			10,
			(newAverage*newRatings-oldAverage*oldRatings)/(newRatings-oldRatings),
		),
	)
}

// StrOrNA replaces empty strings with "N/A"
func StrOrNA(s string) string {
	if s == "" {
		return EmptyStr
	}
	return s
}

// ClimbScoreString is the climb score as a percentage with color and an arrow.
func ClimbScoreString(climbScore float64) string {
	arrow := "-"
	color := "555555"
	if climbScore > 0 {
		arrow = "↗"
		color = "009900"
	} else if climbScore < 0 {
		arrow = "↘"
		color = "990000"
	}
	return fmt.Sprintf(
		"[COLOR=#%s]%s %.3f[/COLOR]",
		color,
		arrow,
		math.Abs(climbScore),
	)
}

// ClimbScorePercString represents the climb ratio as a percentage.
func ClimbScorePercString(climbScore float64) string {
	perc := climbScore
	if perc > 1 {
		perc = 1 / perc
	}
	return fmt.Sprintf("%.2f%%", (1-perc)*100)
}

// DescTableTitle is the title row of the table.
var DescTableTitle = fmt.Sprintf(
	fmt.Sprintf(
		"[BGCOLOR=#000000][COLOR=#FFFFFF][b]%%%ds  %%%ds  %%%ds  %%%ds  %%%ds  %%%ds  %%%ds[/b][/COLOR][/BGCOLOR]",
		TitleLen,
		RankWidth,
		AverageWidth,
		NewAverageWidth,
		BayesAverageWidth,
		UsersRatedWidth,
		ChangeWidth,
	),
	"",
	"Rank",
	"Avg",
	"New",
	"Bay",
	"#Rtg",
	"Chng",
)

// Description outputs a climb score and table of historicals.
func (g Game) Description() string {
	lastRecord := g.Records[len(g.Records)-1]
	return fmt.Sprintf(`[size=18][b]%s[/b][/size]

[size=10]%s since %s[/size]
[c]
%s
%s
[/c]`,
		ClimbScoreString(ClimbScore(g.Records[1].BayesAverage, g.Records[0].BayesAverage)),
		ClimbScoreString(ClimbScore(lastRecord.BayesAverage, g.Records[0].BayesAverage)),
		lastRecord.Date.Format(FileDateFormat),
		DescTableTitle,
		g.DescriptionRows(),
	)
}

// DescriptionRows outputs the rows in the Description table
func (g Game) DescriptionRows() string {
	l := len(g.Records)
	lines := make([]string, l)
	for i := 0; i < l; i++ {
		lines[i] = g.DescriptionRow(l - i - 1)
	}
	return strings.Join(lines, "\n")
}

// DescriptionRow outputs a specific row in the Description table
func (g Game) DescriptionRow(offset int) string {
	record := g.Records[offset]

	newAverage := ""
	if offset < len(g.Records)-1 {
		newAverage = fmt.Sprintf("%.2f", NewRatingAverage(g.Records[offset+1].Record, record.Record))
	}

	row := fmt.Sprintf(
		fmt.Sprintf("%%s  %s  %s  %s  %s  %s  %s", RankFormat, AverageFormat, NewAverageFormat, BayesAverageFormat, UsersRatedFormat, ChangeFormat),
		record.Date.Format(FileDateFormat),
		StrOrNA(record.Record.RankString()),
		StrOrNA(record.Record.Average),
		newAverage,
		record.Record.BayesAverage,
		StrOrNA(record.Record.UsersRated),
		g.ClimbScoreString(offset),
	)
	if offset == 0 {
		row = fmt.Sprintf("[b][BGCOLOR=#FFFF80]%s[/BGCOLOR][/b]", row)
	} else if (len(g.Records)-offset)%2 == 1 {
		row = fmt.Sprintf("[BGCOLOR=#D8D8D8]%s[/BGCOLOR]", row)
	}
	return row
}

// ClimbScoreString generates the climb score for an offset and outputs it.
func (g Game) ClimbScoreString(offset int) string {
	if offset == len(g.Records)-1 {
		// Output the COLOR tag anyway for alignment purposes.
		return "[COLOR=#000000][/COLOR]"
	}
	return ClimbScoreString(ClimbScore(g.Records[offset+1].Record.BayesAverage, g.Records[offset].Record.BayesAverage))
}

// ToCSVRecord outputs a CSV row for output.
func (g Game) ToCSVRecord() []string {
	return []string{
		g.Records[0].Record.ID,
		g.Records[0].Record.Name,
		g.Description(),
		fmt.Sprintf("%.3f", g.ClimbScore()),
	}
}

// A GameRecord is a record with the date included.
type GameRecord struct {
	Record
	Date time.Time
}

// Games is a sortable collection of Game structs.
type Games []Game

func (g Games) Len() int           { return len(g) }
func (g Games) Swap(i, j int)      { g[i], g[j] = g[j], g[i] }
func (g Games) Less(i, j int) bool { return g[i].ClimbScore() < g[j].ClimbScore() }

// ParseRecord parses a record from a CSV row.
func ParseRecord(record []string) (Record, error) {
	if len(record) <= SrcThumbnail {
		return Record{}, fmt.Errorf("record too short: %#v", record)
	}
	rank, err := strconv.Atoi(record[SrcRank])
	if err != nil {
		return Record{}, fmt.Errorf("unable to parse rank '%s', %s", record[SrcRank], err)
	}
	bayes, err := strconv.ParseFloat(record[SrcBayesAverage], 64)
	if err != nil {
		return Record{}, fmt.Errorf("unable to parse bayes average '%s', %s", record[SrcBayesAverage], err)
	}
	return Record{
		ID:           record[SrcID],
		Name:         record[SrcName],
		Year:         record[SrcYear],
		Rank:         rank,
		Average:      record[SrcAverage],
		BayesAverage: bayes,
		UsersRated:   record[SrcUsersRated],
		URL:          record[SrcURL],
		Thumbnail:    record[SrcThumbnail],
	}, nil
}

func main() {
	// Parse flags and arg
	var (
		minRatings int
		period     int
		maxPeriods int
	)
	flag.IntVar(&minRatings, "minratings", 100, "minimum ratings required")
	flag.IntVar(&period, "period", 7, "number of days in a period")
	flag.IntVar(&maxPeriods, "maxperiods", 12, "maximum periods to include")
	flag.Parse()
	args := flag.Args()

	stderr := log.New(os.Stderr, "", 0)
	if len(args) != 1 {
		stderr.Fatalf("Expected bgg-ranking-historicals CSV file")
	}
	latest := args[0]

	// Read files
	files := []File{}
	dir := filepath.Dir(latest)
	timeIter, err := ParseFileDate(latest)
	if err != nil {
		stderr.Fatalf("Error parsing date from %s, %v", latest, err)
	}
	for {
		if len(files) >= maxPeriods {
			break
		}

		csvPath := path.Join(dir, timeIter.Format(FileDateFormat)+".csv")
		log.Printf("Parsing %s", csvPath)
		if _, err := os.Stat(csvPath); os.IsNotExist(err) {
			log.Printf("Could not find file, cancelling further iteration")
			break
		}

		f, err := ParseFile(csvPath)
		if err != nil {
			stderr.Fatalf("Error reading file %s, %s", latest, err)
		}

		files = append(files, f)
		timeIter = timeIter.AddDate(0, 0, -period)
	}

	if len(files) < 2 {
		stderr.Fatal("Parsed less than two files")
	}

	// Build and sort games array, only including games in the latest file.
	gamesMap := map[string]Game{}
	for _, record := range files[0].Records {
		if record.Rank > 0 {
			gamesMap[record.ID] = Game{
				Records: []GameRecord{{
					Record: record,
					Date:   files[0].Date,
				}},
			}
		}
	}
	for _, f := range files[1:] {
		for _, record := range f.Records {
			if game, ok := gamesMap[record.ID]; ok && record.Rank > 0 {
				game.Records = append(game.Records, GameRecord{
					Record: record,
					Date:   f.Date,
				})
				gamesMap[record.ID] = game
			}
		}
	}
	games := Games{}
	for _, g := range gamesMap {
		if len(g.Records) > 1 {
			usersRated, _ := strconv.Atoi(g.Records[1].UsersRated)
			if usersRated >= minRatings {
				games = append(games, g)
			}
		}
	}
	sort.Sort(sort.Reverse(games))

	// Write header
	w := csv.NewWriter(os.Stdout)
	defer w.Flush()
	if err := w.Write([]string{
		"ID",
		"Name",
		"Description",
		"Climb ratio",
	}); err != nil {
		stderr.Fatalf("Error writing header, %v", err)
	}

	for _, g := range games {
		if err := w.Write(g.ToCSVRecord()); err != nil {
			stderr.Fatalf("Unable to write CSV row, %s", err)
		}
	}
}
