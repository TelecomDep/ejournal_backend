package app

import "testing"

func TestShortPersonName(t *testing.T) {
	tests := map[string]string{
		"Добромилов Артём Александрович": "Добромилов А.А.",
		"Иванов Иван":                  "Иванов И.",
		"  Петров   Пётр   Сергеевич ": "Петров П.С.",
		"student-a8f42":                "student-a8f42",
		"":                             "",
	}

	for input, want := range tests {
		if got := shortPersonName(input); got != want {
			t.Errorf("shortPersonName(%q) = %q, want %q", input, got, want)
		}
	}
}
