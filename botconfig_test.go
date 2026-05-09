package main

import (
	"fmt"
	"strconv"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
)

func TestDayOfWeek(t *testing.T) {
	// Yes I acknowledge that this really should be using some form on controlled
	// or mocked time but this is good enough beacuse I doubt I'd ever be testing
	// this exactly as a day rolls over.
	want := time.Now().Weekday().String()

	got := weekday()

	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("weekday() mismatch (-want +got):\n%s", diff)
	}
}

func TestMonthYear(t *testing.T) {
	// Yes I acknowledge that this really should be using some form on controlled
	// or mocked time but this is good enough beacuse I doubt I'd ever be testing
	// this exactly as a day rolls over.
	now := time.Now()
	want := fmt.Sprintf(
		"%s-%s",
		now.Month().String(),
		strconv.Itoa(now.Year()),
	)

	got := monthYear()

	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("monthYear() mismatch (-want +got):\n%s", diff)
	}
}

func TestFullDate(t *testing.T) {
	// Yes I acknowledge that this really should be using some form on controlled
	// or mocked time but this is good enough beacuse I doubt I'd ever be testing
	// this exactly as a day rolls over.
	now := time.Now()
	want := fmt.Sprintf(
		"%s-%s-%s",
		now.Month().String(),
		strconv.Itoa(now.Day()),
		strconv.Itoa(now.Year()),
	)

	got := dateYear()

	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("dateYear() mismatch (-want +got):\n%s", diff)
	}
}

func TestStringDeduplication(t *testing.T) {
	want := []string{"stringA", "stringB", "stringC"}

	got := removeDuplicateStr([]string{"stringA", "stringA", "stringB", "stringC"})

	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("removeDuplicateStr() mismatch (-want +got):\n%s", diff)
	}
}

func TestApplyOverrides(t *testing.T) {
	vnyan := *newConfig()
	vnyan.StreamTags = []string{"VTuber", "RedPanda", "Furry", "ENVTuber"}

	vts := *newConfig()
	vts.StreamTags = []string{"VTuber", "RedPanda", "Furry", "ENVTuber"}
	vts.OutfitPoll = false

	veadotube := *newConfig()
	veadotube.StreamTags = []string{"VTuber", "RedPanda", "ENVTuber"}
	veadotube.OutfitPoll = false

	facecam := *newConfig()

	var tests = []struct {
		name    string
		softare string
		config  config
		want    config
	}{
		{
			"VNyan",
			"VNyan",
			*newConfig(),
			vnyan,
		},
		{
			"VTS",
			"VTS",
			*newConfig(),
			vts,
		},
		{
			"Veadotube",
			"Veadotube",
			*newConfig(),
			veadotube,
		},
		{
			"None",
			"None",
			*newConfig(),
			facecam,
		},
	}

	for _, tc := range tests {
		// Silly hack because this was getting set to true while running mergeConfig tests.
		bambooSpecialPresent = false
		t.Run(tc.name, func(t *testing.T) {
			tc.config.applyOverrides(tc.softare, "")
			if diff := cmp.Diff(tc.want, tc.config, cmpopts.EquateEmpty()); diff != "" {
				t.Errorf("applyOverrides() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestMergeConfigs(t *testing.T) {
	easyBoolMerge := *newConfig()
	easyBoolMerge.PauseableGame = false
	easyBoolMerge.NameAThing = true

	suffixA := *newConfig()
	suffixA.TitleSuffix = "Suffix A"
	suffixB := *newConfig()
	suffixB.TitleSuffix = "Suffix B"
	mergedSuffix := *newConfig()
	mergedSuffix.TitleSuffix = "Suffix A | Suffix B"

	endHourMinute := *newConfig()
	endHourMinute.EndHour = 12
	endHourMinute.EndMinute = 42

	bambooSpecial := *newConfig()
	bambooSpecial.BambooSpecial = true

	lpCostOverride := *newConfig()
	lpCostOverride.LPGameCost = 5000
	lpCostOverride.LPTalkingCost = 5000

	var tests = []struct {
		name   string
		config config
		new    config
		want   config
	}{
		{
			"All default configs",
			*newConfig(),
			*newConfig(),
			*newConfig(),
		},
		{
			"Easy Bool Merges",
			*newConfig(),
			easyBoolMerge,
			easyBoolMerge,
		},
		{
			"Easy Bool Merges Inverse",
			easyBoolMerge,
			*newConfig(),
			easyBoolMerge,
		},
		{
			"Merge Multiple Suffixes",
			suffixA,
			suffixB,
			mergedSuffix,
		},
		{
			"Custom End Hour + Minute",
			*newConfig(),
			endHourMinute,
			endHourMinute,
		},
		{
			"Custom End Hour + Minute Inverse",
			endHourMinute,
			*newConfig(),
			endHourMinute,
		},
		{
			"Don't resolve BambooSpecial",
			*newConfig(),
			bambooSpecial,
			*newConfig(),
		},
		{
			"LP Cost",
			*newConfig(),
			lpCostOverride,
			lpCostOverride,
		},
		{
			"LP Cost Inverse",
			lpCostOverride,
			*newConfig(),
			lpCostOverride,
		},
	}

	for _, tc := range tests {
		// Silly hack because this was getting set to true while running mergeConfig tests.
		bambooSpecialPresent = false
		t.Run(tc.name, func(t *testing.T) {
			tc.config.mergeConfigs(tc.new)
			if diff := cmp.Diff(tc.want, tc.config, cmpopts.EquateEmpty()); diff != "" {
				t.Errorf("mergConfigs() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestTestGame(t *testing.T) {
	// Attempting (poorly) to simulate an end-to-end run and compare against a
	// known good file. The file will need updated from time to time.
	gameName := "Test"
	c := newConfig()
	c.SanitizedGameName = sanitizeGame(gameName)
	global := readFromFile("G:\\My Drive\\Streaming\\Chatbot\\twitch_configs\\global.json")
	g := readFromFile("G:\\My Drive\\Streaming\\Chatbot\\twitch_configs\\games\\Test.json")

	want := readFromFile("C:\\Users\\mbern\\go\\src\\botconfig\\TestOutput.json")

	c.mergeConfigs(*global)
	c.mergeConfigs(*g)
	c.applyOverrides("VNyan", gameName)

	if diff := cmp.Diff(want, c); diff != "" {
		t.Errorf("config mismatch (-want +got):\n%s", diff)
	}
}
