package opencc

import (
	"testing"
)

func TestConvertS2T(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "simple conversion",
			input:    "简体字",
			expected: "簡體字",
		},
		{
			name:     "mixed text",
			input:    "这是一个测试",
			expected: "這是一個測試",
		},
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ConvertS2T(tt.input)
			if err != nil {
				t.Fatalf("ConvertS2T() error = %v", err)
			}
			if result != tt.expected {
				t.Errorf("ConvertS2T() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestConvertT2S(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "simple conversion",
			input:    "繁體字",
			expected: "繁体字",
		},
		{
			name:     "mixed text",
			input:    "這是一個測試",
			expected: "这是一个测试",
		},
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ConvertT2S(tt.input)
			if err != nil {
				t.Fatalf("ConvertT2S() error = %v", err)
			}
			if result != tt.expected {
				t.Errorf("ConvertT2S() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestConverter(t *testing.T) {
	converter, err := NewConverter("s2t.json")
	if err != nil {
		t.Fatalf("NewConverter() error = %v", err)
	}
	defer converter.Close()

	result, err := converter.Convert("简体字")
	if err != nil {
		t.Fatalf("Convert() error = %v", err)
	}

	expected := "簡體字"
	if result != expected {
		t.Errorf("Convert() = %v, want %v", result, expected)
	}
}

func BenchmarkConvertS2T(b *testing.B) {
	input := "这是一个很长的测试文本，用来测试转换性能。包含了很多常用的汉字。"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := ConvertS2T(input)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkConverter(b *testing.B) {
	converter, err := NewConverter("s2t.json")
	if err != nil {
		b.Fatal(err)
	}
	defer converter.Close()

	input := "这是一个很长的测试文本，用来测试转换性能。包含了很多常用的汉字。"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := converter.Convert(input)
		if err != nil {
			b.Fatal(err)
		}
	}
}
