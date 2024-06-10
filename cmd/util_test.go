package cmd

import (
	"testing"
)

func TestSlugify(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"Hello World", "hello-world"},
		{"This is a Test", "this-is-a-test"},
		{"Another Test", "another-test"},
		{"Class 12: RL 2 (2024-04-18)", "class-12-rl-2-2024-04-18"},
		{"", ""},
		{"123", "123"},
	}

	for _, test := range tests {
		result := slugify(test.input, "-")
		if result != test.expected {
			t.Errorf("slugify(%q) = %q, expected %q", test.input, result, test.expected)
		}
	}
}

func TestSlugifySep(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"Hello World", "hello_world"},
		{"This is a Test", "this_is_a_test"},
		{"Another Test", "another_test"},
		{"Class 12: RL 2 (2024-04-18)", "class_12_rl_2_2024-04-18"},
		{"", ""},
		{"123", "123"},
	}

	for _, test := range tests {
		result := slugify(test.input, "_")
		if result != test.expected {
			t.Errorf("slugify(%q) = %q, expected %q", test.input, result, test.expected)
		}
	}
}
