package main

import (
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
		"D:\\Temp\\twitch_config.json",
		"The output file we write merged configs to.",
	)
	writeSchema = flag.Bool("writeSchema", false, "Write a schema file?")
	schemaFile  = flag.String(
		"schemaFile",
		"G:\\My Drive\\Streaming\\Chatbot\\twitch_configs\\schema.json",
		"The schema file used to validate configs.",
	)
	onCall       = flag.Bool("oncall", false, "Am I oncall for work?")
	dayOverride  = flag.String("dayOverride", "", "Manually set day for testing.")
	dateOverride = flag.String("dateOverride", "", "Manually set date for testing.")
	defaultTags  = []string{
		"FirstPlaythrough",
		"NoBackseating",
	}
	// Default costs of LP.
	defaultLPTalkingCost = int(250)
	defaultLPGameCost    = defaultLPTalkingCost * 2
	// A list of all include files read by filename to avoid processing duplicates.
	// Mostly as a cheap backstop to prevent a recursive loop of includes.
	includesSeen = map[string]bool{}
	// We now set this within StreamerBot based on which programs are running.
	vtuberSoftware = flag.String(
		"vtuberSoftware",
		defaultVTuberSoftware,
		"Which VTuber Software we set tags for.",
	)
	validVTuberSoftware = map[string]bool{
		"None":      true,
		"Veadotube": true,
		"VNyan":     true,
		"VTS":       true,
	}
	defaultVTuberSoftware = "VNyan"
)

type config struct {
	// Includes
	Include string `json:"include"`
	// Stream Settings
	StreamTags     []string `json:"streamtags"`
	TitleSuffix    string   `json:"titlesuffix"`
	VTuberSoftware string   `json:"vtubersoftware"` // Move to flag so we can set stream tags.
	// Model Options
	VNyanOutfit string `json:"vnyanoutfit"`
	// Overlays
	DeathCounter  bool   `json:"deathcounter"`
	DeskCam       bool   `json:"deskcam"`
	GamePad       bool   `json:"gamepad"`
	OrpaxMemorial bool   `json:"orpaxmemorial"`
	PandaSign     string `json:"pandasign"`
	Uptime        bool   `json:"uptime"`
	// Other Functions
	OutfitPoll   bool `json:"outfitpoll"`
	SongRequests bool `json:"songrequests"`
	// Rewards
	BambooRequestCost int  `json:"bamboorequestcost"`
	BedTime           bool `json:"bedtime"`
	ChosenOne         bool `json:"chosenone"`
	CreepyTime        bool `json:"creepytime"`
	JibberJabbey      bool `json:"jibberjabbey"`
	LPGameCost        int  `json:"lpgamecost"`
	LPTalkingCost     int  `json:"lptalkingcost"`
	NameAThing        bool `json:"nameathing"`
	NoBeanie          bool `json:"nobeanie"`
	RaidRoulette      bool `json:"raidroulette"`
	// Commands
	// Bot Functions
	Modlist        bool `json:"modlist"`
	NotifyInterval int  `json:"notifyinterval"`
	// Control
	// Note that GameFound serves the dual purpose to communicate to StreamerBot
	// if we have a config for the game as well as to signal if we've found a
	// config file here so we don't need to merge "empty" configs.
	EndHour           int    `json:"endhour"`
	EndMinute         int    `json:"endminute"`
	GameFound         bool   `json:"gamefound"`
	GameName          string `json:"gamename"`
	SanitizedGameName string `json:"sanitizedgamename"`
	OnCall            bool   `json:"oncall"`
	PauseableGame     bool   `json:"pauseablegame"`
	YTGameInTitle     bool   `json:"ytgameintitle"`
}

