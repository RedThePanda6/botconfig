package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"reflect"
	"strconv"
	"strings"
	"time"
)

var (
	debug      = flag.Bool("debug", false, "Print debugging info.")
	game       = flag.String("game", "", "The game we are looking up.")
	configRoot = flag.String(
		"configRoot",
		"G:\\My Drive\\Streaming\\Chatbot\\twitch_configs\\",
		"Root folder where configs are found.",
	)
	writeJSONFile = flag.Bool("writeJSONFile", true, "Write a JSON file?")
	outFile       = flag.String(
		"outFile",
		"D:\\Temp\\config.json",
		"The output file we write merged configs to.",
	)
	dayOverride  = flag.String("dayOverride", "", "Manually set day for testing.")
	dateOverride = flag.String("dateOverride", "", "Manually set date for testing.")
	// Default tags to apply if no game configuration is found.
	defaultTags = []string{}
	// A list of all include files read by filename to avoid processing duplicates.
	// Mostly as a cheap backstop to prevent a recursive loop of includes.
	includesSeen = map[string]bool{}
)

type config struct {
	// Includes
	Include string `json:"include"`
	// Stream Settings
	StreamTags  []string `json:"streamtags"`
	TitleSuffix string   `json:"titlesuffix"`
	// Control
	GameFound bool   `json:"gamefound"`
	GameName  string `json:"gamename"`
}

func newConfig() config {
	// For setting non-standard default values.
	return config{
		// Especially any booleans that default to true.
		// Or strings that should have a dafault value.
	}
}

func getBool(c config, field string) bool {
	r := reflect.ValueOf(c)
	return r.FieldByName(field).Bool()
}

func resolveBool(a config, b config, field string) bool {
	if getBool(newConfig(), field) {
		return getBool(a, field) && getBool(b, field)
	}
	return getBool(a, field) || getBool(b, field)
}

func readFromFile(f string) config {
	c := newConfig()
	c.GameFound = true
	configFile, err := os.Open(f)
	defer configFile.Close()
	if err != nil {
		slog.Debug("Error loading config:", err.Error(), err)
		c.GameFound = false
	}
	jsonParser := json.NewDecoder(configFile)
	jsonParser.Decode(&c)
	return c
}

func writeToFile(f string, c config) {
	// Write the merged data to a new JSON file.
	outputFile, err := os.Create(f)
	if err != nil {
		slog.Debug("Error creating config file:", err.Error(), err)
	}
	defer outputFile.Close()

	output, _ := json.MarshalIndent(c, "", "  ")

	_, err = outputFile.Write(output)
	if err != nil {
		slog.Debug("Error writing config file:", err.Error(), err)
	}

	outputFile.Sync()

	w := bufio.NewWriter(outputFile)
	w.Flush()
}

func removeDuplicateStr(strSlice []string) []string {
	allKeys := make(map[string]bool)
	list := []string{}
	for _, item := range strSlice {
		// Remove spaces otherwise Twitch will reject.
		item := strings.ReplaceAll(item, " ", "")

		if !allKeys[item] {
			list = append(list, item)
			allKeys[item] = true
		}
	}
	return list
}

func mergeConfigs(o config, n config) config {
	// Merge StreamTags
	o.StreamTags = removeDuplicateStr(
		append(o.StreamTags, n.StreamTags...),
	)

	// Keep include processing first!
	// Reason being to have original take precedent over the include.
	// (Last config applied wins.)
	if n.Include != "" {
		includeFile := fmt.Sprintf("%sincludes\\%s.json", *configRoot, n.Include)
		// Skip if we've read this file before.
		if !includesSeen[includeFile] {
			includesSeen[includeFile] = true
			i := readFromFile(includeFile)

			if i.GameFound {
				slog.Debug("    Inlcuded " + n.Include + " configs...")
				o = mergeConfigs(o, i)
			}
		} else {
			slog.Debug("    Already seen " + n.Include + " in another config...")
		}
	}

	if n.TitleSuffix != "" {
		o.TitleSuffix = n.TitleSuffix
	}

	// Pull all bools from config struct to resolve them.
	boolsToResolve := []string{}

	r := reflect.ValueOf(o)

	for i := range r.NumField() {
		if r.Field(i).Kind() == reflect.Bool {
			boolsToResolve = append(boolsToResolve, r.Type().Field(i).Name)
		}
	}

	for _, f := range boolsToResolve {
		reflect.ValueOf(&o).Elem().FieldByName(f).SetBool(
			resolveBool(o, n, f),
		)
	}

	return o
}

