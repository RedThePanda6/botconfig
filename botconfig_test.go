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
	want := time.Now().Weekday().String()

	got := weekday()

	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("weekday() mismatch (-want +got):\n%s", diff)
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
	// Silly hack because this was getting set to true while running mergeConfig tests.
	bambooSpecialPresent = false

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

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.config.applyOverrides(tt.softare)
			if diff := cmp.Diff(tt.want, tt.config, cmpopts.EquateEmpty()); diff != "" {
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
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.config.mergeConfigs(tt.new)
			if diff := cmp.Diff(tt.want, tt.config, cmpopts.EquateEmpty()); diff != "" {
				t.Errorf("mergConfigs() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