func newConfig() config {
	// For setting non-standard default values.
	return config{
		BambooRequestCost: 20,
		ChosenOne:         true,
		JibberJabbey:      true,
		LPGameCost:        defaultLPGameCost,
		LPTalkingCost:     defaultLPTalkingCost,
		NotifyInterval:    5,
		OutfitPoll:        true,
		PandaSign:         "default",
		PauseableGame:     true,
		RaidRoulette:      true,
		YTGameInTitle:     true,
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
	if err != nil {
		slog.Debug("Error loading config:", err.Error(), err)
		c.GameFound = false
	}
	defer configFile.Close()
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

	encoder := json.NewEncoder(outputFile)
	encoder.SetEscapeHTML(false)
	encoder.SetIndent("", "  ")

	err = encoder.Encode(c)
	if err != nil {
		slog.Debug("Error writing config file:", err.Error(), err)
	}
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

	if n.VTuberSoftware != "" {
		if validVTuberSoftware[n.VTuberSoftware] {
			o.VTuberSoftware = n.VTuberSoftware
		} else {
			slog.Debug("Invalid VTuberSoftware found: " + n.VTuberSoftware)
		}
	}

	if n.TitleSuffix != "" {
		o.TitleSuffix = n.TitleSuffix
	}

	if n.NotifyInterval < o.NotifyInterval {
		o.NotifyInterval = n.NotifyInterval
	}

	if n.PandaSign != "default" {
		o.PandaSign = n.PandaSign
	}

	if n.EndHour != 0 {
		o.EndHour = n.EndHour
	}

	if n.EndMinute != 0 {
		o.EndMinute = n.EndMinute
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

	// Outfit conflict resolution.
	if n.VNyanOutfit != "" {
		o.VNyanOutfit = n.VNyanOutfit
	}

	// LP Cost overrides.
	// In Game Cost
	if n.LPGameCost != defaultLPGameCost {
		o.LPGameCost = n.LPGameCost
	}

	// Talking Scene Cost
	if n.LPTalkingCost != defaultLPTalkingCost {
		o.LPTalkingCost = n.LPTalkingCost
	}

	// Bamboo Request Cost
	// Go with whichever is higher.
	if n.BambooRequestCost > o.BambooRequestCost {
		o.BambooRequestCost = n.BambooRequestCost
	}

	return o
}

func applyOverrides(c config) config {
	// Values that don't need to be passed into StreamerBot.
	c.Include = ""

	// Sanity check the VTuber Software set.
	if !validVTuberSoftware[c.VTuberSoftware] {
		slog.Debug("Invalid VTuberSoftware set. Using default: " + defaultVTuberSoftware + ".")
		c.VTuberSoftware = defaultVTuberSoftware
	}

	// Apply overrides based on VTuberSoftware.
	switch c.VTuberSoftware {
	// PNGTuber Settings
	case "Veadotube":
		c.StreamTags = removeDuplicateStr(
			append([]string{"VTuber", "RedPanda", "ENVTuber"}, c.StreamTags...),
		)

		// Disable VNyan Stuff.
		c.OutfitPoll = false
		c.VNyanOutfit = ""

		// Disable incompatible redeems.
		c.NoBeanie = false

	// VTube Studio Settings
	case "VTS":
		c.StreamTags = removeDuplicateStr(
			append([]string{"VTuber", "RedPanda", "Furry", "ENVTuber"}, c.StreamTags...),
		)

		// Disable VNyan Stuff
		c.OutfitPoll = false
		c.VNyanOutfit = ""

		// Disable incompatible redeems.
		c.NoBeanie = false

	// VNyan Settings
	case "VNyan":
		c.StreamTags = removeDuplicateStr(
			append([]string{"VTuber", "RedPanda", "Furry", "ENVTuber"}, c.StreamTags...),
		)

		// Outfit overrides.
		if c.VNyanOutfit != "" {
			c.OutfitPoll = false
		}

		// Disable incompatible redeems.
		c.NoBeanie = false

	// Facecam Settings
	case "None":
		// Disable incompatible redeems.
	}

	// Twitch supports max 10 tags.
	tagCount := len(c.StreamTags)
	slog.Debug("Found " + strconv.Itoa(tagCount) + " tags...")
	if tagCount > 10 {
		slog.Debug("More than 10 tags found. Please clean some of them up!")
		c.StreamTags = c.StreamTags[:10]
	}

	// Set GameName to passed in value.
	c.GameName = *game

	// Oncall overrides.
	if *onCall || c.OnCall {
		c.OnCall = true
		c.BedTime = false
		c.CreepyTime = false
		c.RaidRoulette = false
	}

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

func writeSchemaFile() {
	f := *schemaFile
	config := newConfig()

	// Handle properties separately.
	properties := make(map[string]any)
	// Special cases or properties outside of struct.
	properties["_comment"] = map[string]any{
		"type": "string",
	}
	properties["$schema"] = map[string]any{
		"type": "string",
	}
	properties["streamtags"] = map[string]any{
		"type": "array",
		"items": []map[string]any{
			{"type": "string"},
		},
	}

	r := reflect.ValueOf(config)

	for i := range r.NumField() {
		n := strings.ToLower(r.Type().Field(i).Name)
		// streamtags are handled specially.
		if n == "streamtags" {
			continue
		}

		t := r.Type().Field(i).Type.String()

		// Convert type string to valid JSON schema values.
		switch t {
		case "int":
			t = "integer"
		case "bool":
			t = "boolean"
		}

		properties[n] = map[string]any{
			"type": t,
		}
	}

	schema := make(map[string]any)
	schema["type"] = "object"
	schema["additionalProperties"] = false
	schema["properties"] = properties

	outputFile, err := os.Create(f)
	if err != nil {
		slog.Debug("Error creating schema file:", err.Error(), err)
	}
	defer outputFile.Close()

	encoder := json.NewEncoder(outputFile)
	encoder.SetEscapeHTML(false)
	encoder.SetIndent("", "  ")

	err = encoder.Encode(schema)
	if err != nil {
		slog.Debug("Error writing config file:", err.Error(), err)
	}
}

func main() {
	flag.Parse()

	if *debug {
		handler := slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelDebug})
		slog.SetDefault(slog.New(handler))
	}

	if !validVTuberSoftware[defaultVTuberSoftware] {
		slog.Error("defaultVTuberSoftware is not a valid value. Fix it and recompile!")
		os.Exit(1)
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
	yeardate := fmt.Sprintf(date + "-" + strconv.Itoa(time.Now().Year()))
	slog.Debug("Date w/Year is " + yeardate + "...")

	saneGame := sanitizeGame(*game)

	// Set the names of the JSON files to merge.
	globalFile := fmt.Sprintf("%sglobal.json", *configRoot)
	gameFile := fmt.Sprintf("%sgames\\%s.json", *configRoot, saneGame)
	dayFile := fmt.Sprintf("%sday\\%s.json", *configRoot, weekday)
	dateFile := fmt.Sprintf("%sdate\\%s.json", *configRoot, date)
	yeardateFile := fmt.Sprintf("%sdate\\%s.json", *configRoot, yeardate)

	// Read the JSON files into data structures.
	slog.Debug("Reading configs...")
	globalConfig := readFromFile(globalFile)
	gameConfig := readFromFile(gameFile)
	dayConfig := readFromFile(dayFile)
	dateConfig := readFromFile(dateFile)
	yeardateConfig := readFromFile(yeardateFile)

	// Combine the JSON files with preference for gameConfig.
	// Included/Nested configs will be recursed during each merge.
	slog.Debug("Merging configs...")
	twitchConfigs := newConfig()
	// Set VTuberSoftware based on flag.
	twitchConfigs.VTuberSoftware = *vtuberSoftware

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

	// date w/ year
	if yeardateConfig.GameFound {
		slog.Debug("  Date w/Year configs...")
		twitchConfigs = mergeConfigs(twitchConfigs, yeardateConfig)
	}

	// Apply overrides.
	twitchConfigs = applyOverrides(twitchConfigs)

	// Things we need to set after all is said and done.
	// Typically things we can't do in the applyOverrides scope.
	twitchConfigs.SanitizedGameName = saneGame
	twitchConfigs.GameFound = gameConfig.GameFound

	// Write to output file.
	if *writeJSONFile {
		slog.Debug("Writing JSON file...")
		writeToFile(*outFile, twitchConfigs)
	}

	// Write out JSON.
	// Disable SetEscapeHTML to hopefully avoid mangling characters in titlesuffix.
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetEscapeHTML(false)
	if err := encoder.Encode(twitchConfigs); err != nil {
		panic(err)
	}

	// Write out JSON schema.
	if *writeSchema {
		slog.Debug("Writing schema file...")
		writeSchemaFile()
	}

	slog.Debug("End of Line.")
}
