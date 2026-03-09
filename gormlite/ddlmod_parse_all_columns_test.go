package gormlite

import "testing"

func TestParseAllColumns(t *testing.T) {
	tc := []struct {
		name     string
		input    string
		expected []string
	}{
		{
			name:     "Simple case",
			input:    "PRIMARY KEY (column1, column2)",
			expected: []string{"column1", "column2"},
		},
		{
			name:     "Quoted column name",
			input:    "PRIMARY KEY (`column,xxx`, \"column 2\", \"column)3\", 'column''4', \"column\"\"5\")",
			expected: []string{"column,xxx", "column 2", "column)3", "column'4", "column\"5"},
		},
		{
			name:     "Japanese column name",
			input:    "PRIMARY KEY (カラム1, `カラム2`)",
			expected: []string{"カラム1", "カラム2"},
		},
		{
			name:     "Column name quoted with []",
			input:    "PRIMARY KEY ([column1], [column2])",
			expected: []string{"column1", "column2"},
		},
	}
	for _, tt := range tc {
		t.Run(tt.name, func(t *testing.T) {
			cols, err := parseAllColumns(tt.input)
			if err != nil {
				t.Errorf("Failed to parse columns: %s", err)
			}
			if len(cols) != len(tt.expected) {
				t.Errorf("Expected %d columns, got %d", len(tt.expected), len(cols))
			}
			for i, col := range cols {
				if col != tt.expected[i] {
					t.Errorf("Expected %s, got %s", tt.expected[i], col)
				}
			}
		})
	}
}