func applyOverrides(c config) config {
	// Values that don't need to be passed into StreamerBot.
	c.Include = ""

	// Twitch supports max 10 tags.
	tagCount := len(c.StreamTags)
	slog.Debug("Found " + strconv.Itoa(tagCount) + " tags...")
	if tagCount > 10 {
		slog.Debug("More than 10 tags found. Please clean some of them up!")
		c.StreamTags = c.StreamTags[:10]
	}

	// Set GameName to passed in value.
	c.GameName = *game

	return c
}

func sanitizeGame(s string) string {
	for _, c := range []string{
		":", "&", "#", "\\", "/", "?", "@", "+", "|", "=", ",",
	} {
		s = strings.Replace(s, c, "_", -1)
	}
	return s
}

func main() {
	flag.Parse()

	if *debug {
		handler := slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelDebug})
		slog.SetDefault(slog.New(handler))
	}

	if *game == "" {
		slog.Error("--game flag required.")
		os.Exit(1)
	}

	slog.Debug("Processing game " + *game + ".")

	weekday := time.Now().Weekday().String()
	// Override day of week for testing.
	if len(*dayOverride) > 0 {
		weekday = *dayOverride
	}
	slog.Debug("Today is " + weekday + "...")

	// Grab today's date in <Month>-<Day> format.
	date := fmt.Sprintf(time.Now().Month().String() + "-" + strconv.Itoa(time.Now().Day()))
	if len(*dateOverride) > 0 {
		date = *dateOverride
	}
	slog.Debug("Date is " + date + "...")

	saneGame := sanitizeGame(*game)

	// Set the names of the JSON files to merge.
	globalFile := fmt.Sprintf("%sglobal.json", *configRoot)
	gameFile := fmt.Sprintf("%sgames\\%s.json", *configRoot, saneGame)
	dayFile := fmt.Sprintf("%sday\\%s.json", *configRoot, weekday)
	dateFile := fmt.Sprintf("%sdate\\%s.json", *configRoot, date)

	// Read the JSON files into data structures.
	slog.Debug("Reading configs...")
	globalConfig := readFromFile(globalFile)
	gameConfig := readFromFile(gameFile)
	dayConfig := readFromFile(dayFile)
	dateConfig := readFromFile(dateFile)

	// Combine the JSON files with preference for gameConfig.
	// Included/Nested configs will be recursed during each merge.
	slog.Debug("Merging configs...")
	twitchConfigs := newConfig()

	// global
	if globalConfig.GameFound {
		slog.Debug("  Global configs...")
		twitchConfigs = mergeConfigs(twitchConfigs, globalConfig)
	}

	// game
	if gameConfig.GameFound {
		slog.Debug("  Game configs...")
		twitchConfigs = mergeConfigs(twitchConfigs, gameConfig)
	} else {
		// If we don't find the game file then add the defaultTags.
		twitchConfigs.StreamTags = removeDuplicateStr(
			append(twitchConfigs.StreamTags, defaultTags...),
		)
	}

	// day
	if dayConfig.GameFound {
		slog.Debug("  Day configs...")
		twitchConfigs = mergeConfigs(twitchConfigs, dayConfig)
	}

	// date
	if dateConfig.GameFound {
		slog.Debug("  Date configs...")
		twitchConfigs = mergeConfigs(twitchConfigs, dateConfig)
	}

	// Apply overrides.
	twitchConfigs = applyOverrides(twitchConfigs)

	// Things we need to set after all is said and done.
	// Typically things we can't do in the applyOverrides scope.
	twitchConfigs.GameFound = gameConfig.GameFound

	// Write to output file.
	if *writeJSONFile {
		slog.Debug("Writing JSON file...")
		writeToFile(*outFile, twitchConfigs)
	}

	// Write out JSON.
	if err := json.NewEncoder(os.Stdout).Encode(twitchConfigs); err != nil {
		panic(err)
	}

	slog.Debug("End of Line.")
}
