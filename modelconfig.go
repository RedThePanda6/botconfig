package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"reflect"
	"slices"
	"strings"
)

var (
	debug      = flag.Bool("debug", false, "Print debugging info.")
	modelFile  = flag.String("modelFile", "", "Model file of config we're loading.")
	configRoot = flag.String(
		"configRoot",
		"G:\\My Drive\\Streaming\\Chatbot\\model_configs\\",
		"Root folder where configs are found.",
	)
	writeJSONFile = flag.Bool("writeJSONFile", true, "Write a JSON file?")
	outFile       = flag.String(
		"outFile",
		"D:\\Temp\\model_config.json",
		"The output file we write merged configs to.",
	)
	writeSchema = flag.Bool("writeSchema", false, "Write a schema file?")
	schemaFile  = flag.String(
		"schemaFile",
		"G:\\My Drive\\Streaming\\Chatbot\\model_configs\\schema.json",
		"The schema file used to validate configs.",
	)
	// A list of all include files read by filename to avoid processing duplicates.
	// Mostly as a cheap backstop to prevent a recursive loop of includes.
	includesSeen = map[string]bool{}
)

type config struct {
	// Includes
	Include string `json:"include"`
	// Control
	ConfigFound   bool   `json:"configfound"`
	ModelFileName string `json:"modelfilename"`
	Software      string `json:"software"`
	// Redeems
	AnvilDrop     bool `json:"anvildrop"`
	ASCIIRed      bool `json:"asciired"`
	Bonk          bool `json:"bonk"`
	Boop          bool `json:"boop"`
	Chaos         bool `json:"chaos"`
	Feets         bool `json:"feets"`
	Fisheye       bool `json:"fisheye"`
	Headpats      bool `json:"headpats"`
	NoGlasses     bool `json:"noglasses"`
	NuggiesForRed bool `json:"nuggiesforred"`
	PeltThePanda  bool `json:"peltthepanda"`
	PieDrop       bool `json:"piedrop"`
	PostItRed     bool `json:"postitred"`
	RedInABox     bool `json:"redinabox"`
	RentThisHat   bool `json:"rentthishat"`
	SpinThePanda  bool `json:"spinthepanda"`
	SprayBottle   bool `json:"spraybottle"`
	SuspiciousRed bool `json:"suspiciousred"`
	SwolePanda    bool `json:"swolepanda"`
	Tail          bool `json:"tail"`
	TimeWarpScan  bool `json:"timewarpscan"`
	ToughLove     bool `json:"toughlove"`
}

func newConfig() config {
	// By default we start with all bools as true then override to false.
	config := config{}

	allBools := []string{}

	// Bools that need to be false for reasons.
	controlBools := []string{
		"ConfigFound",
	}

	r := reflect.ValueOf(config)

	// There's probably a better way to do this but I'm lazy right now.
	for i := range r.NumField() {
		if r.Field(i).Kind() == reflect.Bool {
			if !(slices.Contains(controlBools, r.Type().Field(i).Name)) {
				allBools = append(allBools, r.Type().Field(i).Name)
			}
		}
	}

	for _, f := range allBools {
		reflect.ValueOf(&config).Elem().FieldByName(f).SetBool(
			true,
		)
	}

	return config
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
	c.ConfigFound = true
	configFile, err := os.Open(f)
	if err != nil {
		slog.Debug("Error loading config:", err.Error(), err)
		c.ConfigFound = false
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

	output, _ := json.MarshalIndent(c, "", "  ")

	_, err = outputFile.Write(output)
	if err != nil {
		slog.Debug("Error writing config file:", err.Error(), err)
	}

	outputFile.Sync()

	w := bufio.NewWriter(outputFile)
	w.Flush()
}

func mergeConfigs(o config, n config) config {

	// Keep include processing first!
	// Reason being to have original take precedent over the include.
	// (Last config applied wins.)
	if n.Include != "" {
		includeFile := fmt.Sprintf("%sincludes\\%s.json", *configRoot, n.Include)
		// Skip if we've read this file before.
		if !includesSeen[includeFile] {
			includesSeen[includeFile] = true
			i := readFromFile(includeFile)

			if i.ConfigFound {
				slog.Debug("    Inlcuded " + n.Include + " configs...")
				o = mergeConfigs(o, i)
			}
		} else {
			slog.Debug("    Already seen " + n.Include + " in another config...")
		}
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

	// Export which software we've selected.
	// I assume this could be useful. At least for troubleshooting.
	if n.Software != "" {
		o.Software = n.Software
	}

	return o
}

func applyOverrides(c config) config {
	// Values that don't need to be passed into StreamerBot.
	c.Include = ""

	// No config found means no model is found.
	// Disable all redeems.
	if !c.ConfigFound {
		allBools := []string{}

		r := reflect.ValueOf(c)

		// There's probably a better way to do this but I'm lazy right now.
		for i := range r.NumField() {
			if r.Field(i).Kind() == reflect.Bool {
				allBools = append(allBools, r.Type().Field(i).Name)
			}
		}

		for _, f := range allBools {
			reflect.ValueOf(&c).Elem().FieldByName(f).SetBool(
				false,
			)
		}
	}

	return c
}

func sanitizeModelFileName(f string) string {
	// We're expecting a filename.ext format.
	// We want to return just the filename without ext.
	s := strings.Split(f, ".")
	return s[0]
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

	r := reflect.ValueOf(config)

	for i := range r.NumField() {
		n := strings.ToLower(r.Type().Field(i).Name)

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

	output, _ := json.MarshalIndent(schema, "", "  ")

	_, err = outputFile.Write(output)
	if err != nil {
		slog.Debug("Error writing config file:", err.Error(), err)
	}

	outputFile.Sync()

	w := bufio.NewWriter(outputFile)
	w.Flush()
}

func main() {
	flag.Parse()

	if *debug {
		handler := slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelDebug})
		slog.SetDefault(slog.New(handler))
	}

	if *modelFile == "" {
		slog.Error("--modelFile flag required.")
		os.Exit(1)
	}

	saneModelFile := sanitizeModelFileName(*modelFile)

	// Set the names of the JSON files to merge.
	modelFileName := fmt.Sprintf("%s%s.json", *configRoot, saneModelFile)

	// Read the JSON files into data structures.
	slog.Debug("Reading configs...")
	modelConfig := readFromFile(modelFileName)

	// Combine the JSON files with preference for gameConfig.
	// Included/Nested configs will be recursed during each merge.
	slog.Debug("Merging configs...")
	config := newConfig()

	// global
	if modelConfig.ConfigFound {
		slog.Debug("  Model configs...")
		config = mergeConfigs(config, modelConfig)
	}

	// Set ConfigFound to model's setting before we apply overrides.
	config.ConfigFound = modelConfig.ConfigFound

	// Apply overrides.
	config = applyOverrides(config)

	// Things we need to set after all is said and done.
	// Typically things we can't do in the applyOverrides scope.
	config.ModelFileName = saneModelFile

	// Write to output file.
	if *writeJSONFile {
		slog.Debug("Writing JSON file...")
		writeToFile(*outFile, config)
	}

	// Write out JSON.
	if err := json.NewEncoder(os.Stdout).Encode(config); err != nil {
		panic(err)
	}

	// Write out JSON schema.
	if *writeSchema {
		slog.Debug("Writing schema file...")
		writeSchemaFile()
	}

	slog.Debug("End of Line.")
}
