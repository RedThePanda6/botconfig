package main

import (
	"reflect"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestSanitizeFileName(t *testing.T) {
	want := "model filename"

	got := sanitizeModelFileName("model filename.vsfavatar")

	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("sanitizeFileName() mismatch (-want +got):\n%s", diff)
	}
}

func TestApplyOverrides(t *testing.T) {
	notFound := *newConfig()
	r := reflect.ValueOf(&notFound).Elem()
	for i := 0; i < r.NumField(); i++ {
		field := r.Field(i)
		if field.Kind() == reflect.Bool && field.CanSet() {
			field.SetBool(false)
		}
	}

	var tests = []struct {
		name   string
		config config
		want   config
	}{
		{
			"Config Not Found",
			*newConfig(),
			notFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.config.applyOverrides()
			if diff := cmp.Diff(tt.want, tt.config); diff != "" {
				t.Errorf("applyOverrides() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestNewConfi(t *testing.T) {
	// Just setting a couple since we're relying on the same mechanism we're testing.
	want := &config{
		Camera: 0,
	}
	r := reflect.ValueOf(want).Elem()
	for i := 0; i < r.NumField(); i++ {
		field := r.Field(i)
		if field.Kind() == reflect.Bool && field.CanSet() {
			field.SetBool(true)
		}
	}
	want.ConfigFound = false

	var tests = []struct {
		name string
		want *config
	}{
		{
			"New Config",
			want,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := newConfig()
			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Errorf("newConfig() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestMergeConfigs(t *testing.T) {
	camera := *newConfig()
	camera.Camera = 7

	software := *newConfig()
	software.Software = "VNyan"

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
			"Camera Position",
			*newConfig(),
			camera,
			camera,
		},
		{
			"Software",
			*newConfig(),
			software,
			software,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.config.mergeConfigs(tt.new)
			if diff := cmp.Diff(tt.want, tt.config); diff != "" {
				t.Errorf("mergeConfigs() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
